// Package bedrock provides an AWS Bedrock implementation of the [llm.LLM] interface.
//
// Bedrock hosts third-party foundation models; this package detects the model
// vendor by API ID and constructs the appropriate underlying client. Currently
// only Anthropic-on-Bedrock is wired up — the package delegates to
// [llm/anthropic] with the Bedrock backend enabled.
package bedrock

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// Options configures the Bedrock LLM client.
type Options struct {
	apiKey        string
	model         model.Model
	maxTokens     int64
	temperature   *float64
	topP          *float64
	topK          *int64
	stopSequences []string
	timeout       *time.Duration
	disableCache  bool
	httpClient    *http.Client
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key (passed through to the underlying vendor client where applicable).
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

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

// WithHTTPClient injects a custom *http.Client, passed through to the
// underlying Anthropic-on-Bedrock client. Use it for outbound proxies, custom
// TLS (private CAs, mTLS), connection-pool tuning, or transport-level
// instrumentation. It composes with Bedrock's AWS SigV4 signing, which the SDK
// applies as request middleware on top of the injected client's transport. A
// nil client is a no-op, leaving the SDK default client in place. The
// per-request context timeout from WithTimeout still applies on top of the
// injected client's transport: the two compose and the shorter deadline wins.
func WithHTTPClient(c *http.Client) Option {
	return func(o *Options) { o.httpClient = c }
}

// WithDisableCache disables Anthropic prompt caching. Caching is enabled by
// default; when enabled, cache_control breakpoints emitted by the underlying
// Anthropic client reach Bedrock and produce cacheRead/cacheWrite token usage.
func WithDisableCache() Option { return func(o *Options) { o.disableCache = true } }

// Client implements [llm.LLM] against AWS Bedrock by delegating to a child
// vendor client (Anthropic for Claude on Bedrock).
type Client struct {
	options Options
	child   llm.LLM
}

// NewLLM constructs a Bedrock LLM client. The Bedrock model APIModel field is
// used to detect which underlying vendor to use; AWS_REGION (or
// AWS_DEFAULT_REGION) is read from the environment to derive the regional
// model ID prefix that Bedrock requires.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}

	if len(region) < 2 {
		return llm.WithTracing(&Client{options: options}, llm.TracingAttrs{
			MaxTokens:   options.maxTokens,
			Temperature: options.temperature,
			TopP:        options.topP,
		})
	}

	regionPrefix := region[:2]
	options.model.APIModel = fmt.Sprintf(
		"%s.%s",
		regionPrefix,
		options.model.APIModel,
	)

	c := &Client{options: options}
	if strings.Contains(options.model.APIModel, "anthropic") {
		c.child = newAnthropicChild(options)
	}
	return llm.WithTracing(c, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

func newAnthropicChild(options Options) llm.LLM {
	anthOpts := []llmanthropic.Option{
		llmanthropic.WithModel(options.model),
		llmanthropic.WithMaxTokens(options.maxTokens),
		llmanthropic.WithBedrock(true),
	}
	if options.disableCache {
		anthOpts = append(anthOpts, llmanthropic.WithDisableCache())
	}
	if options.apiKey != "" {
		anthOpts = append(anthOpts, llmanthropic.WithAPIKey(options.apiKey))
	}
	if options.temperature != nil {
		anthOpts = append(
			anthOpts,
			llmanthropic.WithTemperature(*options.temperature),
		)
	}
	if options.topP != nil {
		anthOpts = append(anthOpts, llmanthropic.WithTopP(*options.topP))
	}
	if options.topK != nil {
		anthOpts = append(anthOpts, llmanthropic.WithTopK(*options.topK))
	}
	if len(options.stopSequences) > 0 {
		anthOpts = append(
			anthOpts,
			llmanthropic.WithStopSequences(options.stopSequences...),
		)
	}
	if options.timeout != nil {
		anthOpts = append(anthOpts, llmanthropic.WithTimeout(*options.timeout))
	}
	if options.httpClient != nil {
		anthOpts = append(
			anthOpts,
			llmanthropic.WithHTTPClient(options.httpClient),
		)
	}
	return llmanthropic.NewLLM(anthOpts...)
}

// Model returns the configured LLM model.
func (c *Client) Model() model.Model { return c.options.model }

// SupportsStructuredOutput reports whether the underlying child supports it.
func (c *Client) SupportsStructuredOutput() bool {
	if c.child != nil {
		return c.child.SupportsStructuredOutput()
	}
	return false
}

// SendMessages delegates to the child vendor client.
func (c *Client) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	if c.child == nil {
		return nil, errors.New("unsupported model for bedrock provider")
	}
	return c.child.SendMessages(ctx, messages, tools)
}

// StreamResponse delegates to the child vendor client.
func (c *Client) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	if c.child == nil {
		eventChan := make(chan llm.Event, 1)
		eventChan <- llm.Event{
			Type:  types.EventError,
			Error: errors.New("unsupported model for bedrock provider"),
		}
		close(eventChan)
		return eventChan
	}
	return c.child.StreamResponse(ctx, messages, tools)
}

// SendMessagesWithStructuredOutput delegates to the child vendor client.
func (c *Client) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	if c.child == nil {
		return nil, errors.New(
			"structured output not supported by this Bedrock model",
		)
	}
	return c.child.SendMessagesWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
}

// StreamResponseWithStructuredOutput delegates to the child vendor client.
func (c *Client) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	if c.child == nil {
		eventChan := make(chan llm.Event, 1)
		eventChan <- llm.Event{
			Type:  types.EventError,
			Error: errors.New("structured output not supported by this Bedrock model"),
		}
		close(eventChan)
		return eventChan
	}
	return c.child.StreamResponseWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
}
