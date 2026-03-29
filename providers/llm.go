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
	"github.com/joakimcarlsson/ai/tracing"
	"github.com/joakimcarlsson/ai/types"
)

// maxRetries defines the maximum number of retry attempts for failed requests.
const maxRetries = 8

// customProviders stores registered custom provider configurations.
var customProviders = make(map[model.Provider]CustomProviderConfig)
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
// Returns a Provider constant that can be passed to NewLLM() to create clients.
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
func RegisterCustomProvider(
	name string,
	config CustomProviderConfig,
) model.Provider {
	customProvidersMu.Lock()
	defer customProvidersMu.Unlock()

	providerID := model.Provider("custom:" + name)
	customProviders[providerID] = config
	return providerID
}

// getCustomProvider safely retrieves a custom provider configuration.
func getCustomProvider(
	provider model.Provider,
) (CustomProviderConfig, bool) {
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

// Add accumulates token counts from another TokenUsage into this one.
// This is used to aggregate usage across multiple LLM calls in an agent loop.
func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.CacheCreationTokens += other.CacheCreationTokens
	u.CacheReadTokens += other.CacheReadTokens
}

// Response represents the complete response from an LLM provider.
type Response struct {
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

// Event represents a single event in a streaming LLM response.
type Event struct {
	// Type indicates the kind of event (content delta, tool call, completion, error, etc.).
	Type types.EventType

	// Content contains text content for content delta events.
	Content string
	// Thinking contains reasoning content for models that support chain-of-thought.
	Thinking string
	// Response contains the final response for completion events.
	Response *Response
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
	) (*Response, error)

	// SendMessagesWithStructuredOutput sends a conversation and requests structured JSON output
	// conforming to the provided schema. Not all providers support this feature.
	SendMessagesWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) (*Response, error)

	// StreamResponse sends a conversation and returns a channel of streaming events.
	// Events include content deltas, tool calls, and completion notifications.
	StreamResponse(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) <-chan Event

	// StreamResponseWithStructuredOutput streams a response with structured output constraints.
	// The final response will include structured JSON conforming to the provided schema.
	StreamResponseWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) <-chan Event

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

// ClientOption configures an LLM client when passed to NewLLM.
type ClientOption func(*llmClientOptions)

// Client is the provider-specific implementation used by baseLLM.
type Client interface {
	send(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) (*Response, error)
	sendWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) (*Response, error)
	stream(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) <-chan Event
	streamWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) <-chan Event
	supportsStructuredOutput() bool
}

type baseLLM[C Client] struct {
	options llmClientOptions
	client  C
}

// NewLLM creates a new LLM client instance for the specified provider with configuration options
func NewLLM(
	llmProvider model.Provider,
	opts ...ClientOption,
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
	case model.ProviderMistral:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.mistral.ai/v1"),
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

func (p *baseLLM[C]) generateSpanAttrs() []tracing.Attr {
	attrs := []tracing.Attr{
		tracing.AttrRequestMaxTokens.Int64(p.options.maxTokens),
	}
	if p.options.temperature != nil {
		attrs = append(
			attrs,
			tracing.AttrRequestTemperature.Float64(*p.options.temperature),
		)
	}
	if p.options.topP != nil {
		attrs = append(attrs, tracing.AttrRequestTopP.Float64(*p.options.topP))
	}
	return attrs
}

func (p *baseLLM[C]) recordResponseAttrs(
	span tracing.Span,
	resp *Response,
	toolCount int,
) {
	attrs := []tracing.Attr{
		tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
		tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
		tracing.AttrResponseFinishReason.String(string(resp.FinishReason)),
	}
	if resp.Usage.CacheCreationTokens > 0 {
		attrs = append(
			attrs,
			tracing.AttrUsageCacheCreation.Int64(
				resp.Usage.CacheCreationTokens,
			),
		)
	}
	if resp.Usage.CacheReadTokens > 0 {
		attrs = append(
			attrs,
			tracing.AttrUsageCacheRead.Int64(resp.Usage.CacheReadTokens),
		)
	}
	if len(resp.ToolCalls) > 0 {
		attrs = append(
			attrs,
			tracing.AttrToolCallCount.Int(len(resp.ToolCalls)),
		)
	}
	if toolCount > 0 {
		attrs = append(attrs, tracing.AttrToolCount.Int(toolCount))
	}
	tracing.SetResponseAttrs(span, attrs...)
}

func (p *baseLLM[C]) logMessages(
	ctx context.Context,
	messages []message.Message,
	resp *Response,
) {
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			tracing.LogSystemMessage(ctx, messageText(msg))
		case message.User:
			tracing.LogUserMessage(ctx, messageText(msg))
		}
	}
	if resp != nil {
		tracing.LogChoice(ctx, resp.Content, string(resp.FinishReason))
	}
}

func messageText(msg message.Message) string {
	return msg.Content().Text
}

func (p *baseLLM[C]) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*Response, error) {
	messages = p.cleanMessages(messages)

	ctx, span := tracing.StartGenerateSpan(
		ctx,
		p.options.model.APIModel,
		string(p.options.model.Provider),
		p.generateSpanAttrs()...,
	)
	defer span.End()

	response, err := p.client.send(ctx, messages, tools)
	if err != nil {
		tracing.SetError(span, err)
		return nil, err
	}

	p.recordResponseAttrs(span, response, len(tools))
	p.logMessages(ctx, messages, response)

	return response, nil
}

func (p *baseLLM[C]) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*Response, error) {
	if !p.client.supportsStructuredOutput() {
		return nil, fmt.Errorf(
			"structured output not supported by provider: %s",
			p.options.model.Provider,
		)
	}

	messages = p.cleanMessages(messages)

	ctx, span := tracing.StartGenerateSpan(
		ctx,
		p.options.model.APIModel,
		string(p.options.model.Provider),
		p.generateSpanAttrs()...,
	)
	defer span.End()

	response, err := p.client.sendWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
	if err != nil {
		tracing.SetError(span, err)
		return nil, err
	}

	p.recordResponseAttrs(span, response, len(tools))
	p.logMessages(ctx, messages, response)

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
) <-chan Event {
	messages = p.cleanMessages(messages)

	ctx, span := tracing.StartGenerateSpan(
		ctx,
		p.options.model.APIModel,
		string(p.options.model.Provider),
		p.generateSpanAttrs()...,
	)

	innerCh := p.client.stream(ctx, messages, tools)
	outCh := make(chan Event)
	go func() {
		defer close(outCh)
		defer span.End()
		for evt := range innerCh {
			if evt.Type == types.EventComplete && evt.Response != nil {
				p.recordResponseAttrs(span, evt.Response, len(tools))
				tracing.LogChoice(
					ctx,
					evt.Response.Content,
					string(evt.Response.FinishReason),
				)
			}
			if evt.Type == types.EventError && evt.Error != nil {
				tracing.SetError(span, evt.Error)
			}
			outCh <- evt
		}
	}()
	return outCh
}

func (p *baseLLM[C]) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan Event {
	if !p.client.supportsStructuredOutput() {
		errChan := make(chan Event, 1)
		errChan <- Event{
			Type:  types.EventError,
			Error: fmt.Errorf("structured output not supported by provider: %s", p.options.model.Provider),
		}
		close(errChan)
		return errChan
	}

	messages = p.cleanMessages(messages)

	ctx, span := tracing.StartGenerateSpan(
		ctx,
		p.options.model.APIModel,
		string(p.options.model.Provider),
		p.generateSpanAttrs()...,
	)

	innerCh := p.client.streamWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
	outCh := make(chan Event)
	go func() {
		defer close(outCh)
		defer span.End()
		for evt := range innerCh {
			if evt.Type == types.EventComplete && evt.Response != nil {
				p.recordResponseAttrs(span, evt.Response, len(tools))
				tracing.LogChoice(
					ctx,
					evt.Response.Content,
					string(evt.Response.FinishReason),
				)
			}
			if evt.Type == types.EventError && evt.Error != nil {
				tracing.SetError(span, evt.Error)
			}
			outCh <- evt
		}
	}()
	return outCh
}

// WithAPIKey sets the API key for authenticating with the LLM provider
func WithAPIKey(apiKey string) ClientOption {
	return func(options *llmClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which model to use for generating responses
func WithModel(model model.Model) ClientOption {
	return func(options *llmClientOptions) {
		options.model = model
	}
}

// WithMaxTokens sets the maximum number of tokens to generate in responses
func WithMaxTokens(maxTokens int64) ClientOption {
	return func(options *llmClientOptions) {
		options.maxTokens = maxTokens
	}
}

// WithAnthropicOptions applies provider-specific configuration for Anthropic models
func WithAnthropicOptions(anthropicOptions ...AnthropicOption) ClientOption {
	return func(options *llmClientOptions) {
		options.anthropicOptions = anthropicOptions
	}
}

// WithOpenAIOptions applies provider-specific configuration for OpenAI models
func WithOpenAIOptions(openaiOptions ...OpenAIOption) ClientOption {
	return func(options *llmClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

// WithGeminiOptions applies provider-specific configuration for Gemini models
func WithGeminiOptions(geminiOptions ...GeminiOption) ClientOption {
	return func(options *llmClientOptions) {
		options.geminiOptions = geminiOptions
	}
}

// WithBedrockOptions applies provider-specific configuration for AWS Bedrock models
func WithBedrockOptions(bedrockOptions ...BedrockOption) ClientOption {
	return func(options *llmClientOptions) {
		options.bedrockOptions = bedrockOptions
	}
}

// WithAzureOptions applies provider-specific configuration for Azure OpenAI models
func WithAzureOptions(azureOptions ...AzureOption) ClientOption {
	return func(options *llmClientOptions) {
		options.azureOptions = azureOptions
	}
}

// WithTemperature controls the randomness of responses, from 0 (deterministic) to 1 (creative)
func WithTemperature(temperature float64) ClientOption {
	return func(options *llmClientOptions) {
		options.temperature = &temperature
	}
}

// WithTopP sets nucleus sampling to control response diversity by probability mass
func WithTopP(topP float64) ClientOption {
	return func(options *llmClientOptions) {
		options.topP = &topP
	}
}

// WithTopK limits token selection to the top K most likely candidates
func WithTopK(topK int64) ClientOption {
	return func(options *llmClientOptions) {
		options.topK = &topK
	}
}

// WithStopSequences defines text sequences that will halt response generation
func WithStopSequences(stopSequences ...string) ClientOption {
	return func(options *llmClientOptions) {
		options.stopSequences = stopSequences
	}
}

// WithTimeout sets the maximum duration to wait for API responses
func WithTimeout(timeout time.Duration) ClientOption {
	return func(options *llmClientOptions) {
		options.timeout = &timeout
	}
}
