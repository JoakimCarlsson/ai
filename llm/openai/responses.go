// Package openai responses.go provides the OpenAI Responses API client behind
// [NewLLM]. The Responses API accepts function tools on every OpenAI model and
// is the only place OpenAI exposes server-side built-in tools (web_search,
// file_search, code_interpreter). The Chat Completions client
// ([NewChatCompletionsLLM]) is kept for OpenAI-compatible endpoints.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
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

// WithStore sets response storage; when unset the API default applies.
// False requests encrypted reasoning for reasoning-capable models and replays
// it from message history. Items without encrypted content are dropped rather
// than fetched by ID; previous_response_id is never used.
// Azure encrypted reasoning requires api-version v1 via /openai/v1/responses,
// without an api-version query parameter. Dated previews through
// 2025-04-01-preview do not support it. See
// https://learn.microsoft.com/en-us/azure/foundry/openai/api-version-lifecycle.
func WithStore(store bool) Option {
	return func(o *Options) { o.store = &store }
}

// WithWebSearch enables the web_search built-in tool. Pass a [WebSearchOpts]
// to tune context size, allowed domains, or user location. A configured user
// location is always sent with type "approximate": the API requires the field
// and accepts no other value, and leaving it unset would drop the key entirely
// and get the request refused.
func WithWebSearch(opts ...WebSearchOpts) Option {
	return func(o *Options) {
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
					Type:     "approximate",
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
func WithWebSearchPreview(opts ...WebSearchPreviewOpts) Option {
	return func(o *Options) {
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
func WithFileSearch(vectorStoreIDs ...string) Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, responses.ToolUnionParam{
			OfFileSearch: &responses.FileSearchToolParam{
				VectorStoreIDs: vectorStoreIDs,
			},
		})
	}
}

// WithCodeInterpreter enables the code_interpreter built-in tool with an
// auto-provisioned ephemeral container.
func WithCodeInterpreter() Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, responses.ToolUnionParam{
			OfCodeInterpreter: &responses.ToolCodeInterpreterParam{
				Container: responses.ToolCodeInterpreterContainerUnionParam{
					OfCodeInterpreterToolAuto: &responses.ToolCodeInterpreterContainerCodeInterpreterContainerAutoParam{},
				},
			},
		})
	}
}

// optString omits empty optional string parameters.
func optString(s string) param.Opt[string] {
	if s == "" {
		return param.Opt[string]{}
	}
	return openaisdk.String(s)
}

// ResponsesClient implements [llm.LLM] against the OpenAI Responses API.
type ResponsesClient struct {
	options Options
	client  openaisdk.Client
}

// NewLLM constructs an OpenAI LLM client. It targets the Responses API, which
// accepts function tools on every OpenAI model and is the only place OpenAI
// exposes server-side built-in tools: [WithWebSearch], [WithFileSearch] and
// [WithCodeInterpreter]. The returned [llm.LLM] is wrapped with
// [llm.WithTracing], so callers always get tracing spans and metrics.
//
// Options that exist only on Chat Completions (stop sequences, seed, penalties,
// logit bias, n and logprobs) are not accepted by the Responses API. Setting
// one makes every request fail with an error naming it, rather than being
// silently ignored. Use [NewChatCompletionsLLM] for OpenAI-compatible endpoints
// that need the Chat Completions API.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
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
	if options.httpClient != nil {
		clientOpts = append(
			clientOpts,
			option.WithHTTPClient(options.httpClient),
		)
	}

	return llm.WithTracing(
		NewResponsesWithExistingClient(
			options,
			openaisdk.NewClient(clientOpts...),
		),
		llm.TracingAttrs{
			MaxTokens:   options.maxTokens,
			Temperature: options.temperature,
			TopP:        options.topP,
		},
	)
}

// NewResponsesWithExistingClient constructs a bare Responses client with a
// preconfigured SDK client, allowing providers to supply authentication and
// endpoint routing while sharing request building and response handling.
// The caller is responsible for wrapping it with llm.WithTracing.
func NewResponsesWithExistingClient(
	options Options,
	client openaisdk.Client,
) *ResponsesClient {
	return &ResponsesClient{options: options, client: client}
}

// Model returns the configured LLM model.
func (c *ResponsesClient) Model() llm.Model { return c.options.model }

// SupportsStructuredOutput reports whether the configured model supports
// structured output.
func (c *ResponsesClient) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

// convertMessages builds stateless Responses input from conversation history.
func (c *ResponsesClient) convertMessages(
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
					Role:    responses.EasyInputMessageRoleUser,
					Content: userInputContent(msg),
				},
			})
		case message.Assistant:
			if c.options.store != nil && !*c.options.store {
				out = append(out, ReasoningInputItems(msg)...)
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

// userInputContent builds the Responses-API content for a user message. A
// text-only message keeps the simple string form; a message that also
// carries image parts is emitted as a content list of input_text +
// input_image items instead. BinaryContent has no detail hint of its own, so
// it is sent with "auto" and the model decides.
func userInputContent(
	msg message.Message,
) responses.EasyInputMessageContentUnionParam {
	imageURLs := msg.ImageURLContent()
	binaries := msg.BinaryContent()
	if len(imageURLs) == 0 && len(binaries) == 0 {
		return responses.EasyInputMessageContentUnionParam{
			OfString: openaisdk.String(msg.Content().String()),
		}
	}

	var parts responses.ResponseInputMessageContentListParam
	if text := msg.Content().String(); text != "" {
		parts = append(parts, responses.ResponseInputContentUnionParam{
			OfInputText: &responses.ResponseInputTextParam{Text: text},
		})
	}
	for _, img := range imageURLs {
		parts = append(parts, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				ImageURL: openaisdk.String(img.URL),
				Detail:   inputImageDetail(img.Detail),
			},
		})
	}
	for _, bin := range binaries {
		parts = append(parts, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				ImageURL: openaisdk.String(bin.String("openai")),
				Detail:   responses.ResponseInputImageDetailAuto,
			},
		})
	}
	return responses.EasyInputMessageContentUnionParam{
		OfInputItemContentList: parts,
	}
}

// inputImageDetail maps message.ImageURLContent.Detail ("low"/"high"/"auto",
// or empty) onto the Responses input-image detail enum, defaulting an empty or
// unrecognized value to "auto" (the API's own default) rather than sending an
// invalid detail string.
func inputImageDetail(detail string) responses.ResponseInputImageDetail {
	switch detail {
	case "low":
		return responses.ResponseInputImageDetailLow
	case "high":
		return responses.ResponseInputImageDetailHigh
	default:
		return responses.ResponseInputImageDetailAuto
	}
}

// convertTools combines function definitions with configured built-in tools.
func (c *ResponsesClient) convertTools(
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

// preparedParams applies request options to Responses input and tools.
func (c *ResponsesClient) preparedParams(
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
			params.ToolChoice = responsesToolChoice(*c.options.toolChoice)
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
	if c.options.model.CanReason && c.options.reasoningEffort != nil {
		switch *c.options.reasoningEffort {
		case ReasoningEffortNone:
			params.Reasoning.Effort = shared.ReasoningEffortNone
		case ReasoningEffortMinimal:
			params.Reasoning.Effort = shared.ReasoningEffortMinimal
		case ReasoningEffortLow:
			params.Reasoning.Effort = shared.ReasoningEffortLow
		case ReasoningEffortMedium:
			params.Reasoning.Effort = shared.ReasoningEffortMedium
		case ReasoningEffortHigh:
			params.Reasoning.Effort = shared.ReasoningEffortHigh
		case ReasoningEffortXhigh:
			params.Reasoning.Effort = shared.ReasoningEffortXhigh
		case ReasoningEffortMax:
			params.Reasoning.Effort = shared.ReasoningEffortMax
		}
	}
	return params
}

// extractOutput walks a completed Response and returns assistant content,
// function tool calls, and provider metadata (citations from output_text
// annotations).
func (c *ResponsesClient) extractOutput(
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

	meta := webSearchMetadata(citations, webSearchCalls(resp.Output))
	return content.String(), toolCalls, meta
}

// extractReasoning retains completed reasoning items for stateless replay.
func (c *ResponsesClient) extractReasoning(
	output []responses.ResponseOutputItemUnion,
) []message.ReasoningContent {
	if c.options.store == nil || *c.options.store {
		return nil
	}
	return ReasoningFromOutput(output)
}

// ReasoningInputItems converts the encrypted reasoning parts of an assistant
// message into Responses input items, in message order. Parts without
// encrypted content cannot be replayed statelessly and are dropped.
func ReasoningInputItems(
	msg message.Message,
) []responses.ResponseInputItemUnionParam {
	var out []responses.ResponseInputItemUnionParam
	for _, item := range msg.ReasoningContent() {
		if item.EncryptedContent == "" {
			continue
		}
		summary := make(
			[]responses.ResponseReasoningItemSummaryParam,
			0,
			len(item.Summary),
		)
		for _, text := range item.Summary {
			summary = append(
				summary,
				responses.ResponseReasoningItemSummaryParam{Text: text},
			)
		}
		out = append(out, responses.ResponseInputItemUnionParam{
			OfReasoning: &responses.ResponseReasoningItemParam{
				ID:               item.ID,
				EncryptedContent: openaisdk.String(item.EncryptedContent),
				Summary:          summary,
			},
		})
	}
	return out
}

// ReasoningFromOutput collects the reasoning items of a completed Responses
// output in order, keeping their IDs, encrypted content and summaries.
func ReasoningFromOutput(
	output []responses.ResponseOutputItemUnion,
) []message.ReasoningContent {
	var items []message.ReasoningContent
	for _, item := range output {
		if item.Type != "reasoning" {
			continue
		}
		reasoning := message.ReasoningContent{
			ID:               item.ID,
			EncryptedContent: item.EncryptedContent,
		}
		for _, summary := range item.Summary {
			reasoning.Summary = append(reasoning.Summary, summary.Text)
		}
		reasoning.Text = strings.Join(reasoning.Summary, "\n")
		items = append(items, reasoning)
	}
	return items
}

// webSearchCall flattens a web_search_call output item into the entry reported
// under openai.web_search_calls. Each item carries its own action (search,
// open_page, find_in_page) rather than being summed. The queries fall back to
// the deprecated singular query field when the plural list is empty, so a
// reply that only fills the old field still reports what was searched.
func webSearchCall(item responses.ResponseOutputItemUnion) map[string]any {
	queries := append([]string(nil), item.Action.Queries...)
	if len(queries) == 0 && item.Action.Query != "" {
		queries = []string{item.Action.Query}
	}
	return map[string]any{
		"id":      item.ID,
		"status":  item.Status,
		"action":  item.Action.Type,
		"queries": queries,
	}
}

// webSearchCalls collects every web_search_call item in output, in order.
func webSearchCalls(
	output []responses.ResponseOutputItemUnion,
) []map[string]any {
	var searches []map[string]any
	for _, item := range output {
		if item.Type == "web_search_call" {
			searches = append(searches, webSearchCall(item))
		}
	}
	return searches
}

// webSearchMetadata builds the provider metadata shared by the streaming and
// non-streaming paths: openai.url_citations and openai.web_search_calls, each
// present only when non-empty. It returns nil when both are empty.
func webSearchMetadata(citations, searches []map[string]any) map[string]any {
	if len(citations) == 0 && len(searches) == 0 {
		return nil
	}
	meta := map[string]any{}
	if len(citations) > 0 {
		meta["openai.url_citations"] = citations
	}
	if len(searches) > 0 {
		meta["openai.web_search_calls"] = searches
	}
	return meta
}

// usage maps a Response's token usage onto [llm.TokenUsage]. The API reports
// input_tokens as the whole prompt and cached_tokens as the part of it that hit
// the cache, a subset rather than an addition, so InputTokens holds the
// uncached remainder, matching Client.usage's chat-completions mapping.
func (c *ResponsesClient) usage(resp *responses.Response) llm.TokenUsage {
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

// finishReason maps the response status and output items to a finish reason.
func (c *ResponsesClient) finishReason(
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

// SendMessages sends a conversation and returns the complete response.
func (c *ResponsesClient) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
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
			var raw *http.Response
			resp, err := c.client.Responses.New(
				ctx, params, append(c.requestOptions(), option.WithResponseInto(&raw))...,
			)
			if err != nil {
				return nil, wrapError(err)
			}
			content, toolCalls, meta := c.extractOutput(resp)
			out := &llm.Response{
				Content:            content,
				ToolCalls:          toolCalls,
				Usage:              c.usage(resp),
				FinishReason:       c.finishReason(resp),
				ProviderMetadata:   meta,
				ProviderResponseID: resp.ID,
				ReasoningItems:     c.extractReasoning(resp.Output),
			}
			applyResponseHeaders(out, raw)
			return out, nil
		},
	)
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *ResponsesClient) SendMessagesWithStructuredOutput(
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
			var raw *http.Response
			resp, err := c.client.Responses.New(
				ctx, params, append(c.requestOptions(), option.WithResponseInto(&raw))...,
			)
			if err != nil {
				return nil, wrapError(err)
			}
			content, toolCalls, meta := c.extractOutput(resp)
			out := &llm.Response{
				Content:                    content,
				ToolCalls:                  toolCalls,
				Usage:                      c.usage(resp),
				FinishReason:               c.finishReason(resp),
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
				ProviderMetadata:           meta,
				ProviderResponseID:         resp.ID,
				ReasoningItems:             c.extractReasoning(resp.Output),
			}
			applyResponseHeaders(out, raw)
			return out, nil
		},
	)
}

// structuredTextConfig builds a closed object schema for strict Responses output.
func (c *ResponsesClient) structuredTextConfig(
	outputSchema *schema.StructuredOutputInfo,
) responses.ResponseTextConfigParam {
	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
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
func (c *ResponsesClient) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	if err := c.validate(); err != nil {
		return errorEvent(err)
	}
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	return c.runStream(ctx, params, false)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *ResponsesClient) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	if err := c.validate(); err != nil {
		return errorEvent(err)
	}
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.Text = c.structuredTextConfig(outputSchema)
	return c.runStream(ctx, params, true)
}

// runStream translates Responses stream events and retains completed reasoning.
func (c *ResponsesClient) runStream(
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
			var raw *http.Response
			stream := c.client.Responses.NewStreaming(
				ctx, params, append(c.requestOptions(), option.WithResponseInto(&raw))...,
			)
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
					meta := webSearchMetadata(
						citations,
						webSearchCalls(event.Response.Output),
					)
					finalResp := &llm.Response{
						Content:            contentStr,
						ToolCalls:          toolCalls,
						Usage:              c.usage(&event.Response),
						FinishReason:       c.finishReason(&event.Response),
						ProviderMetadata:   meta,
						ProviderResponseID: event.Response.ID,
						ReasoningItems:     c.extractReasoning(reasoningOutput),
					}
					if len(finalResp.ReasoningItems) == 0 {
						finalResp.ReasoningItems = c.extractReasoning(
							event.Response.Output,
						)
					}
					applyResponseHeaders(finalResp, raw)
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

// urlCitationFromAnnotation extracts a url_citation streaming annotation into
// the same flat shape produced by [ResponsesClient.extractOutput], so
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

// responsesToolChoice maps a vendor-neutral [llm.ToolChoice] onto the Responses
// API tool_choice union.
func responsesToolChoice(
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

// validate rejects a request whose options the Responses API has no field for,
// so they fail loudly instead of being dropped.
func (c *ResponsesClient) validate() error {
	o := c.options
	if o.toolChoice != nil {
		if err := o.toolChoice.Validate(); err != nil {
			return err
		}
	}
	var unsupported []string
	if len(o.stopSequences) > 0 {
		unsupported = append(unsupported, "WithStopSequences")
	}
	if o.seed != nil {
		unsupported = append(unsupported, "WithSeed")
	}
	if o.frequencyPenalty != nil {
		unsupported = append(unsupported, "WithFrequencyPenalty")
	}
	if o.presencePenalty != nil {
		unsupported = append(unsupported, "WithPresencePenalty")
	}
	if len(o.logitBias) > 0 {
		unsupported = append(unsupported, "WithLogitBias")
	}
	if o.n != nil {
		unsupported = append(unsupported, "WithN")
	}
	if o.topLogprobs != nil {
		unsupported = append(unsupported, "WithLogprobs")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf(
			"openai: %s not supported by the Responses API; "+
				"use NewChatCompletionsLLM for Chat Completions-only options",
			strings.Join(unsupported, ", "),
		)
	}
	return nil
}

// requestOptions returns per-call SDK options that inject request body fields
// the SDK has no typed parameter for. top_k is only sent to a custom base URL,
// because OpenAI's own API rejects it.
func (c *ResponsesClient) requestOptions() []option.RequestOption {
	var opts []option.RequestOption
	if c.options.topK != nil && c.options.baseURL != "" {
		opts = append(opts, option.WithJSONSet("top_k", *c.options.topK))
	}
	for k, v := range c.options.extraBodyFields {
		opts = append(opts, option.WithJSONSet(k, v))
	}
	if c.options.model.CanReason {
		for k, v := range c.options.reasoningBodyFields {
			opts = append(opts, option.WithJSONSet(k, v))
		}
	}
	for _, f := range c.options.modelBodyFields {
		if f.matches(c.options.model) {
			opts = append(opts, option.WithJSONSet(f.key, f.value))
		}
	}
	return opts
}
