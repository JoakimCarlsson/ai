// File: compound.go — Standalone client for Groq's compound models, which
// expose server-side built-in tools (browser_search, code_execution,
// visit_website). Compound features are not modeled by the OpenAI SDK; this
// client borrows the SDK for HTTP transport but builds requests and parses
// responses on its own. The thin OpenAI-compatible wrapper [NewLLM] remains
// the right choice for users who don't need compound features.
package groq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// BrowserSearchOpts configures Groq's browser_search built-in tool.
type BrowserSearchOpts struct {
	Country        string
	IncludeImages  bool
	AllowedDomains []string
	BlockedDomains []string
}

// CompoundOptions configures the Groq compound LLM client.
type CompoundOptions struct {
	apiKey        string
	model         model.Model
	maxTokens     int64
	temperature   *float64
	topP          *float64
	stopSequences []string
	timeout       *time.Duration
	baseURL       string
	extraHeaders  map[string]string
	builtinTools  []map[string]any
}

// CompoundOption configures [CompoundOptions].
type CompoundOption func(*CompoundOptions)

// WithCompoundAPIKey sets the API key used to authenticate.
func WithCompoundAPIKey(k string) CompoundOption {
	return func(o *CompoundOptions) { o.apiKey = k }
}

// WithCompoundModel selects a Groq compound model (e.g. groq/compound).
func WithCompoundModel(m model.Model) CompoundOption {
	return func(o *CompoundOptions) { o.model = m }
}

// WithCompoundMaxTokens sets the maximum number of tokens to generate.
func WithCompoundMaxTokens(n int64) CompoundOption {
	return func(o *CompoundOptions) { o.maxTokens = n }
}

// WithCompoundTemperature controls randomness.
func WithCompoundTemperature(t float64) CompoundOption {
	return func(o *CompoundOptions) { o.temperature = &t }
}

// WithCompoundTopP sets nucleus sampling probability mass.
func WithCompoundTopP(p float64) CompoundOption {
	return func(o *CompoundOptions) { o.topP = &p }
}

// WithCompoundStopSequences sets text sequences that halt generation.
func WithCompoundStopSequences(seqs ...string) CompoundOption {
	return func(o *CompoundOptions) { o.stopSequences = seqs }
}

// WithCompoundTimeout sets the maximum duration to wait for API responses.
func WithCompoundTimeout(d time.Duration) CompoundOption {
	return func(o *CompoundOptions) { o.timeout = &d }
}

// WithCompoundBaseURL overrides Groq's default API endpoint.
func WithCompoundBaseURL(u string) CompoundOption {
	return func(o *CompoundOptions) { o.baseURL = u }
}

// WithCompoundExtraHeaders adds custom HTTP headers to API requests.
func WithCompoundExtraHeaders(h map[string]string) CompoundOption {
	return func(o *CompoundOptions) { o.extraHeaders = h }
}

// WithBrowserSearch enables Groq's browser_search built-in tool. Requires a
// groq/compound* model passed to [WithCompoundModel].
func WithBrowserSearch(opts ...BrowserSearchOpts) CompoundOption {
	return func(o *CompoundOptions) {
		entry := map[string]any{"type": "browser_search"}
		if len(opts) > 0 {
			cfg := map[string]any{}
			if opts[0].Country != "" {
				cfg["country"] = opts[0].Country
			}
			if opts[0].IncludeImages {
				cfg["include_images"] = true
			}
			if len(opts[0].AllowedDomains) > 0 {
				cfg["allowed_domains"] = opts[0].AllowedDomains
			}
			if len(opts[0].BlockedDomains) > 0 {
				cfg["blocked_domains"] = opts[0].BlockedDomains
			}
			if len(cfg) > 0 {
				entry["browser_search"] = cfg
			}
		}
		o.builtinTools = append(o.builtinTools, entry)
	}
}

// WithCodeExecution enables Groq's code_execution built-in tool. Requires a
// groq/compound* model passed to [WithCompoundModel].
func WithCodeExecution() CompoundOption {
	return func(o *CompoundOptions) {
		o.builtinTools = append(
			o.builtinTools,
			map[string]any{"type": "code_interpreter"},
		)
	}
}

// WithVisitWebsite enables Groq's visit_website built-in tool. Requires a
// groq/compound* model passed to [WithCompoundModel].
func WithVisitWebsite() CompoundOption {
	return func(o *CompoundOptions) {
		o.builtinTools = append(
			o.builtinTools,
			map[string]any{"type": "visit_website"},
		)
	}
}

// compoundClient implements [llm.LLM] against Groq's compound chat-completions
// endpoint, with native support for browser_search / code_execution /
// visit_website built-in tools and parsing of executed_tools metadata.
type compoundClient struct {
	options CompoundOptions
	client  openaisdk.Client
}

// NewCompoundLLM constructs a Groq client purpose-built for compound models.
// Pair with [WithBrowserSearch], [WithCodeExecution], and [WithVisitWebsite]
// to enable Groq's server-side tools.
func NewCompoundLLM(opts ...CompoundOption) llm.LLM {
	options := CompoundOptions{baseURL: DefaultBaseURL}
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

	return llm.WithTracing(&compoundClient{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

func (c *compoundClient) Model() model.Model { return c.options.model }

func (c *compoundClient) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *compoundClient) convertMessages(
	messages []message.Message,
) []openaisdk.ChatCompletionMessageParamUnion {
	out := make([]openaisdk.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			out = append(out, openaisdk.SystemMessage(msg.Content().String()))
		case message.User:
			out = append(out, openaisdk.UserMessage(msg.Content().String()))
		case message.Assistant:
			am := openaisdk.ChatCompletionAssistantMessageParam{
				Content: openaisdk.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openaisdk.String(msg.Content().String()),
				},
			}
			if calls := msg.ToolCalls(); len(calls) > 0 {
				am.ToolCalls = make(
					[]openaisdk.ChatCompletionMessageToolCallUnionParam,
					len(calls),
				)
				for i, call := range calls {
					am.ToolCalls[i] = openaisdk.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openaisdk.ChatCompletionMessageFunctionToolCallParam{
							ID: call.ID,
							Function: openaisdk.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      call.Name,
								Arguments: call.Input,
							},
						},
					}
				}
			}
			out = append(
				out,
				openaisdk.ChatCompletionMessageParamUnion{OfAssistant: &am},
			)
		case message.Tool:
			for _, result := range msg.ToolResults() {
				out = append(
					out,
					openaisdk.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}
	return out
}

func (c *compoundClient) convertTools(
	tools []tool.BaseTool,
) []openaisdk.ChatCompletionToolUnionParam {
	out := make([]openaisdk.ChatCompletionToolUnionParam, len(tools))
	for i, t := range tools {
		info := t.Info()
		params := openaisdk.FunctionParameters{
			"type":       "object",
			"properties": info.Parameters,
		}
		if len(info.Required) > 0 {
			params["required"] = info.Required
		}
		out[i] = openaisdk.ChatCompletionToolUnionParam{
			OfFunction: &openaisdk.ChatCompletionFunctionToolParam{
				Function: openaisdk.FunctionDefinitionParam{
					Name:        info.Name,
					Description: openaisdk.String(info.Description),
					Parameters:  params,
				},
			},
		}
	}
	return out
}

func (c *compoundClient) preparedParams(
	messages []openaisdk.ChatCompletionMessageParamUnion,
	tools []openaisdk.ChatCompletionToolUnionParam,
) openaisdk.ChatCompletionNewParams {
	params := openaisdk.ChatCompletionNewParams{
		Model:    openaisdk.ChatModel(c.options.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}
	if c.options.maxTokens > 0 {
		params.MaxTokens = openaisdk.Int(c.options.maxTokens)
	}
	if c.options.temperature != nil {
		params.Temperature = openaisdk.Float(*c.options.temperature)
	}
	if c.options.topP != nil {
		params.TopP = openaisdk.Float(*c.options.topP)
	}
	if len(c.options.stopSequences) > 0 {
		params.Stop = openaisdk.ChatCompletionNewParamsStopUnion{
			OfString: openaisdk.String(c.options.stopSequences[0]),
		}
	}
	return params
}

// requestOptions returns per-call SDK options. When built-in tools are
// configured, a JSON overlay replaces the request's tools array with the
// merged set (typed function tools first, raw built-in entries appended).
func (c *compoundClient) requestOptions(
	tools []openaisdk.ChatCompletionToolUnionParam,
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

// executedTools walks the raw response JSON for any non-standard
// executed_tools field (Groq compound responses) and returns a flat slice
// suitable for [llm.Response.ProviderMetadata].
func executedTools(rawJSON string) []map[string]any {
	if rawJSON == "" {
		return nil
	}
	var root struct {
		Choices []struct {
			Message struct {
				ExecutedTools []map[string]any `json:"executed_tools"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &root); err != nil {
		return nil
	}
	if len(root.Choices) == 0 {
		return nil
	}
	return root.Choices[0].Message.ExecutedTools
}

func (c *compoundClient) toolCalls(
	completion openaisdk.ChatCompletion,
) []message.ToolCall {
	if len(completion.Choices) == 0 {
		return nil
	}
	calls := completion.Choices[0].Message.ToolCalls
	if len(calls) == 0 {
		return nil
	}
	out := make([]message.ToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, message.ToolCall{
			ID:       call.ID,
			Name:     call.Function.Name,
			Input:    call.Function.Arguments,
			Type:     "function",
			Finished: true,
		})
	}
	return out
}

func (c *compoundClient) usage(
	completion openaisdk.ChatCompletion,
) llm.TokenUsage {
	return llm.TokenUsage{
		InputTokens:  completion.Usage.PromptTokens,
		OutputTokens: completion.Usage.CompletionTokens,
	}
}

func (c *compoundClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (c *compoundClient) buildResponse(
	completion *openaisdk.ChatCompletion,
) *llm.Response {
	content := ""
	if len(completion.Choices) > 0 {
		content = completion.Choices[0].Message.Content
	}
	toolCalls := c.toolCalls(*completion)
	finishReason := message.FinishReasonUnknown
	if len(completion.Choices) > 0 {
		finishReason = c.finishReason(
			string(completion.Choices[0].FinishReason),
		)
	}
	if len(toolCalls) > 0 {
		finishReason = message.FinishReasonToolUse
	}

	var meta map[string]any
	if tools := executedTools(completion.RawJSON()); len(tools) > 0 {
		meta = map[string]any{"groq.executed_tools": tools}
	}

	return &llm.Response{
		Content:          content,
		ToolCalls:        toolCalls,
		Usage:            c.usage(*completion),
		FinishReason:     finishReason,
		ProviderMetadata: meta,
	}
}

// SendMessages sends a conversation and returns the complete response.
func (c *compoundClient) SendMessages(
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
			resp, err := c.client.Chat.Completions.New(ctx, params, reqOpts...)
			if err != nil {
				return nil, err
			}
			if len(resp.Choices) == 0 {
				return nil, errors.New("groq: no response choices returned")
			}
			return c.buildResponse(resp), nil
		},
	)
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *compoundClient) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	params.ResponseFormat = c.responseFormat(outputSchema)
	reqOpts := c.requestOptions(convertedTools)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		llm.DefaultRetryConfig(),
		func() (*llm.Response, error) {
			resp, err := c.client.Chat.Completions.New(ctx, params, reqOpts...)
			if err != nil {
				return nil, err
			}
			if len(resp.Choices) == 0 {
				return nil, errors.New("groq: no response choices returned")
			}
			out := c.buildResponse(resp)
			out.StructuredOutput = &out.Content
			out.UsedNativeStructuredOutput = true
			return out, nil
		},
	)
}

func (c *compoundClient) responseFormat(
	outputSchema *schema.StructuredOutputInfo,
) openaisdk.ChatCompletionNewParamsResponseFormatUnion {
	schemaMap := map[string]any{
		"type":       "object",
		"properties": outputSchema.Parameters,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}
	return openaisdk.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openaisdk.ResponseFormatJSONSchemaParam{
			JSONSchema: openaisdk.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   "structured_output",
				Schema: schemaMap,
				Strict: openaisdk.Bool(true),
			},
		},
	}
}

// StreamResponse sends a conversation and returns a channel of streaming events.
func (c *compoundClient) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	return c.runStream(ctx, messages, tools, nil)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *compoundClient) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return c.runStream(ctx, messages, tools, outputSchema)
}

func (c *compoundClient) runStream(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	convertedTools := c.convertTools(tools)
	params := c.preparedParams(c.convertMessages(messages), convertedTools)
	params.StreamOptions = openaisdk.ChatCompletionStreamOptionsParam{
		IncludeUsage: openaisdk.Bool(true),
	}
	if outputSchema != nil {
		params.ResponseFormat = c.responseFormat(outputSchema)
	}
	reqOpts := c.requestOptions(convertedTools)

	eventChan := make(chan llm.Event)
	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)

	go func() {
		defer close(eventChan)
		defer cancel()

		llm.ExecuteStreamWithRetry(ctx, llm.DefaultRetryConfig(), func() error {
			stream := c.client.Chat.Completions.NewStreaming(
				ctx,
				params,
				reqOpts...)
			acc := openaisdk.ChatCompletionAccumulator{}
			currentContent := ""

			eventChan <- llm.Event{Type: types.EventContentStart}

			for stream.Next() {
				chunk := stream.Current()
				acc.AddChunk(chunk)
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta.Content
					if delta != "" {
						currentContent += delta
						eventChan <- llm.Event{
							Type:    types.EventContentDelta,
							Content: delta,
						}
					}
				}
			}

			if err := stream.Err(); err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("groq stream: %w", err)
			}

			eventChan <- llm.Event{Type: types.EventContentStop}

			final := c.buildResponse(&acc.ChatCompletion)
			if outputSchema != nil {
				final.StructuredOutput = &currentContent
				final.UsedNativeStructuredOutput = true
			}
			eventChan <- llm.Event{Type: types.EventComplete, Response: final}
			return nil
		}, eventChan)
	}()

	return eventChan
}
