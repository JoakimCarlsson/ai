// Package batch provides async batch processing for sending bulk LLM, embedding,
// and other API requests efficiently using provider batch APIs where available,
// with a fallback concurrent execution strategy.
package batch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/openai/openai-go"
	"google.golang.org/genai"
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

type executor interface {
	execute(
		ctx context.Context,
		requests []Request,
		opts processorOptions,
	) (*Response, error)
	executeAsync(
		ctx context.Context,
		requests []Request,
		opts processorOptions,
	) (<-chan Event, error)
}

type processorOptions struct {
	llmClient        llm.LLM
	embeddingClient  embeddings.Embedding
	maxConcurrency   int
	progressCallback ProgressCallback
	pollInterval     time.Duration

	openaiClient    *openai.Client
	anthropicClient *anthropic.Client
	geminiClient    *genai.Client
	geminiModel     string
}

// Option configures a Processor.
type Option func(*processorOptions)

// WithLLM sets the LLM client used for chat completion requests in concurrent mode.
func WithLLM(client llm.LLM) Option {
	return func(o *processorOptions) {
		o.llmClient = client
	}
}

// WithEmbedding sets the embedding client used for embedding requests in concurrent mode.
func WithEmbedding(client embeddings.Embedding) Option {
	return func(o *processorOptions) {
		o.embeddingClient = client
	}
}

// WithMaxConcurrency sets the maximum number of concurrent requests for the fallback executor.
func WithMaxConcurrency(n int) Option {
	return func(o *processorOptions) {
		o.maxConcurrency = n
	}
}

// WithProgressCallback sets a callback invoked with progress updates during batch processing.
func WithProgressCallback(fn ProgressCallback) Option {
	return func(o *processorOptions) {
		o.progressCallback = fn
	}
}

// WithPollInterval sets the polling interval for native batch APIs.
func WithPollInterval(d time.Duration) Option {
	return func(o *processorOptions) {
		o.pollInterval = d
	}
}

// WithOpenAIClient enables the OpenAI native Batch API executor.
func WithOpenAIClient(client openai.Client) Option {
	return func(o *processorOptions) {
		o.openaiClient = &client
	}
}

// WithAnthropicClient enables the Anthropic Message Batches API executor.
func WithAnthropicClient(client anthropic.Client) Option {
	return func(o *processorOptions) {
		o.anthropicClient = &client
	}
}

// WithGeminiClient enables the Gemini/Vertex AI batch executor.
func WithGeminiClient(client *genai.Client, model string) Option {
	return func(o *processorOptions) {
		o.geminiClient = client
		o.geminiModel = model
	}
}

// Processor submits and manages batch requests.
type Processor struct {
	options  processorOptions
	executor executor
}

// New creates a Processor with the given options.
func New(opts ...Option) *Processor {
	options := processorOptions{
		maxConcurrency: 10,
		pollInterval:   30 * time.Second,
	}
	for _, o := range opts {
		o(&options)
	}

	p := &Processor{options: options}
	p.executor = p.selectExecutor()
	return p
}

func (p *Processor) selectExecutor() executor {
	switch {
	case p.options.openaiClient != nil:
		return &openaiNativeExecutor{client: p.options.openaiClient}
	case p.options.anthropicClient != nil:
		return &anthropicNativeExecutor{
			client: p.options.anthropicClient,
		}
	case p.options.geminiClient != nil:
		return &geminiNativeExecutor{
			client: p.options.geminiClient,
			model:  p.options.geminiModel,
		}
	default:
		return &concurrentExecutor{}
	}
}

// Process submits all requests and blocks until the batch completes.
func (p *Processor) Process(
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

	return p.executor.execute(ctx, requests, p.options)
}

// ProcessAsync submits all requests and returns a channel of Events for progress tracking.
func (p *Processor) ProcessAsync(
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

	return p.executor.executeAsync(ctx, requests, p.options)
}
