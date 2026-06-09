// Package anthropic provides an Anthropic implementation of the [llm.LLM] interface.
//
// The package supports both direct Anthropic API access and the Anthropic-on-Bedrock
// path (the [llm/bedrock] package wraps this one for that case). [WithBedrock] toggles
// the latter at construction time.
package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// ReasoningEffort controls thinking depth for Anthropic models.
type ReasoningEffort string

// ReasoningEffort values.
const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortMax    ReasoningEffort = "max"
)

// Options configures the Anthropic LLM client.
type Options struct {
	apiKey          string
	model           model.Model
	maxTokens       int64
	temperature     *float64
	topP            *float64
	topK            *int64
	stopSequences   []string
	timeout         *time.Duration
	useBedrock      bool
	disableCache    bool
	reasoningEffort *ReasoningEffort
	toolChoice      *llm.ToolChoice
	builtinTools    []anthropicsdk.ToolUnionParam
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Anthropic.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the LLM model.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithMaxTokens sets the maximum number of tokens to generate.
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

// WithBedrock configures the client to talk to Anthropic models hosted on AWS Bedrock.
func WithBedrock(useBedrock bool) Option {
	return func(o *Options) { o.useBedrock = useBedrock }
}

// WithDisableCache disables prompt caching for Anthropic requests.
func WithDisableCache() Option { return func(o *Options) { o.disableCache = true } }

// WithReasoningEffort sets the reasoning/thinking effort level.
func WithReasoningEffort(effort ReasoningEffort) Option {
	return func(o *Options) { o.reasoningEffort = &effort }
}

// WithToolChoice controls whether and which tool the model may call. It maps to
// Anthropic's tool_choice field: {"type":"auto"} / {"type":"none"} /
// {"type":"any"} / {"type":"tool","name":...}. The field is emitted only when
// tools are sent.
func WithToolChoice(choice llm.ToolChoice) Option {
	return func(o *Options) { o.toolChoice = &choice }
}

// WebSearchConfig configures the Anthropic server-side web_search tool.
type WebSearchConfig struct {
	MaxUses        int64
	AllowedDomains []string
	BlockedDomains []string
	UserLocation   *WebSearchUserLocation
}

// WebSearchUserLocation provides approximate user location for search relevance.
type WebSearchUserLocation struct {
	City     string
	Country  string
	Region   string
	Timezone string
}

// WithWebSearch enables Anthropic's server-side web_search tool. Each call to
// the API that triggers a search incurs Anthropic's per-search charge.
// Conversations containing web_search results are effectively single-turn:
// follow-up turns do not re-attach the server tool result blocks.
func WithWebSearch(cfg WebSearchConfig) Option {
	return func(o *Options) {
		p := anthropicsdk.WebSearchTool20250305Param{
			AllowedDomains: cfg.AllowedDomains,
			BlockedDomains: cfg.BlockedDomains,
		}
		if cfg.MaxUses > 0 {
			p.MaxUses = anthropicsdk.Int(cfg.MaxUses)
		}
		if cfg.UserLocation != nil {
			p.UserLocation = anthropicsdk.UserLocationParam{
				City:     optString(cfg.UserLocation.City),
				Country:  optString(cfg.UserLocation.Country),
				Region:   optString(cfg.UserLocation.Region),
				Timezone: optString(cfg.UserLocation.Timezone),
			}
		}
		o.builtinTools = append(o.builtinTools, anthropicsdk.ToolUnionParam{
			OfWebSearchTool20250305: &p,
		})
	}
}

func optString(s string) param.Opt[string] {
	if s == "" {
		return param.Opt[string]{}
	}
	return anthropicsdk.String(s)
}

// RetryConfig provides retry settings tuned for Anthropic API behavior.
func RetryConfig() llm.RetryConfig {
	cfg := llm.DefaultRetryConfig()
	cfg.RetryStatusCodes = []int{429, 529}
	return cfg
}

// retryableError wraps an Anthropic SDK error so the modality's retry helpers
// can dispatch via [llm.RetryableError]'s [errors.As] handling.
type retryableError struct {
	err *anthropicsdk.Error
}

func (e retryableError) Error() string      { return e.err.Error() }
func (e retryableError) Unwrap() error      { return e.err }
func (e retryableError) GetStatusCode() int { return e.err.StatusCode }
func (e retryableError) GetRetryAfter() string {
	if e.err.Response != nil {
		v := e.err.Response.Header.Values("Retry-After")
		if len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// wrapError converts an Anthropic SDK error into a [retryableError] so it
// satisfies [llm.RetryableError]; non-SDK errors pass through unchanged.
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *anthropicsdk.Error
	if errors.As(err, &sdkErr) {
		return retryableError{err: sdkErr}
	}
	return err
}

// Client implements [llm.LLM] against the Anthropic API.
type Client struct {
	options Options
	client  anthropicsdk.Client
}

// NewLLM constructs an Anthropic LLM client. The returned [llm.LLM] is wrapped
// with [llm.WithTracing], so callers always get tracing spans and metrics.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}
	if options.useBedrock {
		clientOpts = append(
			clientOpts,
			bedrock.WithLoadDefaultConfig(context.Background()),
		)
	}

	return llm.WithTracing(&Client{
		options: options,
		client:  anthropicsdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

// Model returns the configured LLM model.
func (c *Client) Model() model.Model { return c.options.model }

// SupportsStructuredOutput reports whether the configured model supports structured output.
func (c *Client) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *Client) convertMessages(
	messages []message.Message,
) (anthropicMessages []anthropicsdk.MessageParam, systemMessages []string) {
	for i, msg := range messages {
		cache := false
		if i == len(messages)-1 && !c.options.disableCache {
			cache = true
		}
		switch msg.Role {
		case message.System:
			systemMessages = append(systemMessages, msg.Content().String())
		case message.User:
			content := anthropicsdk.NewTextBlock(msg.Content().String())
			if cache {
				content.OfText.CacheControl = anthropicsdk.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			var contentBlocks []anthropicsdk.ContentBlockParamUnion
			contentBlocks = append(contentBlocks, content)

			for _, binaryContent := range msg.BinaryContent() {
				base64Image := binaryContent.String(model.ProviderAnthropic)
				imageBlock := anthropicsdk.NewImageBlockBase64(
					binaryContent.MIMEType,
					base64Image,
				)
				contentBlocks = append(contentBlocks, imageBlock)
			}

			for _, imageURLContent := range msg.ImageURLContent() {
				imageBlock := anthropicsdk.NewImageBlock(
					anthropicsdk.URLImageSourceParam{
						Type: "url",
						URL:  imageURLContent.URL,
					},
				)
				contentBlocks = append(contentBlocks, imageBlock)
			}

			anthropicMessages = append(
				anthropicMessages,
				anthropicsdk.NewUserMessage(contentBlocks...),
			)

		case message.Assistant:
			blocks := []anthropicsdk.ContentBlockParamUnion{}
			if msg.Content().String() != "" {
				blocks = append(
					blocks,
					anthropicsdk.NewTextBlock(msg.Content().String()),
				)
			}

			for _, toolCall := range msg.ToolCalls() {
				var inputMap map[string]any
				if err := json.Unmarshal([]byte(toolCall.Input), &inputMap); err != nil {
					continue
				}
				blocks = append(blocks, anthropicsdk.NewToolUseBlock(
					toolCall.ID, inputMap, toolCall.Name,
				))
			}

			if len(blocks) == 0 {
				slog.Warn(
					"There is a message without content, investigate, this should not happen",
				)
				continue
			}
			anthropicMessages = append(
				anthropicMessages,
				anthropicsdk.NewAssistantMessage(blocks...),
			)

		case message.Tool:
			results := make(
				[]anthropicsdk.ContentBlockParamUnion,
				len(msg.ToolResults()),
			)
			for i, toolResult := range msg.ToolResults() {
				results[i] = anthropicsdk.NewToolResultBlock(
					toolResult.ToolCallID,
					toolResult.Content,
					toolResult.IsError,
				)
			}
			anthropicMessages = append(
				anthropicMessages,
				anthropicsdk.NewUserMessage(results...),
			)
		}
	}
	return
}

func (c *Client) convertTools(
	tools []tool.BaseTool,
) []anthropicsdk.ToolUnionParam {
	out := make([]anthropicsdk.ToolUnionParam, len(tools))

	for i, t := range tools {
		info := t.Info()
		toolParam := anthropicsdk.ToolParam{
			Name:        info.Name,
			Description: anthropicsdk.String(info.Description),
			InputSchema: anthropicsdk.ToolInputSchemaParam{
				Properties: info.Parameters,
			},
		}

		if i == len(tools)-1 && !c.options.disableCache {
			toolParam.CacheControl = anthropicsdk.CacheControlEphemeralParam{
				Type: "ephemeral",
			}
		}

		out[i] = anthropicsdk.ToolUnionParam{OfTool: &toolParam}
	}

	return append(out, c.options.builtinTools...)
}

// toolChoiceParam maps a vendor-neutral [llm.ToolChoice] to Anthropic's
// tool_choice union: {"type":"auto"} / {"type":"none"} / {"type":"any"} for the
// required case, or {"type":"tool","name":...} for [llm.ToolChoiceSpecific].
func toolChoiceParam(choice llm.ToolChoice) anthropicsdk.ToolChoiceUnionParam {
	switch choice.Mode {
	case llm.ToolChoiceNone:
		none := anthropicsdk.NewToolChoiceNoneParam()
		return anthropicsdk.ToolChoiceUnionParam{OfNone: &none}
	case llm.ToolChoiceRequired:
		return anthropicsdk.ToolChoiceUnionParam{
			OfAny: &anthropicsdk.ToolChoiceAnyParam{},
		}
	case llm.ToolChoiceSpecific:
		return anthropicsdk.ToolChoiceParamOfTool(choice.Name)
	default:
		return anthropicsdk.ToolChoiceUnionParam{
			OfAuto: &anthropicsdk.ToolChoiceAutoParam{},
		}
	}
}

func (c *Client) finishReason(reason string) message.FinishReason {
	switch reason {
	case "end_turn":
		return message.FinishReasonEndTurn
	case "max_tokens":
		return message.FinishReasonMaxTokens
	case "tool_use":
		return message.FinishReasonToolUse
	case "stop_sequence":
		return message.FinishReasonEndTurn
	default:
		return message.FinishReasonUnknown
	}
}

func usesLegacyExtendedThinking(apiModel string) bool {
	for _, legacy := range []string{
		"claude-sonnet-4-20",
		"claude-opus-4-20",
		"claude-sonnet-4-5",
		"claude-opus-4-5",
		"claude-haiku-4-5",
	} {
		if strings.HasPrefix(apiModel, legacy) {
			return true
		}
	}
	return false
}

func (c *Client) preparedMessages(
	messages []anthropicsdk.MessageParam,
	tools []anthropicsdk.ToolUnionParam,
	systemMessages []string,
) anthropicsdk.MessageNewParams {
	var thinkingParam anthropicsdk.ThinkingConfigParamUnion
	var outputConfig anthropicsdk.OutputConfigParam
	var temperature param.Opt[float64]
	pb := llm.NewParameterBuilder(
		c.options.temperature,
		c.options.topP,
		c.options.topK,
	)
	pb.ApplyFloat64Temperature(
		func(t *float64) { temperature = anthropicsdk.Float(*t) },
	)

	if c.options.reasoningEffort != nil && c.options.model.CanReason {
		if usesLegacyExtendedThinking(c.options.model.APIModel) {
			temperature = anthropicsdk.Float(1)
			thinkingParam = anthropicsdk.ThinkingConfigParamUnion{
				OfEnabled: &anthropicsdk.ThinkingConfigEnabledParam{
					BudgetTokens: int64(float64(c.options.maxTokens) * 0.8),
				},
			}
		} else {
			thinkingParam = anthropicsdk.ThinkingConfigParamUnion{
				OfAdaptive: &anthropicsdk.ThinkingConfigAdaptiveParam{},
			}
			switch *c.options.reasoningEffort {
			case ReasoningEffortLow:
				outputConfig.Effort = anthropicsdk.OutputConfigEffortLow
			case ReasoningEffortMedium:
				outputConfig.Effort = anthropicsdk.OutputConfigEffortMedium
			case ReasoningEffortHigh:
				outputConfig.Effort = anthropicsdk.OutputConfigEffortHigh
			case ReasoningEffortMax:
				outputConfig.Effort = anthropicsdk.OutputConfigEffortMax
			}
		}
	}

	maxTokens := c.options.maxTokens
	if maxTokens == 0 {
		maxTokens = c.options.model.DefaultMaxTokens
	}

	params := anthropicsdk.MessageNewParams{
		Model:        anthropicsdk.Model(c.options.model.APIModel),
		MaxTokens:    maxTokens,
		Temperature:  temperature,
		Messages:     messages,
		Tools:        tools,
		Thinking:     thinkingParam,
		OutputConfig: outputConfig,
	}

	pb.ApplyFloat64TopP(
		func(p *float64) { params.TopP = anthropicsdk.Float(*p) },
	)
	pb.ApplyInt64TopK(func(k *int64) { params.TopK = anthropicsdk.Int(*k) })

	if len(c.options.stopSequences) > 0 {
		params.StopSequences = c.options.stopSequences
	}

	if c.options.toolChoice != nil && len(tools) > 0 {
		params.ToolChoice = toolChoiceParam(*c.options.toolChoice)
	}

	if len(systemMessages) > 0 {
		systemBlocks := make([]anthropicsdk.TextBlockParam, len(systemMessages))
		for i, sysMsg := range systemMessages {
			block := anthropicsdk.TextBlockParam{Text: sysMsg}
			if i == len(systemMessages)-1 && !c.options.disableCache {
				block.CacheControl = anthropicsdk.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			systemBlocks[i] = block
		}
		params.System = systemBlocks
	}

	return params
}

// validateToolChoice rejects a malformed tool choice before a request is sent.
func (c *Client) validateToolChoice() error {
	if c.options.toolChoice == nil {
		return nil
	}
	return c.options.toolChoice.Validate()
}

// errorEvent returns a closed channel carrying a single error event, used to
// surface pre-flight failures (such as an invalid tool choice) on the streaming
// API where the method signature has no error return.
func errorEvent(err error) <-chan llm.Event {
	eventChan := make(chan llm.Event, 1)
	eventChan <- llm.Event{Type: types.EventError, Error: err}
	close(eventChan)
	return eventChan
}

// SendMessages sends a conversation and returns the complete response.
func (c *Client) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	if err := c.validateToolChoice(); err != nil {
		return nil, err
	}
	anthropicMessages, systemMessages := c.convertMessages(messages)
	preparedMessages := c.preparedMessages(
		anthropicMessages, c.convertTools(tools), systemMessages,
	)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			var raw *http.Response
			anthropicResponse, err := c.client.Messages.New(
				ctx,
				preparedMessages,
				option.WithResponseInto(&raw),
			)
			if err != nil {
				return nil, wrapError(err)
			}

			content, meta := c.extractContent(*anthropicResponse)
			resp := &llm.Response{
				Content:   content,
				ToolCalls: c.toolCalls(*anthropicResponse),
				Usage:     c.usage(*anthropicResponse),
				FinishReason: c.finishReason(
					string(anthropicResponse.StopReason),
				),
				ProviderMetadata: meta,
			}
			applyResponseHeaders(resp, raw)
			return resp, nil
		},
	)
}

// applyResponseHeaders lifts the provider request id and selected response
// headers from a captured raw HTTP response onto resp. It is a no-op when the
// response was not captured (raw is nil).
func applyResponseHeaders(resp *llm.Response, raw *http.Response) {
	if resp == nil || raw == nil {
		return
	}
	resp.RequestID, resp.ResponseHeaders = llm.SelectResponseHeaders(raw.Header)
}

// StreamResponse sends a conversation and returns a channel of streaming events.
func (c *Client) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	if err := c.validateToolChoice(); err != nil {
		return errorEvent(err)
	}
	anthropicMessages, systemMessages := c.convertMessages(messages)
	preparedMessages := c.preparedMessages(
		anthropicMessages, c.convertTools(tools), systemMessages,
	)
	eventChan := make(chan llm.Event)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	go func() {
		defer close(eventChan)
		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			return c.runStream(ctx, preparedMessages, eventChan, false)
		}, eventChan)
	}()
	return eventChan
}

func (c *Client) runStream(
	ctx context.Context,
	preparedMessages anthropicsdk.MessageNewParams,
	eventChan chan<- llm.Event,
	structured bool,
) error {
	var raw *http.Response
	anthropicStream := c.client.Messages.NewStreaming(
		ctx, preparedMessages, option.WithResponseInto(&raw),
	)
	accumulatedMessage := anthropicsdk.Message{}

	currentBlockType := ""
	currentToolCallID := ""
	for anthropicStream.Next() {
		event := anthropicStream.Current()
		if err := accumulatedMessage.Accumulate(event); err != nil {
			slog.Warn("Error accumulating message", "error", err)
			continue
		}

		switch event := event.AsAny().(type) {
		case anthropicsdk.ContentBlockStartEvent:
			currentBlockType = event.ContentBlock.Type
			switch event.ContentBlock.Type {
			case "text":
				eventChan <- llm.Event{Type: types.EventContentStart}
			case "tool_use":
				currentToolCallID = event.ContentBlock.ID
				eventChan <- llm.Event{
					Type: types.EventToolUseStart,
					ToolCall: &message.ToolCall{
						ID:       event.ContentBlock.ID,
						Name:     event.ContentBlock.Name,
						Finished: false,
					},
				}
			}

		case anthropicsdk.ContentBlockDeltaEvent:
			switch event.Delta.Type {
			case "thinking_delta":
				if event.Delta.Thinking != "" {
					eventChan <- llm.Event{
						Type:     types.EventThinkingDelta,
						Thinking: event.Delta.Thinking,
					}
				}
			case "text_delta":
				if event.Delta.Text != "" {
					eventChan <- llm.Event{
						Type:    types.EventContentDelta,
						Content: event.Delta.Text,
					}
				}
			case "input_json_delta":
				if currentToolCallID != "" {
					eventChan <- llm.Event{
						Type: types.EventToolUseDelta,
						ToolCall: &message.ToolCall{
							ID:       currentToolCallID,
							Finished: false,
							Input:    event.Delta.JSON.PartialJSON.Raw(),
						},
					}
				}
			}
		case anthropicsdk.ContentBlockStopEvent:
			switch currentBlockType {
			case "tool_use":
				eventChan <- llm.Event{
					Type:     types.EventToolUseStop,
					ToolCall: &message.ToolCall{ID: currentToolCallID},
				}
			case "text":
				eventChan <- llm.Event{Type: types.EventContentStop}
			}
			currentBlockType = ""
			currentToolCallID = ""

		case anthropicsdk.MessageStopEvent:
			content, meta := c.extractContent(accumulatedMessage)
			resp := &llm.Response{
				Content:          content,
				ToolCalls:        c.toolCalls(accumulatedMessage),
				Usage:            c.usage(accumulatedMessage),
				FinishReason:     c.finishReason(string(accumulatedMessage.StopReason)),
				ProviderMetadata: meta,
			}
			applyResponseHeaders(resp, raw)
			if structured {
				resp.StructuredOutput = &content
				resp.UsedNativeStructuredOutput = true
			}
			eventChan <- llm.Event{Type: types.EventComplete, Response: resp}
		}
	}

	if err := anthropicStream.Err(); err != nil && !errors.Is(err, io.EOF) {
		return wrapError(err)
	}
	return nil
}

// extractContent walks an Anthropic response and returns the concatenated
// assistant text plus any provider metadata from server-side built-in tools.
func (c *Client) extractContent(
	msg anthropicsdk.Message,
) (string, map[string]any) {
	var content string
	var searchResults []map[string]any
	for _, block := range msg.Content {
		switch v := block.AsAny().(type) {
		case anthropicsdk.TextBlock:
			content += v.Text
		case anthropicsdk.WebSearchToolResultBlock:
			results := v.Content.AsWebSearchResultBlockArray()
			for _, r := range results {
				searchResults = append(searchResults, map[string]any{
					"tool_use_id":       v.ToolUseID,
					"url":               r.URL,
					"title":             r.Title,
					"page_age":          r.PageAge,
					"encrypted_content": r.EncryptedContent,
				})
			}
		}
	}
	var meta map[string]any
	if len(searchResults) > 0 {
		meta = map[string]any{"anthropic.web_search_results": searchResults}
	}
	return content, meta
}

func (c *Client) toolCalls(msg anthropicsdk.Message) []message.ToolCall {
	var toolCalls []message.ToolCall
	for _, block := range msg.Content {
		if variant, ok := block.AsAny().(anthropicsdk.ToolUseBlock); ok {
			toolCalls = append(toolCalls, message.ToolCall{
				ID:       variant.ID,
				Name:     variant.Name,
				Input:    string(variant.Input),
				Type:     string(variant.Type),
				Finished: true,
			})
		}
	}
	return toolCalls
}

func (c *Client) usage(msg anthropicsdk.Message) llm.TokenUsage {
	return llm.TokenUsage{
		InputTokens:         msg.Usage.InputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
	}
}

func (c *Client) buildOutputConfig(
	outputSchema *schema.StructuredOutputInfo,
) anthropicsdk.OutputConfigParam {
	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}
	return anthropicsdk.OutputConfigParam{
		Format: anthropicsdk.JSONOutputFormatParam{Schema: schemaMap},
	}
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *Client) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	if err := c.validateToolChoice(); err != nil {
		return nil, err
	}
	anthropicMessages, systemMessages := c.convertMessages(messages)
	preparedMessages := c.preparedMessages(
		anthropicMessages, c.convertTools(tools), systemMessages,
	)
	preparedMessages.OutputConfig = c.buildOutputConfig(outputSchema)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			var raw *http.Response
			anthropicResponse, err := c.client.Messages.New(
				ctx,
				preparedMessages,
				option.WithResponseInto(&raw),
			)
			if err != nil {
				return nil, wrapError(err)
			}

			content, meta := c.extractContent(*anthropicResponse)
			resp := &llm.Response{
				Content:   content,
				ToolCalls: c.toolCalls(*anthropicResponse),
				Usage:     c.usage(*anthropicResponse),
				FinishReason: c.finishReason(
					string(anthropicResponse.StopReason),
				),
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
				ProviderMetadata:           meta,
			}
			applyResponseHeaders(resp, raw)
			return resp, nil
		},
	)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *Client) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	if err := c.validateToolChoice(); err != nil {
		return errorEvent(err)
	}
	anthropicMessages, systemMessages := c.convertMessages(messages)
	preparedMessages := c.preparedMessages(
		anthropicMessages, c.convertTools(tools), systemMessages,
	)
	preparedMessages.OutputConfig = c.buildOutputConfig(outputSchema)

	eventChan := make(chan llm.Event)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	go func() {
		defer close(eventChan)
		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			return c.runStream(ctx, preparedMessages, eventChan, true)
		}, eventChan)
	}()
	return eventChan
}
