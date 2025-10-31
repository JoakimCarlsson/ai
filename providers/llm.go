// Package llm provides a unified interface for interacting with various Large Language Model providers.
//
// This package abstracts away the differences between AI providers like OpenAI, Anthropic, Google,
// AWS Bedrock, and others, providing a consistent API for sending messages, streaming responses,
// and handling structured output and tool calling.
//
// The main interface is LLM, which supports both synchronous and streaming interactions,
// with optional structured output and tool calling capabilities. The package handles
// provider-specific authentication, request formatting, and response parsing automatically.
//
// Key features include:
//   - Multi-provider support (OpenAI, Anthropic, Google, AWS Bedrock, Azure, etc.)
//   - Streaming and non-streaming responses
//   - Tool calling with automatic function execution (see package tool)
//   - Structured output with JSON schema validation (see package schema)
//   - Automatic retry logic with exponential backoff
//   - Token usage tracking and cost calculation (see package model)
//   - Provider-specific optimizations and features
//
// Messages are created using the message package, which provides support for text,
// images, and multimodal content. Tools can be implemented using the tool package
// interfaces, and model configurations are available in the model package.
//
// For streaming responses, events are defined in the types package.
//
// Example usage:
//
//	client, err := llm.NewLLM(model.ProviderOpenAI,
//		llm.WithAPIKey("your-api-key"),
//		llm.WithModel(model.OpenAIModels[model.GPT4o]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	messages := []message.Message{
//		message.NewUserMessage("Hello, how are you?"),
//	}
//
//	response, err := client.SendMessages(ctx, messages, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(response.Content)
package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// maxRetries defines the maximum number of retry attempts for failed requests.
const maxRetries = 8

// customProviders stores registered custom provider configurations.
var customProviders = make(map[model.ModelProvider]CustomProviderConfig)
var customProvidersMu sync.RWMutex

// CustomProviderConfig defines configuration for OpenAI-compatible custom providers.
// This enables BYOM (Bring Your Own Model) support for Ollama, local endpoints,
// and custom API providers that implement OpenAI-compatible APIs.
//
// Example usage:
//
//	ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
//	    BaseURL: "http://localhost:11434/v1",
//	    DefaultModel: model.NewCustomModel(
//	        model.WithModelID("llama3.2"),
//	        model.WithAPIModel("llama3.2:latest"),
//	    ),
//	})
//	client, _ := llm.NewLLM(ollama)
type CustomProviderConfig struct {
	// BaseURL is the base URL for the custom provider's API endpoint.
	// For Ollama, this is typically "http://localhost:11434/v1".
	BaseURL string

	// ExtraHeaders contains additional HTTP headers to include in requests.
	// Useful for authentication tokens, tenant IDs, or custom headers.
	ExtraHeaders map[string]string

	// DefaultModel is the model configuration to use when WithModel is not specified.
	// This can be created using model.NewCustomModel().
	DefaultModel model.Model
}

// RegisterCustomProvider registers an OpenAI-compatible custom provider for use with NewLLM.
// Returns a ModelProvider constant that can be passed to NewLLM() to create clients.
//
// Custom providers must implement OpenAI-compatible APIs for message formatting and streaming.
// This works well with Ollama, LocalAI, and other OpenAI-compatible local inference servers.
//
// Example with Ollama:
//
//	llamaModel := model.NewCustomModel(
//	    model.WithModelID("llama3.2"),
//	    model.WithAPIModel("llama3.2:latest"),
//	    model.WithContextWindow(128_000),
//	    model.WithStructuredOutput(true),
//	)
//
//	ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
//	    BaseURL: "http://localhost:11434/v1",
//	    DefaultModel: llamaModel,
//	})
//
//	client, err := llm.NewLLM(ollama,
//	    llm.WithMaxTokens(2000),
//	    llm.WithTemperature(0.7),
//	)
//
// Example with authentication:
//
//	custom := llm.RegisterCustomProvider("my-service", llm.CustomProviderConfig{
//	    BaseURL: "https://my-ai-service.com/v1",
//	    ExtraHeaders: map[string]string{
//	        "X-API-Key": "secret-key",
//	    },
//	    DefaultModel: customModel,
//	})
//
//	client, _ := llm.NewLLM(custom, llm.WithAPIKey("bearer-token"))
func RegisterCustomProvider(name string, config CustomProviderConfig) model.ModelProvider {
	customProvidersMu.Lock()
	defer customProvidersMu.Unlock()

	providerID := model.ModelProvider("custom:" + name)
	customProviders[providerID] = config
	return providerID
}

// getCustomProvider safely retrieves a custom provider configuration.
func getCustomProvider(provider model.ModelProvider) (CustomProviderConfig, bool) {
	customProvidersMu.RLock()
	defer customProvidersMu.RUnlock()

	config, exists := customProviders[provider]
	return config, exists
}

// TokenUsage tracks the number of tokens consumed during an LLM interaction.
type TokenUsage struct {
	// InputTokens is the number of tokens in the input prompt.
	InputTokens int64
	// OutputTokens is the number of tokens generated in the response.
	OutputTokens int64
	// CacheCreationTokens is the number of tokens used to create cache entries.
	CacheCreationTokens int64
	// CacheReadTokens is the number of tokens read from cache.
	CacheReadTokens int64
}

// LLMResponse represents the complete response from an LLM provider.
type LLMResponse struct {
	// Content is the generated text response from the model.
	Content string
	// ToolCalls contains any tool calls requested by the model.
	ToolCalls []message.ToolCall
	// Usage tracks token consumption for this request.
	Usage TokenUsage
	// FinishReason indicates why the model stopped generating.
	FinishReason message.FinishReason
	// StructuredOutput contains JSON-formatted structured output if requested.
	StructuredOutput *string
	// UsedNativeStructuredOutput indicates if the provider's native structured output was used.
	UsedNativeStructuredOutput bool
}

// LLMEvent represents a single event in a streaming LLM response.
type LLMEvent struct {
	// Type indicates the kind of event (content delta, tool call, completion, error, etc.).
	Type types.EventType

	// Content contains text content for content delta events.
	Content string
	// Thinking contains reasoning content for models that support chain-of-thought.
	Thinking string
	// Response contains the final response for completion events.
	Response *LLMResponse
	// ToolCall contains tool call information for tool use events.
	ToolCall *message.ToolCall
	// Error contains error information for error events.
	Error error
}

// LLM defines the interface for interacting with Large Language Model providers.
// It provides methods for both synchronous and streaming interactions, with support
// for tool calling and structured output generation.
type LLM interface {
	// SendMessages sends a conversation to the LLM and returns the complete response.
	// It supports tool calling if tools are provided.
	SendMessages(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) (*LLMResponse, error)

	// SendMessagesWithStructuredOutput sends a conversation and requests structured JSON output
	// conforming to the provided schema. Not all providers support this feature.
	SendMessagesWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) (*LLMResponse, error)

	// StreamResponse sends a conversation and returns a channel of streaming events.
	// Events include content deltas, tool calls, and completion notifications.
	StreamResponse(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) <-chan LLMEvent

	// StreamResponseWithStructuredOutput streams a response with structured output constraints.
	// The final response will include structured JSON conforming to the provided schema.
	StreamResponseWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) <-chan LLMEvent

	// Model returns the model configuration being used by this LLM instance.
	Model() model.Model

	// SupportsStructuredOutput returns true if the provider supports structured output generation.
	SupportsStructuredOutput() bool
}

type llmClientOptions struct {
	apiKey        string
	model         model.Model
	maxTokens     int64
	temperature   *float64
	topP          *float64
	topK          *int64
	stopSequences []string
	timeout       *time.Duration

	anthropicOptions []AnthropicOption
	openaiOptions    []OpenAIOption
	geminiOptions    []GeminiOption
	bedrockOptions   []BedrockOption
	azureOptions     []AzureOption
}

type LLMClientOption func(*llmClientOptions)

type LLMClient interface {
	send(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) (*LLMResponse, error)
	sendWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) (*LLMResponse, error)
	stream(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) <-chan LLMEvent
	streamWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) <-chan LLMEvent
	supportsStructuredOutput() bool
}

type baseLLM[C LLMClient] struct {
	options llmClientOptions
	client  C
}

// NewLLM creates a new LLM client instance for the specified provider with configuration options
func NewLLM(
	llmProvider model.ModelProvider,
	opts ...LLMClientOption,
) (LLM, error) {
	clientOptions := llmClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}
	switch llmProvider {
	case model.ProviderAnthropic:
		return &baseLLM[AnthropicClient]{
			options: clientOptions,
			client:  newAnthropicClient(clientOptions),
		}, nil
	case model.ProviderOpenAI:
		return &baseLLM[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case model.ProviderGemini:
		return &baseLLM[GeminiClient]{
			options: clientOptions,
			client:  newGeminiClient(clientOptions),
		}, nil
	case model.ProviderBedrock:
		return &baseLLM[BedrockClient]{
			options: clientOptions,
			client:  newBedrockClient(clientOptions),
		}, nil
	case model.ProviderGROQ:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.groq.com/openai/v1"),
		)
		return &baseLLM[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case model.ProviderAzure:
		return &baseLLM[AzureClient]{
			options: clientOptions,
			client:  newAzureClient(clientOptions),
		}, nil
	case model.ProviderVertexAI:
		return &baseLLM[VertexAIClient]{
			options: clientOptions,
			client:  newVertexAIClient(clientOptions),
		}, nil
	case model.ProviderOpenRouter:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://openrouter.ai/api/v1"),
			WithOpenAIExtraHeaders(map[string]string{
				"HTTP-Referer": "go-llm.com",
				"X-Title":      "GoLLM",
			}),
		)
		return &baseLLM[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case model.ProviderXAI:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.x.ai/v1"),
		)
		return &baseLLM[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	}

	if config, exists := getCustomProvider(llmProvider); exists {
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(config.BaseURL),
		)
		if config.ExtraHeaders != nil {
			clientOptions.openaiOptions = append(clientOptions.openaiOptions,
				WithOpenAIExtraHeaders(config.ExtraHeaders),
			)
		}
		if clientOptions.model.ID == "" && config.DefaultModel.ID != "" {
			clientOptions.model = config.DefaultModel
		}
		return &baseLLM[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf("llm provider not supported: %s", llmProvider)
}

func (p *baseLLM[C]) cleanMessages(
	messages []message.Message,
) (cleaned []message.Message) {
	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (p *baseLLM[C]) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*LLMResponse, error) {
	messages = p.cleanMessages(messages)
	response, err := p.client.send(ctx, messages, tools)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *baseLLM[C]) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*LLMResponse, error) {
	if !p.client.supportsStructuredOutput() {
		return nil, fmt.Errorf(
			"structured output not supported by provider: %s",
			p.options.model.Provider,
		)
	}

	messages = p.cleanMessages(messages)
	response, err := p.client.sendWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *baseLLM[C]) Model() model.Model {
	return p.options.model
}

func (p *baseLLM[C]) SupportsStructuredOutput() bool {
	return p.client.supportsStructuredOutput()
}

func (p *baseLLM[C]) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan LLMEvent {
	messages = p.cleanMessages(messages)
	return p.client.stream(ctx, messages, tools)
}

func (p *baseLLM[C]) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan LLMEvent {
	if !p.client.supportsStructuredOutput() {
		errChan := make(chan LLMEvent, 1)
		errChan <- LLMEvent{
			Type:  types.EventError,
			Error: fmt.Errorf("structured output not supported by provider: %s", p.options.model.Provider),
		}
		close(errChan)
		return errChan
	}

	messages = p.cleanMessages(messages)
	return p.client.streamWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
}

// WithAPIKey sets the API key for authenticating with the LLM provider
func WithAPIKey(apiKey string) LLMClientOption {
	return func(options *llmClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which model to use for generating responses
func WithModel(model model.Model) LLMClientOption {
	return func(options *llmClientOptions) {
		options.model = model
	}
}

// WithMaxTokens sets the maximum number of tokens to generate in responses
func WithMaxTokens(maxTokens int64) LLMClientOption {
	return func(options *llmClientOptions) {
		options.maxTokens = maxTokens
	}
}

// WithAnthropicOptions applies provider-specific configuration for Anthropic models
func WithAnthropicOptions(anthropicOptions ...AnthropicOption) LLMClientOption {
	return func(options *llmClientOptions) {
		options.anthropicOptions = anthropicOptions
	}
}

// WithOpenAIOptions applies provider-specific configuration for OpenAI models
func WithOpenAIOptions(openaiOptions ...OpenAIOption) LLMClientOption {
	return func(options *llmClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

// WithGeminiOptions applies provider-specific configuration for Gemini models
func WithGeminiOptions(geminiOptions ...GeminiOption) LLMClientOption {
	return func(options *llmClientOptions) {
		options.geminiOptions = geminiOptions
	}
}

// WithBedrockOptions applies provider-specific configuration for AWS Bedrock models
func WithBedrockOptions(bedrockOptions ...BedrockOption) LLMClientOption {
	return func(options *llmClientOptions) {
		options.bedrockOptions = bedrockOptions
	}
}

// WithAzureOptions applies provider-specific configuration for Azure OpenAI models
func WithAzureOptions(azureOptions ...AzureOption) LLMClientOption {
	return func(options *llmClientOptions) {
		options.azureOptions = azureOptions
	}
}

// WithTemperature controls the randomness of responses, from 0 (deterministic) to 1 (creative)
func WithTemperature(temperature float64) LLMClientOption {
	return func(options *llmClientOptions) {
		options.temperature = &temperature
	}
}

// WithTopP sets nucleus sampling to control response diversity by probability mass
func WithTopP(topP float64) LLMClientOption {
	return func(options *llmClientOptions) {
		options.topP = &topP
	}
}

// WithTopK limits token selection to the top K most likely candidates
func WithTopK(topK int64) LLMClientOption {
	return func(options *llmClientOptions) {
		options.topK = &topK
	}
}

// WithStopSequences defines text sequences that will halt response generation
func WithStopSequences(stopSequences ...string) LLMClientOption {
	return func(options *llmClientOptions) {
		options.stopSequences = stopSequences
	}
}

// WithTimeout sets the maximum duration to wait for API responses
func WithTimeout(timeout time.Duration) LLMClientOption {
	return func(options *llmClientOptions) {
		options.timeout = &timeout
	}
}
