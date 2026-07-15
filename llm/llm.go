// Package llm provides a unified interface for interacting with various Large
// Language Model providers.
//
// This package defines the [LLM] interface and the data types that flow through
// it. Concrete vendor implementations live in subpackages (llm/anthropic,
// llm/openai, llm/gemini, llm/azure, llm/bedrock, llm/vertexai); each subpackage
// exports its own NewLLM constructor that returns a tracing-wrapped client
// implementing the interface.
//
// OpenAI-compatible providers (Groq, OpenRouter, xAI, Mistral, Ollama, etc.) are
// not separate vendors — point [llm/openai].WithBaseURL at the appropriate
// endpoint:
//
//	import llmopenai "github.com/joakimcarlsson/ai/llm/openai"
//
//	groq := llmopenai.NewLLM(
//		llmopenai.WithAPIKey(os.Getenv("GROQ_API_KEY")),
//		llmopenai.WithBaseURL("https://api.groq.com/openai/v1"),
//		llmopenai.WithModel(model.GroqModels[model.LLaMA3_70B]),
//	)
//
// The [RegisterCustomProvider] / [GetCustomProvider] registry stores BYOM
// (Bring Your Own Model) configurations as data — callers look up the config
// and construct the client themselves; the registry has no implicit factory.
package llm

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tracing"
	"github.com/joakimcarlsson/ai/types"
)

// MaxRetries is the default maximum number of retry attempts.
// Vendor packages may override this in their RetryConfig.
const MaxRetries = 8

var customProviders = make(map[model.Provider]CustomProviderConfig)
var customProvidersMu sync.RWMutex

// CustomProviderConfig defines configuration for OpenAI-compatible custom providers.
// Use this to register BYOM (Bring Your Own Model) configurations like Ollama, LocalAI,
// or any OpenAI-compatible API.
//
// The registry is a config store; callers construct the client explicitly using
// the registered values:
//
//	ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
//	    BaseURL: "http://localhost:11434/v1",
//	    DefaultModel: model.NewCustomModel(...),
//	})
//
//	cfg, _ := llm.GetCustomProvider(ollama)
//	client := llmopenai.NewLLM(
//	    llmopenai.WithBaseURL(cfg.BaseURL),
//	    llmopenai.WithExtraHeaders(cfg.ExtraHeaders),
//	    llmopenai.WithModel(cfg.DefaultModel),
//	)
type CustomProviderConfig struct {
	// BaseURL is the base URL for the custom provider's API endpoint.
	BaseURL string

	// ExtraHeaders contains additional HTTP headers to include in requests.
	ExtraHeaders map[string]string

	// DefaultModel is the model configuration to use when none is specified.
	DefaultModel model.Model
}

// RegisterCustomProvider stores a BYOM configuration under a synthetic provider ID
// and returns that ID. Pair with [GetCustomProvider] when constructing the client.
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

// GetCustomProvider retrieves a previously-registered custom provider configuration.
func GetCustomProvider(provider model.Provider) (CustomProviderConfig, bool) {
	customProvidersMu.RLock()
	defer customProvidersMu.RUnlock()
	config, exists := customProviders[provider]
	return config, exists
}

// TokenUsage tracks the number of tokens consumed during an LLM interaction.
type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	// ReasoningTokens counts tokens spent on internal reasoning/thinking, as
	// reported by providers that surface it (OpenAI o-series, Gemini, DeepSeek).
	// These are billed within OutputTokens, not in addition to them.
	ReasoningTokens int64
}

// Add accumulates token counts from another TokenUsage into this one.
func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.CacheCreationTokens += other.CacheCreationTokens
	u.CacheReadTokens += other.CacheReadTokens
	u.ReasoningTokens += other.ReasoningTokens
}

// TokenTopLogProb is one of the most likely alternative tokens at a given
// position, paired with its log probability.
type TokenTopLogProb struct {
	Token   string
	LogProb float64
}

// TokenLogProb carries the log-probability information for a single output
// token. Top holds the most likely alternative tokens at that position (the
// requested top_logprobs), empty when none were requested.
type TokenLogProb struct {
	Token   string
	LogProb float64
	Top     []TokenTopLogProb
}

// Choice is one completion among several returned when more than one was
// requested (see llm/openai.WithN). Each choice carries its own content, finish
// reason, tool calls, and per-token log probabilities.
type Choice struct {
	Content      string
	FinishReason message.FinishReason
	ToolCalls    []message.ToolCall
	LogProbs     []TokenLogProb
}

// Response represents the complete response from an LLM provider.
type Response struct {
	Content                    string
	Reasoning                  string
	ToolCalls                  []message.ToolCall
	Usage                      TokenUsage
	FinishReason               message.FinishReason
	StructuredOutput           *string
	UsedNativeStructuredOutput bool
	// ProviderMetadata carries provider-specific structured data from
	// server-side built-in tools. Keys are namespaced per provider.
	ProviderMetadata map[string]any
	// LogProbs holds per-token log probabilities for the primary choice when
	// log probabilities were requested (llm/openai.WithLogprobs); nil
	// otherwise. Only OpenAI and OpenAI-compatible providers populate it.
	LogProbs []TokenLogProb
	// Choices holds every completion when more than one was requested
	// (llm/openai.WithN). The top-level Content / FinishReason / ToolCalls /
	// LogProbs mirror Choices[0]. Empty when a single choice was returned, so
	// callers use the top-level fields then. Only OpenAI and OpenAI-compatible
	// providers populate it.
	Choices []Choice
	// ProviderResponseID is the provider-assigned identifier for this response
	// (e.g. the OpenAI Responses API `response.id`). Empty for providers that do
	// not expose one. Callers can feed it back as the previous-response id to
	// chain server-side conversation state (prompt-cache continuity).
	ProviderResponseID string
	// RequestID is the provider request id (the x-request-id / request-id
	// response header) — the first thing provider support asks for when
	// debugging a bad response. Empty when the provider does not supply one
	// (e.g. Gemini).
	RequestID string
	// ResponseHeaders holds a small allowlist of debugging-relevant response
	// headers (request id, rate-limit, retry-after) copied from the provider's
	// HTTP response. Only those headers are retained — never the full set — to
	// avoid leaking auth-echo headers. Nil when unavailable.
	ResponseHeaders http.Header
}

// SelectResponseHeaders extracts the provider request id and a small allowlist
// of debugging-relevant headers (request id, rate-limit, retry-after) from an
// HTTP response header set. Vendor packages call it after a successful request
// to populate [Response.RequestID] and [Response.ResponseHeaders]. Only the
// allowlisted headers are copied, so callers never surface auth-echo headers.
// Both results are empty/nil when h carries none of them.
func SelectResponseHeaders(
	h http.Header,
) (requestID string, selected http.Header) {
	if len(h) == 0 {
		return "", nil
	}
	requestID = h.Get("X-Request-Id")
	if requestID == "" {
		requestID = h.Get("Request-Id")
	}
	for key, values := range h {
		lower := strings.ToLower(key)
		if lower == "x-request-id" || lower == "request-id" ||
			lower == "retry-after" || strings.Contains(lower, "ratelimit") {
			if selected == nil {
				selected = make(http.Header)
			}
			selected[key] = append([]string(nil), values...)
		}
	}
	return requestID, selected
}

// Event represents a single event in a streaming LLM response.
type Event struct {
	Type     types.EventType
	Content  string
	Thinking string
	Response *Response
	ToolCall *message.ToolCall
	Error    error
}

// LLM defines the interface for interacting with Large Language Model providers.
type LLM interface {
	// SendMessages sends a conversation to the LLM and returns the complete response.
	SendMessages(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) (*Response, error)

	// SendMessagesWithStructuredOutput sends a conversation and requests structured JSON output.
	SendMessagesWithStructuredOutput(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
		outputSchema *schema.StructuredOutputInfo,
	) (*Response, error)

	// StreamResponse sends a conversation and returns a channel of streaming events.
	//
	// Callers that stop reading before the channel closes MUST cancel ctx;
	// cancellation is what releases the stream's internal goroutines.
	StreamResponse(
		ctx context.Context,
		messages []message.Message,
		tools []tool.BaseTool,
	) <-chan Event

	// StreamResponseWithStructuredOutput streams a response with structured output constraints.
	//
	// Callers that stop reading before the channel closes MUST cancel ctx;
	// cancellation is what releases the stream's internal goroutines.
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

// TracingAttrs are construction-time attributes vendor packages forward to the
// [WithTracing] wrapper so they appear on every span produced for the wrapped
// client.
type TracingAttrs struct {
	MaxTokens   int64
	Temperature *float64
	TopP        *float64
}

// WithTracing wraps an LLM client so every call records OpenTelemetry spans and metrics.
// Vendor sub-packages return their concrete client wrapped in this so consumers always
// get tracing without thinking about it.
func WithTracing(inner LLM, attrs TracingAttrs) LLM {
	return &tracingLLM{inner: inner, attrs: attrs}
}

type tracingLLM struct {
	inner LLM
	attrs TracingAttrs
}

func (t *tracingLLM) Model() model.Model {
	return t.inner.Model()
}

func (t *tracingLLM) SupportsStructuredOutput() bool {
	return t.inner.SupportsStructuredOutput()
}

func (t *tracingLLM) spanAttrs() []tracing.Attr {
	var attrs []tracing.Attr
	if t.attrs.MaxTokens > 0 {
		attrs = append(
			attrs,
			tracing.AttrRequestMaxTokens.Int64(t.attrs.MaxTokens),
		)
	}
	if t.attrs.Temperature != nil {
		attrs = append(
			attrs,
			tracing.AttrRequestTemperature.Float64(*t.attrs.Temperature),
		)
	}
	if t.attrs.TopP != nil {
		attrs = append(attrs, tracing.AttrRequestTopP.Float64(*t.attrs.TopP))
	}
	return attrs
}

func (t *tracingLLM) recordResponseAttrs(
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
		attrs = append(attrs,
			tracing.AttrUsageCacheRead.Int64(resp.Usage.CacheReadTokens))
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

func cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func messageText(msg message.Message) string {
	return msg.Content().Text
}

func (t *tracingLLM) logMessages(
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

func (t *tracingLLM) recordMetrics(
	ctx context.Context,
	start time.Time,
	resp *Response,
	err error,
) {
	m := t.inner.Model()
	var inputTokens, outputTokens int64
	if resp != nil {
		inputTokens = resp.Usage.InputTokens
		outputTokens = resp.Usage.OutputTokens
	}
	tracing.RecordMetrics(
		ctx,
		"generate_content",
		m.APIModel,
		string(m.Provider),
		time.Since(start),
		inputTokens,
		outputTokens,
		err,
	)
}

func (t *tracingLLM) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*Response, error) {
	m := t.inner.Model()
	messages = cleanMessages(messages)
	start := time.Now()

	ctx, span := tracing.StartGenerateSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
	defer span.End()

	response, err := t.inner.SendMessages(ctx, messages, tools)
	if err != nil {
		tracing.SetError(span, err)
		t.recordMetrics(ctx, start, nil, err)
		return nil, err
	}

	t.recordResponseAttrs(span, response, len(tools))
	t.logMessages(ctx, messages, response)
	t.recordMetrics(ctx, start, response, nil)
	return response, nil
}

func (t *tracingLLM) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*Response, error) {
	m := t.inner.Model()
	messages = cleanMessages(messages)
	start := time.Now()

	ctx, span := tracing.StartGenerateSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)
	defer span.End()

	response, err := t.inner.SendMessagesWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
	if err != nil {
		tracing.SetError(span, err)
		t.recordMetrics(ctx, start, nil, err)
		return nil, err
	}

	t.recordResponseAttrs(span, response, len(tools))
	t.logMessages(ctx, messages, response)
	t.recordMetrics(ctx, start, response, nil)
	return response, nil
}

func (t *tracingLLM) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan Event {
	m := t.inner.Model()
	messages = cleanMessages(messages)
	start := time.Now()

	ctx, span := tracing.StartGenerateSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)

	innerCh := t.inner.StreamResponse(ctx, messages, tools)
	outCh := make(chan Event)
	go func() {
		defer close(outCh)
		defer span.End()
		for evt := range innerCh {
			if evt.Type == types.EventComplete && evt.Response != nil {
				t.recordResponseAttrs(span, evt.Response, len(tools))
				tracing.LogChoice(
					ctx,
					evt.Response.Content,
					string(evt.Response.FinishReason),
				)
				t.recordMetrics(ctx, start, evt.Response, nil)
			}
			if evt.Type == types.EventError && evt.Error != nil {
				tracing.SetError(span, evt.Error)
				t.recordMetrics(ctx, start, nil, evt.Error)
			}
			select {
			case outCh <- evt:
			case <-ctx.Done():
				// The consumer abandoned outCh. Drain innerCh so the
				// producer's blocking sends unblock and it can close the
				// channel, then return so the span ends instead of leaking
				// with this goroutine.
				drainEvents(innerCh)
				return
			}
		}
	}()
	return outCh
}

// drainEvents consumes the remaining events on ch so the producer's blocking
// sends unblock and it can reach its close. Used by the tracing forwarders
// when the consumer abandons the output channel.
func drainEvents(ch <-chan Event) {
	for {
		if _, ok := <-ch; !ok {
			return
		}
	}
}

func (t *tracingLLM) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan Event {
	m := t.inner.Model()
	messages = cleanMessages(messages)
	start := time.Now()

	ctx, span := tracing.StartGenerateSpan(
		ctx, m.APIModel, string(m.Provider), t.spanAttrs()...,
	)

	innerCh := t.inner.StreamResponseWithStructuredOutput(
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
				t.recordResponseAttrs(span, evt.Response, len(tools))
				tracing.LogChoice(
					ctx,
					evt.Response.Content,
					string(evt.Response.FinishReason),
				)
				t.recordMetrics(ctx, start, evt.Response, nil)
			}
			if evt.Type == types.EventError && evt.Error != nil {
				tracing.SetError(span, evt.Error)
				t.recordMetrics(ctx, start, nil, evt.Error)
			}
			select {
			case outCh <- evt:
			case <-ctx.Done():
				// The consumer abandoned outCh. Drain innerCh so the
				// producer's blocking sends unblock and it can close the
				// channel, then return so the span ends instead of leaking
				// with this goroutine.
				drainEvents(innerCh)
				return
			}
		}
	}()
	return outCh
}
