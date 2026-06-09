// Package vertexai provides a Google Vertex AI implementation of the [llm.LLM]
// interface. It reuses the request/response logic from [llm/gemini] with a
// Vertex-AI-backed [genai.Client].
package vertexai

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
	"github.com/joakimcarlsson/ai/model"
	"google.golang.org/genai"
)

// Options configures the Vertex AI LLM client.
type Options struct {
	model         model.Model
	maxTokens     int64
	temperature   *float64
	topP          *float64
	topK          *int64
	stopSequences []string
	timeout       *time.Duration
	thinkingLevel *llmgemini.ThinkingLevel
	project       string
	location      string
	httpClient    *http.Client
}

// Option configures Options.
type Option func(*Options)

// WithModel selects the LLM model.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithMaxTokens sets the max generation tokens.
func WithMaxTokens(
	maxTokens int64,
) Option {
	return func(o *Options) { o.maxTokens = maxTokens }
}

// WithTemperature controls randomness.
func WithTemperature(
	t float64,
) Option {
	return func(o *Options) { o.temperature = &t }
}

// WithTopP sets nucleus sampling probability mass.
func WithTopP(p float64) Option { return func(o *Options) { o.topP = &p } }

// WithTopK limits token selection to the top K candidates.
func WithTopK(k int64) Option { return func(o *Options) { o.topK = &k } }

// WithStopSequences sets text sequences that halt generation.
func WithStopSequences(
	seqs ...string,
) Option {
	return func(o *Options) { o.stopSequences = seqs }
}

// WithTimeout sets the maximum duration to wait for API responses.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithThinkingLevel sets the thinking level for Gemini models that support reasoning.
func WithThinkingLevel(level llmgemini.ThinkingLevel) Option {
	return func(o *Options) { o.thinkingLevel = &level }
}

// WithHTTPClient injects a custom *http.Client, set on the genai ClientConfig's
// HTTPClient field. Use it for outbound proxies, custom TLS (private CAs, mTLS),
// connection-pool tuning, or transport-level instrumentation. A nil client is a
// no-op, leaving the SDK default client in place. The per-request context
// timeout from WithTimeout still applies on top of the injected client's
// transport: the two compose and the shorter deadline wins.
func WithHTTPClient(c *http.Client) Option {
	return func(o *Options) { o.httpClient = c }
}

// WithProject sets the GCP project ID. Defaults to $VERTEXAI_PROJECT.
func WithProject(
	project string,
) Option {
	return func(o *Options) { o.project = project }
}

// WithLocation sets the GCP location. Defaults to $VERTEXAI_LOCATION.
func WithLocation(
	location string,
) Option {
	return func(o *Options) { o.location = location }
}

// Client implements [llm.LLM] against Vertex AI by embedding [llm/gemini].Client
// constructed with a Vertex-AI-backed [genai.Client].
type Client struct {
	*llmgemini.Client
}

// NewLLM constructs a Vertex AI LLM client.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	project := options.project
	if project == "" {
		project = os.Getenv("VERTEXAI_PROJECT")
	}
	location := options.location
	if location == "" {
		location = os.Getenv("VERTEXAI_LOCATION")
	}

	cfg := &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	}
	if options.httpClient != nil {
		cfg.HTTPClient = options.httpClient
	}
	client, err := genai.NewClient(context.Background(), cfg)
	if err != nil {
		// Match the previous nil-on-error behavior; tracing wrapper handles nil safely
		// because Model() panics only on use, which mirrors the original code path.
		client = nil
	}

	gemOpts := buildGeminiOptions(options)
	bare := llmgemini.NewWithExistingClient(gemOpts, client)
	return llm.WithTracing(&Client{Client: bare}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

func buildGeminiOptions(o Options) llmgemini.Options {
	var dst llmgemini.Options
	llmgemini.WithModel(o.model)(&dst)
	llmgemini.WithMaxTokens(o.maxTokens)(&dst)
	if o.temperature != nil {
		llmgemini.WithTemperature(*o.temperature)(&dst)
	}
	if o.topP != nil {
		llmgemini.WithTopP(*o.topP)(&dst)
	}
	if o.topK != nil {
		llmgemini.WithTopK(*o.topK)(&dst)
	}
	if len(o.stopSequences) > 0 {
		llmgemini.WithStopSequences(o.stopSequences...)(&dst)
	}
	if o.timeout != nil {
		llmgemini.WithTimeout(*o.timeout)(&dst)
	}
	if o.thinkingLevel != nil {
		llmgemini.WithThinkingLevel(*o.thinkingLevel)(&dst)
	}
	return dst
}
