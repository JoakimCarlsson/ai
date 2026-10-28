// Package xai responses.go defines a standalone client for xAI's Responses
// API, which exposes server-side built-in tools (web_search, x_search,
// code_execution). xAI's Responses endpoint mirrors OpenAI's wire format,
// so this client reuses openai-go for HTTP transport but builds tool entries
// directly so the xAI-specific x_search tool is first-class. The thin
// OpenAI-compatible wrapper [NewLLM] remains the right choice for users
// without built-ins.
package xai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// SearchContextSize controls how much context window space the web_search
// tool is permitted to consume.
type SearchContextSize string

// SearchContextSize values.
const (
	SearchContextLow    SearchContextSize = "low"
	SearchContextMedium SearchContextSize = "medium"
	SearchContextHigh   SearchContextSize = "high"
)

// UserLocation provides approximate user location for web_search relevance.
type UserLocation struct {
	City     string
	Country  string
	Region   string
	Timezone string
}

// WebSearchOpts configures the web_search built-in tool.
type WebSearchOpts struct {
	SearchContextSize SearchContextSize
	AllowedDomains    []string
	UserLocation      *UserLocation
}

// XSearchOpts configures the x_search built-in tool which searches X
// posts. Handle limits and date ranges follow xAI's documented bounds.
type XSearchOpts struct {
	AllowedXHandles          []string
	ExcludedXHandles         []string
	FromDate                 string
	ToDate                   string
	EnableImageUnderstanding bool
	EnableVideoUnderstanding bool
}

// Options configures the xAI LLM client.
type Options struct {
	apiKey            string
	model             llm.Model
	maxTokens         int64
	temperature       *float64
	topP              *float64
	timeout           *time.Duration
	baseURL           string
	extraHeaders      map[string]string
	reasoningEffort   *ReasoningEffort
	builtinTools      []map[string]any
	httpClient        *http.Client
	store             *bool
	toolChoice        *llm.ToolChoice
	parallelToolCalls *bool
}

// ReasoningEffort controls reasoning depth for xAI reasoning models.
type ReasoningEffort string

// ReasoningEffort values.
const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortXhigh  ReasoningEffort = "xhigh"
	ReasoningEffortHigh   ReasoningEffort = "high"
)

// Option configures [Options].
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate.
func WithAPIKey(k string) Option {
	return func(o *Options) { o.apiKey = k }
}

// WithModel selects the xAI model (e.g. grok-4 or grok-4.20-reasoning).
func WithModel(m llm.Model) Option {
	return func(o *Options) { o.model = m }
}

// WithMaxTokens sets the maximum number of output tokens.
func WithMaxTokens(n int64) Option {
	return func(o *Options) { o.maxTokens = n }
}

// WithTemperature controls randomness.
func WithTemperature(t float64) Option {
	return func(o *Options) { o.temperature = &t }
}

// WithTopP sets nucleus sampling probability mass.
func WithTopP(p float64) Option {
	return func(o *Options) { o.topP = &p }
}

// WithTimeout sets the maximum duration to wait for API responses.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) { o.timeout = &d }
}

// WithBaseURL overrides xAI's default Responses endpoint.
func WithBaseURL(u string) Option {
	return func(o *Options) { o.baseURL = u }
}

// WithExtraHeaders adds custom HTTP headers to API requests.
func WithExtraHeaders(h map[string]string) Option {
	return func(o *Options) { o.extraHeaders = h }
}

// WithStore sets response storage; when unset the API default
// applies. False requests encrypted reasoning for reasoning-capable models
// with include reasoning.encrypted_content, retains the completed reasoning
// items in [llm.Response].ReasoningItems and replays them from message history
// before their assistant output and function calls. Items without encrypted
// content are dropped; previous_response_id is never used.
func WithStore(store bool) Option {
	return func(o *Options) { o.store = &store }
}

// WithReasoningEffort sets effort for reasoning models. Low is the
// lowest effort; xAI reasoning models cannot turn reasoning off. Accepted levels
// vary by model. Unset uses the provider default.
// Note: xAI's grok-4 ignores this parameter.
func WithReasoningEffort(e ReasoningEffort) Option {
	return func(o *Options) { o.reasoningEffort = &e }
}

// WithHTTPClient injects a custom *http.Client, threaded into the
// OpenAI SDK via option.WithHTTPClient. Use it for outbound proxies, custom TLS
// (private CAs, mTLS), connection-pool tuning, or transport-level
// instrumentation. A nil client is a no-op, leaving the SDK default client in
// place. The per-request context timeout from WithTimeout still
// applies on top of the injected client's transport: the two compose and the
// shorter deadline wins.
func WithHTTPClient(c *http.Client) Option {
	return func(o *Options) { o.httpClient = c }
}

// WithToolChoice controls whether and which tool the model may call. It maps to
// tool_choice and is emitted only when tools are sent.
func WithToolChoice(choice llm.ToolChoice) Option {
	return func(o *Options) { o.toolChoice = &choice }
}

// WithParallelToolCalls toggles whether the model may return several tool calls
// in one response.
func WithParallelToolCalls(enabled bool) Option {
	return func(o *Options) { o.parallelToolCalls = &enabled }
}

// WithWebSearch enables the web_search built-in tool.
func WithWebSearch(opts ...WebSearchOpts) Option {
	return func(o *Options) {
		entry := map[string]any{"type": "web_search"}
		if len(opts) > 0 {
			cfg := opts[0]
			if cfg.SearchContextSize != "" {
				entry["search_context_size"] = string(cfg.SearchContextSize)
			}
			if len(cfg.AllowedDomains) > 0 {
				entry["filters"] = map[string]any{
					"allowed_domains": cfg.AllowedDomains,
				}
			}
			if cfg.UserLocation != nil {
				entry["user_location"] = userLocationMap(cfg.UserLocation)
			}
		}
		o.builtinTools = append(o.builtinTools, entry)
	}
}

// WithXSearch enables the x_search built-in tool for searching X posts.
func WithXSearch(opts ...XSearchOpts) Option {
	return func(o *Options) {
		entry := map[string]any{"type": "x_search"}
		if len(opts) > 0 {
			cfg := opts[0]
			if len(cfg.AllowedXHandles) > 0 {
				entry["allowed_x_handles"] = cfg.AllowedXHandles
			}
			if len(cfg.ExcludedXHandles) > 0 {
				entry["excluded_x_handles"] = cfg.ExcludedXHandles
			}
			if cfg.FromDate != "" {
				entry["from_date"] = cfg.FromDate
			}
			if cfg.ToDate != "" {
				entry["to_date"] = cfg.ToDate
			}
			if cfg.EnableImageUnderstanding {
				entry["enable_image_understanding"] = true
			}
			if cfg.EnableVideoUnderstanding {
				entry["enable_video_understanding"] = true
			}
		}
		o.builtinTools = append(o.builtinTools, entry)
	}
}

// WithCodeExecution enables the code_execution built-in tool.
func WithCodeExecution() Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, map[string]any{
			"type": "code_execution",
		})
	}
}

func userLocationMap(loc *UserLocation) map[string]any {
	m := map[string]any{"type": "approximate"}
	if loc.City != "" {
		m["city"] = loc.City
	}
	if loc.Country != "" {
		m["country"] = loc.Country
	}
	if loc.Region != "" {
		m["region"] = loc.Region
	}
	if loc.Timezone != "" {
		m["timezone"] = loc.Timezone
	}
	return m
}

// xaiResponsesClient implements [llm.LLM] against xAI's Responses API.
type xaiResponsesClient struct {
	options Options
	client  openaisdk.Client
}

// NewLLM constructs an xAI client targeting the Responses API.
// Pair with [WithWebSearch], [WithXSearch], and [WithCodeExecution] to enable
// xAI's server-side built-in tools.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{baseURL: DefaultBaseURL}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{option.WithBaseURL(options.baseURL)}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}
	for k, v := range options.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}
	if options.httpClient != nil {
		clientOpts = append(
			clientOpts,
			option.WithHTTPClient(options.httpClient),
		)
	}

	return llm.WithTracing(&xaiResponsesClient{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

func (c *xaiResponsesClient) Model() llm.Model { return c.options.model }

func (c *xaiResponsesClient) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *xaiResponsesClient) convertMessages(
	messages []message.Message,
) []responses.ResponseInputItemUnionParam {
	out := make([]responses.ResponseInputItemUnionParam, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			out = append(out, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleSystem,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openaisdk.String(msg.Content().String()),
					},
				},
			})
		case message.User:
			out = append(out, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openaisdk.String(msg.Content().String()),
					},
				},
			})
		case message.Assistant:
			if c.options.store != nil && !*c.options.store {
				out = append(out, llmopenai.ReasoningInputItems(msg)...)
			}
			if msg.Content().String() != "" {
				out = append(out, responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleAssistant,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openaisdk.String(msg.Content().String()),
						},
					},
				})
			}
			for _, call := range msg.ToolCalls() {
				out = append(out, responses.ResponseInputItemUnionParam{
					OfFunctionCall: &responses.ResponseFunctionToolCallParam{
						CallID:    call.ID,
						Name:      call.Name,
						Arguments: call.Input,
					},
				})
			}
		case message.Tool:
			for _, result := range msg.ToolResults() {
				out = append(out, responses.ResponseInputItemUnionParam{
					OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
						CallID: openaisdk.String(result.ToolCallID),
						Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
							OfString: openaisdk.String(result.Content),
						},
					},
				})
			}
		}
	}
	return out
}

func (c *xaiResponsesClient) convertTools(
	tools []tool.BaseTool,
) []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		info := t.Info()
		params := map[string]any{
			"type":       "object",
			"properties": info.Parameters,
		}
		if len(info.Required) > 0 {
			params["required"] = info.Required
		}
		out = append(out, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        info.Name,
				Description: openaisdk.String(info.Description),
				Parameters:  params,
				Strict:      openaisdk.Bool(false),
			},
		})
	}
	return out
}

func (c *xaiResponsesClient) preparedParams(
	input []responses.ResponseInputItemUnionParam,
	tools []responses.ToolUnionParam,
) responses.ResponseNewParams {
	params := responses.ResponseNewParams{
		Model: shared.ResponsesModel(c.options.model.APIModel),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		Tools: tools,
	}
	if c.options.store != nil {
		params.Store = openaisdk.Bool(*c.options.store)
		if !*c.options.store && c.options.model.CanReason {
			params.Include = []responses.ResponseIncludable{
				responses.ResponseIncludableReasoningEncryptedContent,
			}
		}
	}
	if len(tools) > 0 {
		if c.options.parallelToolCalls != nil {
			params.ParallelToolCalls = openaisdk.Bool(
				*c.options.parallelToolCalls,
			)
		}
		if c.options.toolChoice != nil {
			params.ToolChoice = toolChoiceParam(*c.options.toolChoice)
		}
	}
	if c.options.maxTokens > 0 {
		params.MaxOutputTokens = openaisdk.Int(c.options.maxTokens)
	}
	if c.options.temperature != nil {
		params.Temperature = openaisdk.Float(*c.options.temperature)
	}
	if c.options.topP != nil {
		params.TopP = openaisdk.Float(*c.options.topP)
	}
	if c.options.reasoningEffort != nil && c.options.model.CanReason {
		switch *c.options.reasoningEffort {
		case ReasoningEffortLow:
			params.Reasoning.Effort = shared.ReasoningEffortLow
		case ReasoningEffortHigh:
			params.Reasoning.Effort = shared.ReasoningEffortHigh
		case ReasoningEffortMedium:
			params.Reasoning.Effort = shared.ReasoningEffortMedium
		case ReasoningEffortXhigh:
			params.Reasoning.Effort = shared.ReasoningEffortXhigh
		}
	}
	return params
}

// requestOptions returns per-call SDK options. When built-in tools are
// configured, a JSON overlay replaces the request's tools array with the
// merged set (typed function tools first, raw built-in entries appended).
func (c *xaiResponsesClient) requestOptions(
	tools []responses.ToolUnionParam,
) []option.RequestOption {
	if len(c.options.builtinTools) == 0 {
		return nil
	}
	merged := make([]any, 0, len(tools)+len(c.options.builtinTools))
	for _, t := range tools {
		merged = append(merged, t)
	}
	for _, entry := range c.options.builtinTools {
		merged = append(merged, entry)
	}
	return []option.RequestOption{option.WithJSONSet("tools", merged)}
}

// extractOutput walks a completed Response and returns assistant content,
// function tool calls, and provider metadata (citations from output_text
// annotations).
func (c *xaiResponsesClient) extractOutput(
	resp *responses.Response,
) (string, []message.ToolCall, map[string]any) {
	var content strings.Builder
	var toolCalls []message.ToolCall
	var citations []map[string]any

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type != "output_text" {
					continue
				}
				content.WriteString(part.Text)
				for _, ann := range part.Annotations {
					if ann.Type == "url_citation" {
						citations = append(citations, map[string]any{
							"url":         ann.URL,
							"title":       ann.Title,
							"start_index": ann.StartIndex,
							"end_index":   ann.EndIndex,
						})
					}
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, message.ToolCall{
				ID:       item.CallID,
				Name:     item.Name,
				Input:    item.Arguments.OfString,
				Type:     "function",
				Finished: true,
			})
		}
	}

	var meta map[string]any
	if len(citations) > 0 {
		meta = map[string]any{"xai.citations": citations}
	}
	return content.String(), toolCalls, meta
}

// usage maps the Responses usage block onto llm.TokenUsage. The API reports
// input_tokens as the whole prompt and cached_tokens as the cache-hit subset
// of it, so InputTokens carries the uncached remainder, clamped at zero.
func (c *xaiResponsesClient) usage(resp *responses.Response) llm.TokenUsage {
	if resp == nil {
		return llm.TokenUsage{}
	}
	cached := resp.Usage.InputTokensDetails.CachedTokens
	return llm.TokenUsage{
		InputTokens:     max(resp.Usage.InputTokens-cached, 0),
		OutputTokens:    resp.Usage.OutputTokens,
		CacheReadTokens: cached,
		ReasoningTokens: resp.Usage.OutputTokensDetails.ReasoningTokens,
	}
}

func (c *xaiResponsesClient) finishReason(
	resp *responses.Response,
) message.FinishReason {
	if resp == nil {
		return message.FinishReasonUnknown
	}
	if resp.IncompleteDetails.Reason == "max_output_tokens" {
		return message.FinishReasonMaxTokens
	}
	for _, item := range resp.Output {
		if item.Type == "function_call" {
			return message.FinishReasonToolUse
		}
	}
	return message.FinishReasonEndTurn
}

func (c *xaiResponsesClient) buildResponse(
	resp *responses.Response,
) *llm.Response {
	content, toolCalls, meta := c.extractOutput(resp)
	return &llm.Response{
		Content:          content,
		ToolCalls:        toolCalls,
		Usage:            c.usage(resp),
		FinishReason:     c.finishReason(resp),
		ProviderMetadata: meta,
		ReasoningItems:   c.extractReasoning(resp.Output),
	}
}

// extractReasoning retains completed reasoning items for stateless replay.
func (c *xaiResponsesClient) extractReasoning(
	output []responses.ResponseOutputItemUnion,
) []message.ReasoningContent {
	if c.options.store == nil || *c.options.store {
		return nil
	}
	return llmopenai.ReasoningFromOutput(output)
}

// SendMessages sends a conversation and returns the complete response.
func (c *xaiResponsesClient) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	reqOpts := c.requestOptions(convertedTools)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		llm.DefaultRetryConfig(),
		func() (*llm.Response, error) {
			resp, err := c.client.Responses.New(ctx, params, reqOpts...)
			if err != nil {
				return nil, err
			}
			return c.buildResponse(resp), nil
		},
	)
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *xaiResponsesClient) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	params.Text = c.structuredTextConfig(outputSchema)
	reqOpts := c.requestOptions(convertedTools)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		llm.DefaultRetryConfig(),
		func() (*llm.Response, error) {
			resp, err := c.client.Responses.New(ctx, params, reqOpts...)
			if err != nil {
				return nil, err
			}
			out := c.buildResponse(resp)
			out.StructuredOutput = &out.Content
			out.UsedNativeStructuredOutput = true
			return out, nil
		},
	)
}

func (c *xaiResponsesClient) structuredTextConfig(
	outputSchema *schema.StructuredOutputInfo,
) responses.ResponseTextConfigParam {
	schemaMap := map[string]any{
		"type":       "object",
		"properties": outputSchema.Parameters,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}
	return responses.ResponseTextConfigParam{
		Format: responses.ResponseFormatTextConfigUnionParam{
			OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
				Name:   "structured_output",
				Schema: schemaMap,
				Strict: openaisdk.Bool(true),
			},
		},
	}
}

// StreamResponse sends a conversation and returns a channel of streaming events.
func (c *xaiResponsesClient) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	if err := c.validate(); err != nil {
		return errorEvent(err)
	}
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	return c.runStream(ctx, params, c.requestOptions(convertedTools), false)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *xaiResponsesClient) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	if err := c.validate(); err != nil {
		return errorEvent(err)
	}
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	params.Text = c.structuredTextConfig(outputSchema)
	return c.runStream(ctx, params, c.requestOptions(convertedTools), true)
}

type streamingFunctionCall struct {
	callID string
	name   string
	args   strings.Builder
}

func (c *xaiResponsesClient) runStream(
	ctx context.Context,
	params responses.ResponseNewParams,
	reqOpts []option.RequestOption,
	structured bool,
) <-chan llm.Event {
	eventChan := make(chan llm.Event)
	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)

	go func() {
		defer close(eventChan)
		defer cancel()

		llm.ExecuteStreamWithRetry(ctx, llm.DefaultRetryConfig(), func() error {
			stream := c.client.Responses.NewStreaming(ctx, params, reqOpts...)
			var content strings.Builder
			var citations []map[string]any
			var reasoningOutput []responses.ResponseOutputItemUnion
			pendingCalls := map[string]*streamingFunctionCall{}
			contentStarted := false

			for stream.Next() {
				event := stream.Current()
				switch event.Type {
				case "response.output_text.delta":
					if !contentStarted {
						eventChan <- llm.Event{Type: types.EventContentStart}
						contentStarted = true
					}
					content.WriteString(event.Delta)
					eventChan <- llm.Event{
						Type:    types.EventContentDelta,
						Content: event.Delta,
					}

				case "response.output_text.done":
					if contentStarted {
						eventChan <- llm.Event{Type: types.EventContentStop}
						contentStarted = false
					}

				case "response.output_item.added":
					if event.Item.Type == "function_call" {
						pendingCalls[event.Item.ID] = &streamingFunctionCall{
							callID: event.Item.CallID,
							name:   event.Item.Name,
						}
						eventChan <- llm.Event{
							Type: types.EventToolUseStart,
							ToolCall: &message.ToolCall{
								ID:   event.Item.CallID,
								Name: event.Item.Name,
							},
						}
					}

				case "response.output_item.done":
					if event.Item.Type == "reasoning" {
						reasoningOutput = append(reasoningOutput, event.Item)
					}

				case "response.function_call_arguments.delta":
					call, ok := pendingCalls[event.ItemID]
					if !ok {
						continue
					}
					call.args.WriteString(event.Delta)
					eventChan <- llm.Event{
						Type: types.EventToolUseDelta,
						ToolCall: &message.ToolCall{
							ID:    call.callID,
							Input: event.Delta,
						},
					}

				case "response.output_text.annotation.added":
					if cit, ok := urlCitationFromAnnotation(
						event.Annotation,
					); ok {
						citations = append(citations, cit)
					}

				case "response.completed":
					var toolCalls []message.ToolCall
					for _, call := range pendingCalls {
						toolCalls = append(toolCalls, message.ToolCall{
							ID:       call.callID,
							Name:     call.name,
							Input:    call.args.String(),
							Type:     "function",
							Finished: true,
						})
						eventChan <- llm.Event{
							Type:     types.EventToolUseStop,
							ToolCall: &message.ToolCall{ID: call.callID},
						}
					}
					if contentStarted {
						eventChan <- llm.Event{Type: types.EventContentStop}
					}
					contentStr := content.String()
					var meta map[string]any
					if len(citations) > 0 {
						meta = map[string]any{"xai.citations": citations}
					}
					finalResp := &llm.Response{
						Content:          contentStr,
						ToolCalls:        toolCalls,
						Usage:            c.usage(&event.Response),
						FinishReason:     c.finishReason(&event.Response),
						ProviderMetadata: meta,
						ReasoningItems:   c.extractReasoning(reasoningOutput),
					}
					if len(finalResp.ReasoningItems) == 0 {
						finalResp.ReasoningItems = c.extractReasoning(
							event.Response.Output,
						)
					}
					if structured {
						finalResp.StructuredOutput = &contentStr
						finalResp.UsedNativeStructuredOutput = true
					}
					eventChan <- llm.Event{Type: types.EventComplete, Response: finalResp}

				case "error", "response.failed", "response.incomplete":
					if event.Message != "" {
						return errors.New(event.Message)
					}
				}
			}

			if err := stream.Err(); err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			return nil
		}, eventChan)
	}()

	return eventChan
}

// urlCitationFromAnnotation extracts a url_citation streaming annotation into
// the same flat shape produced by [xaiResponsesClient.extractOutput], so
// streaming and non-streaming consumers see identical citation entries.
func urlCitationFromAnnotation(a any) (map[string]any, bool) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, false
	}
	var raw struct {
		Type       string `json:"type"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		StartIndex int64  `json:"start_index"`
		EndIndex   int64  `json:"end_index"`
	}
	if err := json.Unmarshal(b, &raw); err != nil ||
		raw.Type != "url_citation" {
		return nil, false
	}
	return map[string]any{
		"url":         raw.URL,
		"title":       raw.Title,
		"start_index": raw.StartIndex,
		"end_index":   raw.EndIndex,
	}, true
}

// validate rejects a malformed tool choice before a request is sent.
func (c *xaiResponsesClient) validate() error {
	if c.options.toolChoice == nil {
		return nil
	}
	return c.options.toolChoice.Validate()
}

// errorEvent returns a closed channel carrying a single error event, used to
// surface pre-flight failures on the streaming API.
func errorEvent(err error) <-chan llm.Event {
	ch := make(chan llm.Event, 1)
	ch <- llm.Event{Type: types.EventError, Error: err}
	close(ch)
	return ch
}

// toolChoiceParam maps a vendor-neutral [llm.ToolChoice] onto the Responses
// API tool_choice union.
func toolChoiceParam(
	choice llm.ToolChoice,
) responses.ResponseNewParamsToolChoiceUnion {
	switch choice.Mode {
	case llm.ToolChoiceNone:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openaisdk.Opt(responses.ToolChoiceOptionsNone),
		}
	case llm.ToolChoiceRequired:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openaisdk.Opt(
				responses.ToolChoiceOptionsRequired,
			),
		}
	case llm.ToolChoiceSpecific:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfFunctionTool: &responses.ToolChoiceFunctionParam{
				Name: choice.Name,
			},
		}
	default:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openaisdk.Opt(responses.ToolChoiceOptionsAuto),
		}
	}
}
