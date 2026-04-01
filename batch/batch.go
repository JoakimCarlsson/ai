// Package batch provides async batch processing for sending bulk LLM, embedding,
// and other API requests efficiently using provider batch APIs where available,
// with a fallback concurrent execution strategy.
package batch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

// ErrNoLLMClient is returned when a chat request is submitted without an LLM client.
var ErrNoLLMClient = errors.New("batch: no LLM client configured")

// ErrNoEmbeddingClient is returned when an embedding request is submitted without an embedding client.
var ErrNoEmbeddingClient = errors.New(
	"batch: no embedding client configured",
)

// RequestType identifies whether a batch request is a chat completion or embedding.
type RequestType int

// Request types.
const (
	RequestTypeChat RequestType = iota
	RequestTypeEmbedding
)

// Request represents a single item in a batch.
type Request struct {
	ID       string
	Type     RequestType
	Messages []message.Message
	Tools    []tool.BaseTool
	Texts    []string
}

// Result holds the outcome of a single batch request.
type Result struct {
	ID            string
	Index         int
	ChatResponse  *llm.Response
	EmbedResponse *embeddings.EmbeddingResponse
	Err           error
}

// Response contains the aggregated results of a batch operation.
type Response struct {
	Results   []Result
	Completed int
	Failed    int
	Total     int
}

// Processor submits and manages batch requests.
type Processor interface {
	Process(ctx context.Context, requests []Request) (*Response, error)
	ProcessAsync(ctx context.Context, requests []Request) (<-chan Event, error)
}

type batchClient interface {
	executeBatch(
		ctx context.Context,
		requests []Request,
		opts clientOptions,
	) (*Response, error)
	executeBatchAsync(
		ctx context.Context,
		requests []Request,
		opts clientOptions,
	) (<-chan Event, error)
}

type clientOptions struct {
	apiKey           string
	model            model.Model
	embeddingModel   model.EmbeddingModel
	maxTokens        int64
	maxConcurrency   int
	progressCallback ProgressCallback
	pollInterval     time.Duration
	timeout          *time.Duration

	llmClient       llm.LLM
	embeddingClient embeddings.Embedding

	openaiOptions    []OpenAIOption
	anthropicOptions []AnthropicOption
	geminiOptions    []GeminiOption
}

// Option configures a batch Processor.
type Option func(*clientOptions)

// WithAPIKey sets the API key for authentication with the batch provider.
func WithAPIKey(apiKey string) Option {
	return func(o *clientOptions) {
		o.apiKey = apiKey
	}
}

// WithModel sets the LLM model for chat completion batch requests.
func WithModel(m model.Model) Option {
	return func(o *clientOptions) {
		o.model = m
	}
}

// WithEmbeddingModel sets the embedding model for embedding batch requests.
func WithEmbeddingModel(m model.EmbeddingModel) Option {
	return func(o *clientOptions) {
		o.embeddingModel = m
	}
}

// WithMaxTokens sets the maximum number of tokens to generate per request.
func WithMaxTokens(maxTokens int64) Option {
	return func(o *clientOptions) {
		o.maxTokens = maxTokens
	}
}

// WithMaxConcurrency sets the maximum number of concurrent requests for the fallback executor.
func WithMaxConcurrency(n int) Option {
	return func(o *clientOptions) {
		o.maxConcurrency = n
	}
}

// WithProgressCallback sets a callback invoked with progress updates during batch processing.
func WithProgressCallback(fn ProgressCallback) Option {
	return func(o *clientOptions) {
		o.progressCallback = fn
	}
}

// WithPollInterval sets the polling interval for native batch APIs.
func WithPollInterval(d time.Duration) Option {
	return func(o *clientOptions) {
		o.pollInterval = d
	}
}

// WithTimeout sets the maximum duration for batch requests.
func WithTimeout(timeout time.Duration) Option {
	return func(o *clientOptions) {
		o.timeout = &timeout
	}
}

// WithLLM sets an existing LLM client for concurrent fallback mode.
func WithLLM(client llm.LLM) Option {
	return func(o *clientOptions) {
		o.llmClient = client
	}
}

// WithEmbedding sets an existing embedding client for concurrent fallback mode.
func WithEmbedding(client embeddings.Embedding) Option {
	return func(o *clientOptions) {
		o.embeddingClient = client
	}
}

// WithOpenAIOptions applies OpenAI-specific configuration options.
func WithOpenAIOptions(opts ...OpenAIOption) Option {
	return func(o *clientOptions) {
		o.openaiOptions = opts
	}
}

// WithAnthropicOptions applies Anthropic-specific configuration options.
func WithAnthropicOptions(opts ...AnthropicOption) Option {
	return func(o *clientOptions) {
		o.anthropicOptions = opts
	}
}

// WithGeminiOptions applies Gemini-specific configuration options.
func WithGeminiOptions(opts ...GeminiOption) Option {
	return func(o *clientOptions) {
		o.geminiOptions = opts
	}
}

type baseProcessor[C batchClient] struct {
	options clientOptions
	client  C
}

// New creates a batch Processor for the specified provider.
func New(
	provider model.Provider,
	opts ...Option,
) (Processor, error) {
	options := clientOptions{
		maxConcurrency: 10,
		pollInterval:   30 * time.Second,
		maxTokens:      4096,
	}
	for _, o := range opts {
		o(&options)
	}

	switch provider {
	case model.ProviderOpenAI:
		return &baseProcessor[*openaiClient]{
			options: options,
			client:  newOpenAIBatchClient(options),
		}, nil
	case model.ProviderAnthropic:
		return &baseProcessor[*anthropicClient]{
			options: options,
			client:  newAnthropicBatchClient(options),
		}, nil
	case model.ProviderGemini:
		return &baseProcessor[*geminiBatchClient]{
			options: options,
			client:  newGeminiBatchClient(options),
		}, nil
	default:
		return &baseProcessor[*concurrentClient]{
			options: options,
			client:  &concurrentClient{},
		}, nil
	}
}

func (p *baseProcessor[C]) Process(
	ctx context.Context,
	requests []Request,
) (*Response, error) {
	if len(requests) == 0 {
		return &Response{Results: []Result{}, Total: 0}, nil
	}

	for i := range requests {
		if requests[i].ID == "" {
			requests[i].ID = fmt.Sprintf("req_%d", i)
		}
	}

	return p.client.executeBatch(ctx, requests, p.options)
}

func (p *baseProcessor[C]) ProcessAsync(
	ctx context.Context,
	requests []Request,
) (<-chan Event, error) {
	if len(requests) == 0 {
		ch := make(chan Event, 1)
		ch <- Event{
			Type:     EventComplete,
			Progress: &Progress{Total: 0, Status: "complete"},
		}
		close(ch)
		return ch, nil
	}

	for i := range requests {
		if requests[i].ID == "" {
			requests[i].ID = fmt.Sprintf("req_%d", i)
		}
	}

	return p.client.executeBatchAsync(ctx, requests, p.options)
}
