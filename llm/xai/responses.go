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
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
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

// ResponsesOptions configures the xAI Responses LLM client.
type ResponsesOptions struct {
	apiKey          string
	model           model.Model
	maxOutputTokens int64
	temperature     *float64
	topP            *float64
	timeout         *time.Duration
	baseURL         string
	extraHeaders    map[string]string
	reasoningEffort *ReasoningEffort
	builtinTools    []map[string]any
}

// ReasoningEffort controls reasoning depth for xAI reasoning models.
type ReasoningEffort string

// ReasoningEffort values.
const (
	ReasoningEffortLow  ReasoningEffort = "low"
	ReasoningEffortHigh ReasoningEffort = "high"
)

// ResponsesOption configures [ResponsesOptions].
type ResponsesOption func(*ResponsesOptions)

// WithResponsesAPIKey sets the API key used to authenticate.
func WithResponsesAPIKey(k string) ResponsesOption {
	return func(o *ResponsesOptions) { o.apiKey = k }
}

// WithResponsesModel selects the xAI model (e.g. grok-4 or grok-4.20-reasoning).
func WithResponsesModel(m model.Model) ResponsesOption {
	return func(o *ResponsesOptions) { o.model = m }
}

// WithResponsesMaxTokens sets the maximum number of output tokens.
func WithResponsesMaxTokens(n int64) ResponsesOption {
	return func(o *ResponsesOptions) { o.maxOutputTokens = n }
}

// WithResponsesTemperature controls randomness.
func WithResponsesTemperature(t float64) ResponsesOption {
	return func(o *ResponsesOptions) { o.temperature = &t }
}

// WithResponsesTopP sets nucleus sampling probability mass.
func WithResponsesTopP(p float64) ResponsesOption {
	return func(o *ResponsesOptions) { o.topP = &p }
}

// WithResponsesTimeout sets the maximum duration to wait for API responses.
func WithResponsesTimeout(d time.Duration) ResponsesOption {
	return func(o *ResponsesOptions) { o.timeout = &d }
}

// WithResponsesBaseURL overrides xAI's default Responses endpoint.
func WithResponsesBaseURL(u string) ResponsesOption {
	return func(o *ResponsesOptions) { o.baseURL = u }
}

// WithResponsesExtraHeaders adds custom HTTP headers to API requests.
func WithResponsesExtraHeaders(h map[string]string) ResponsesOption {
	return func(o *ResponsesOptions) { o.extraHeaders = h }
}

// WithResponsesReasoningEffort sets the reasoning effort for reasoning models.
// Note: xAI's grok-4 ignores this parameter.
func WithResponsesReasoningEffort(e ReasoningEffort) ResponsesOption {
	return func(o *ResponsesOptions) { o.reasoningEffort = &e }
}

// WithWebSearch enables the web_search built-in tool.
func WithWebSearch(opts ...WebSearchOpts) ResponsesOption {
	return func(o *ResponsesOptions) {
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
func WithXSearch(opts ...XSearchOpts) ResponsesOption {
	return func(o *ResponsesOptions) {
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
func WithCodeExecution() ResponsesOption {
	return func(o *ResponsesOptions) {
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
	options ResponsesOptions
	client  openaisdk.Client
}

// NewResponsesLLM constructs an xAI client targeting the Responses API.
// Pair with [WithWebSearch], [WithXSearch], and [WithCodeExecution] to enable
// xAI's server-side built-in tools.
func NewResponsesLLM(opts ...ResponsesOption) llm.LLM {
	options := ResponsesOptions{baseURL: DefaultBaseURL}
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

	return llm.WithTracing(&xaiResponsesClient{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxOutputTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

func (c *xaiResponsesClient) Model() model.Model { return c.options.model }

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
						CallID: result.ToolCallID,
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
	if c.options.maxOutputTokens > 0 {
		params.MaxOutputTokens = openaisdk.Int(c.options.maxOutputTokens)
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

func (c *xaiResponsesClient) usage(resp *responses.Response) llm.TokenUsage {
	if resp == nil {
		return llm.TokenUsage{}
	}
	return llm.TokenUsage{
		InputTokens:     resp.Usage.InputTokens,
		OutputTokens:    resp.Usage.OutputTokens,
		CacheReadTokens: resp.Usage.InputTokensDetails.CachedTokens,
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
	}
}

// SendMessages sends a conversation and returns the complete response.
func (c *xaiResponsesClient) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
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
					if cit, ok := urlCitationFromAnnotation(event.Annotation); ok {
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
