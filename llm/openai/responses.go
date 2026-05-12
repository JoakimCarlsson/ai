// File: responses.go — OpenAI Responses API client. The Responses API is a
// separate surface from Chat Completions and is the only place OpenAI exposes
// server-side built-in tools (web_search, file_search, code_interpreter).
// Construct with [NewResponsesLLM]; the Chat Completions client ([NewLLM])
// remains untouched and is the right choice for OpenAI-compatible wrappers.
package openai

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
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// SearchContextSize controls how much context window space the web search tool
// is permitted to consume. Maps to the Responses API search_context_size field.
type SearchContextSize string

// SearchContextSize values.
const (
	SearchContextLow    SearchContextSize = "low"
	SearchContextMedium SearchContextSize = "medium"
	SearchContextHigh   SearchContextSize = "high"
)

// UserLocation provides approximate user location for search relevance. Used
// by both [WithWebSearch] and [WithWebSearchPreview].
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

// WebSearchPreviewOpts configures the legacy web_search_preview built-in tool.
type WebSearchPreviewOpts struct {
	SearchContextSize  SearchContextSize
	SearchContentTypes []string
	UserLocation       *UserLocation
}

// ResponsesOptions configures the OpenAI Responses LLM client.
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
	builtinTools    []responses.ToolUnionParam
}

// ResponsesOption configures [ResponsesOptions].
type ResponsesOption func(*ResponsesOptions)

// WithResponsesAPIKey sets the API key used to authenticate.
func WithResponsesAPIKey(k string) ResponsesOption {
	return func(o *ResponsesOptions) { o.apiKey = k }
}

// WithResponsesModel selects the LLM model.
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

// WithResponsesBaseURL sets a custom API endpoint.
func WithResponsesBaseURL(u string) ResponsesOption {
	return func(o *ResponsesOptions) { o.baseURL = u }
}

// WithResponsesExtraHeaders adds custom HTTP headers to API requests.
func WithResponsesExtraHeaders(h map[string]string) ResponsesOption {
	return func(o *ResponsesOptions) { o.extraHeaders = h }
}

// WithResponsesReasoningEffort sets the reasoning effort for o-series / gpt-5
// reasoning models.
func WithResponsesReasoningEffort(e ReasoningEffort) ResponsesOption {
	return func(o *ResponsesOptions) { o.reasoningEffort = &e }
}

// WithWebSearch enables the web_search built-in tool. Pass a [WebSearchOpts]
// to tune context size, allowed domains, or user location.
func WithWebSearch(opts ...WebSearchOpts) ResponsesOption {
	return func(o *ResponsesOptions) {
		p := responses.WebSearchToolParam{
			Type: responses.WebSearchToolTypeWebSearch,
		}
		if len(opts) > 0 {
			c := opts[0]
			if c.SearchContextSize != "" {
				p.SearchContextSize = responses.WebSearchToolSearchContextSize(
					c.SearchContextSize,
				)
			}
			if len(c.AllowedDomains) > 0 {
				p.Filters = responses.WebSearchToolFiltersParam{
					AllowedDomains: c.AllowedDomains,
				}
			}
			if c.UserLocation != nil {
				p.UserLocation = responses.WebSearchToolUserLocationParam{
					City:     optString(c.UserLocation.City),
					Country:  optString(c.UserLocation.Country),
					Region:   optString(c.UserLocation.Region),
					Timezone: optString(c.UserLocation.Timezone),
				}
			}
		}
		o.builtinTools = append(
			o.builtinTools,
			responses.ToolUnionParam{OfWebSearch: &p},
		)
	}
}

// WithWebSearchPreview enables the legacy web_search_preview built-in tool for
// models that don't yet support [WithWebSearch].
func WithWebSearchPreview(opts ...WebSearchPreviewOpts) ResponsesOption {
	return func(o *ResponsesOptions) {
		p := responses.WebSearchPreviewToolParam{
			Type: responses.WebSearchPreviewToolTypeWebSearchPreview,
		}
		if len(opts) > 0 {
			c := opts[0]
			if c.SearchContextSize != "" {
				p.SearchContextSize = responses.WebSearchPreviewToolSearchContextSize(
					c.SearchContextSize,
				)
			}
			if len(c.SearchContentTypes) > 0 {
				p.SearchContentTypes = c.SearchContentTypes
			}
			if c.UserLocation != nil {
				p.UserLocation = responses.WebSearchPreviewToolUserLocationParam{
					City:     optString(c.UserLocation.City),
					Country:  optString(c.UserLocation.Country),
					Region:   optString(c.UserLocation.Region),
					Timezone: optString(c.UserLocation.Timezone),
				}
			}
		}
		o.builtinTools = append(
			o.builtinTools,
			responses.ToolUnionParam{OfWebSearchPreview: &p},
		)
	}
}

// WithFileSearch enables the file_search built-in tool against the given
// vector store IDs.
func WithFileSearch(vectorStoreIDs ...string) ResponsesOption {
	return func(o *ResponsesOptions) {
		o.builtinTools = append(o.builtinTools, responses.ToolUnionParam{
			OfFileSearch: &responses.FileSearchToolParam{
				VectorStoreIDs: vectorStoreIDs,
			},
		})
	}
}

// WithCodeInterpreter enables the code_interpreter built-in tool with an
// auto-provisioned ephemeral container.
func WithCodeInterpreter() ResponsesOption {
	return func(o *ResponsesOptions) {
		o.builtinTools = append(o.builtinTools, responses.ToolUnionParam{
			OfCodeInterpreter: &responses.ToolCodeInterpreterParam{
				Container: responses.ToolCodeInterpreterContainerUnionParam{
					OfCodeInterpreterToolAuto: &responses.ToolCodeInterpreterContainerCodeInterpreterContainerAutoParam{},
				},
			},
		})
	}
}

func optString(s string) param.Opt[string] {
	if s == "" {
		return param.Opt[string]{}
	}
	return openaisdk.String(s)
}

// responsesClient implements [llm.LLM] against the OpenAI Responses API.
type responsesClient struct {
	options ResponsesOptions
	client  openaisdk.Client
}

// NewResponsesLLM constructs an OpenAI client targeting the Responses API.
// Unlike [NewLLM] (Chat Completions), this client supports server-side
// built-in tools: [WithWebSearch], [WithFileSearch], [WithCodeInterpreter].
func NewResponsesLLM(opts ...ResponsesOption) llm.LLM {
	options := ResponsesOptions{}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}
	for k, v := range options.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	return llm.WithTracing(&responsesClient{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxOutputTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

// Model returns the configured LLM model.
func (c *responsesClient) Model() model.Model { return c.options.model }

// SupportsStructuredOutput reports whether the configured model supports
// structured output.
func (c *responsesClient) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *responsesClient) convertMessages(
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

func (c *responsesClient) convertTools(
	tools []tool.BaseTool,
) []responses.ToolUnionParam {
	out := make(
		[]responses.ToolUnionParam,
		0,
		len(tools)+len(c.options.builtinTools),
	)
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
	return append(out, c.options.builtinTools...)
}

func (c *responsesClient) preparedParams(
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
	if c.options.model.CanReason && c.options.reasoningEffort != nil {
		switch *c.options.reasoningEffort {
		case ReasoningEffortLow:
			params.Reasoning.Effort = shared.ReasoningEffortLow
		case ReasoningEffortMedium:
			params.Reasoning.Effort = shared.ReasoningEffortMedium
		case ReasoningEffortHigh:
			params.Reasoning.Effort = shared.ReasoningEffortHigh
		}
	}
	return params
}

// extractOutput walks a completed Response and returns assistant content,
// function tool calls, and provider metadata (citations from output_text
// annotations).
func (c *responsesClient) extractOutput(
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
		case "code_interpreter_call":
			if item.Code != "" {
				content.WriteString("\n```python\n")
				content.WriteString(item.Code)
				content.WriteString("\n```\n")
			}
			for _, out := range item.Outputs {
				if out.Logs != "" {
					content.WriteString("\n```\n")
					content.WriteString(out.Logs)
					content.WriteString("\n```\n")
				}
			}
		}
	}

	var meta map[string]any
	if len(citations) > 0 {
		meta = map[string]any{"openai.url_citations": citations}
	}
	return content.String(), toolCalls, meta
}

func (c *responsesClient) usage(resp *responses.Response) llm.TokenUsage {
	if resp == nil {
		return llm.TokenUsage{}
	}
	return llm.TokenUsage{
		InputTokens:     resp.Usage.InputTokens,
		OutputTokens:    resp.Usage.OutputTokens,
		CacheReadTokens: resp.Usage.InputTokensDetails.CachedTokens,
	}
}

func (c *responsesClient) finishReason(
	resp *responses.Response,
) message.FinishReason {
	if resp == nil {
		return message.FinishReasonUnknown
	}
	switch resp.IncompleteDetails.Reason {
	case "max_output_tokens":
		return message.FinishReasonMaxTokens
	}
	for _, item := range resp.Output {
		if item.Type == "function_call" {
			return message.FinishReasonToolUse
		}
	}
	return message.FinishReasonEndTurn
}

// SendMessages sends a conversation and returns the complete response.
func (c *responsesClient) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			resp, err := c.client.Responses.New(ctx, params)
			if err != nil {
				return nil, wrapError(err)
			}
			content, toolCalls, meta := c.extractOutput(resp)
			return &llm.Response{
				Content:          content,
				ToolCalls:        toolCalls,
				Usage:            c.usage(resp),
				FinishReason:     c.finishReason(resp),
				ProviderMetadata: meta,
			}, nil
		},
	)
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *responsesClient) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.Text = c.structuredTextConfig(outputSchema)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			resp, err := c.client.Responses.New(ctx, params)
			if err != nil {
				return nil, wrapError(err)
			}
			content, toolCalls, meta := c.extractOutput(resp)
			return &llm.Response{
				Content:                    content,
				ToolCalls:                  toolCalls,
				Usage:                      c.usage(resp),
				FinishReason:               c.finishReason(resp),
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
				ProviderMetadata:           meta,
			}, nil
		},
	)
}

func (c *responsesClient) structuredTextConfig(
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
func (c *responsesClient) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	return c.runStream(ctx, params, false)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *responsesClient) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.Text = c.structuredTextConfig(outputSchema)
	return c.runStream(ctx, params, true)
}

func (c *responsesClient) runStream(
	ctx context.Context,
	params responses.ResponseNewParams,
	structured bool,
) <-chan llm.Event {
	eventChan := make(chan llm.Event)
	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)

	go func() {
		defer close(eventChan)
		defer cancel()

		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			stream := c.client.Responses.NewStreaming(ctx, params)
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
					eventChan <- llm.Event{Type: types.EventContentDelta, Content: event.Delta}

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
					if ann, ok := annotationAsMap(event.Annotation); ok &&
						ann["type"] == "url_citation" {
						citations = append(citations, ann)
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
						meta = map[string]any{"openai.url_citations": citations}
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
				return wrapError(err)
			}
			return nil
		}, eventChan)
	}()

	return eventChan
}

type streamingFunctionCall struct {
	callID string
	name   string
	args   strings.Builder
}

func annotationAsMap(a any) (map[string]any, bool) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}
