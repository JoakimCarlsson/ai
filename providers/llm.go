package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

const maxRetries = 8

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type LLMResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	Usage        TokenUsage
	FinishReason message.FinishReason
}

type LLMEvent struct {
	Type types.EventType

	Content  string
	Thinking string
	Response *LLMResponse
	ToolCall *message.ToolCall
	Error    error
}

type LLM interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (*LLMResponse, error)

	StreamResponse(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent

	Model() model.Model
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
}

type LLMClientOption func(*llmClientOptions)

type LLMClient interface {
	send(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (*LLMResponse, error)
	stream(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent
}

type baseLLM[C LLMClient] struct {
	options llmClientOptions
	client  C
}

// NewLLM creates a new LLM client instance for the specified provider with configuration options
func NewLLM(llmProvider model.ModelProvider, opts ...LLMClientOption) (LLM, error) {
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

	return nil, fmt.Errorf("llm provider not supported: %s", llmProvider)
}

func (p *baseLLM[C]) cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (p *baseLLM[C]) SendMessages(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (*LLMResponse, error) {
	messages = p.cleanMessages(messages)
	response, err := p.client.send(ctx, messages, tools)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *baseLLM[C]) Model() model.Model {
	return p.options.model
}

func (p *baseLLM[C]) StreamResponse(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent {
	messages = p.cleanMessages(messages)
	return p.client.stream(ctx, messages, tools)
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
