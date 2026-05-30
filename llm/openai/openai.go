// Package openai provides an OpenAI implementation of the [llm.LLM] interface.
//
// This package also serves OpenAI-compatible providers (Groq, OpenRouter, xAI,
// Mistral, Ollama, etc.) — point [WithBaseURL] at the appropriate endpoint.
// The [llm/azure] package wraps this one for Azure OpenAI.
package openai

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
	"github.com/openai/openai-go/v3/shared"
)

// ReasoningEffort controls reasoning depth for OpenAI o-series models.
type ReasoningEffort string

// ReasoningEffort values.
const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
)

// Options configures the OpenAI LLM client.
type Options struct {
	apiKey            string
	model             model.Model
	maxTokens         int64
	temperature       *float64
	topP              *float64
	topK              *int64
	stopSequences     []string
	timeout           *time.Duration
	baseURL           string
	disableCache      bool
	reasoningEffort   *ReasoningEffort
	extraHeaders      map[string]string
	frequencyPenalty  *float64
	presencePenalty   *float64
	seed              *int64
	parallelToolCalls *bool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate.
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

// WithTopK limits token selection to the top K candidates. OpenAI and Azure
// reject top_k, so it is sent only when a custom base URL targets an
// OpenAI-compatible provider that accepts it (Together, OpenRouter, Fireworks,
// ...); otherwise it has no effect. See requestOptions.
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

// WithBaseURL sets a custom API endpoint for OpenAI-compatible services.
func WithBaseURL(
	baseURL string,
) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithExtraHeaders adds custom HTTP headers to API requests.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// WithDisableCache disables response caching for OpenAI requests.
func WithDisableCache() Option { return func(o *Options) { o.disableCache = true } }

// WithReasoningEffort sets the reasoning effort level for OpenAI o-series models.
func WithReasoningEffort(effort ReasoningEffort) Option {
	return func(o *Options) { o.reasoningEffort = &effort }
}

// WithFrequencyPenalty sets the frequency penalty.
func WithFrequencyPenalty(
	p float64,
) Option {
	return func(o *Options) { o.frequencyPenalty = &p }
}

// WithPresencePenalty sets the presence penalty.
func WithPresencePenalty(
	p float64,
) Option {
	return func(o *Options) { o.presencePenalty = &p }
}

// WithSeed sets a random seed for deterministic generation.
func WithSeed(seed int64) Option { return func(o *Options) { o.seed = &seed } }

// WithParallelToolCalls toggles whether OpenAI returns multiple tool calls in a single response.
func WithParallelToolCalls(enabled bool) Option {
	return func(o *Options) { o.parallelToolCalls = &enabled }
}

// RetryConfig provides retry settings tuned for OpenAI API behavior.
func RetryConfig() llm.RetryConfig {
	cfg := llm.DefaultRetryConfig()
	cfg.RetryStatusCodes = []int{429, 500}
	return cfg
}

// retryableError wraps an OpenAI SDK error so the modality's retry helpers
// can dispatch via [llm.RetryableError]'s [errors.As] handling.
type retryableError struct {
	err *openaisdk.Error
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

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *openaisdk.Error
	if errors.As(err, &sdkErr) {
		return retryableError{err: sdkErr}
	}
	return err
}

// Client implements [llm.LLM] against the OpenAI API.
type Client struct {
	options Options
	client  openaisdk.Client
}

// NewLLM constructs an OpenAI LLM client. The returned [llm.LLM] is wrapped
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
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}
	for k, v := range options.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(k, v))
	}

	return llm.WithTracing(&Client{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

// NewWithExistingClient is for embedding by other packages (e.g. llm/azure) that
// build the OpenAI SDK client themselves and want this package's request logic.
// The returned *Client is the bare implementation, not wrapped in tracing.
func NewWithExistingClient(options Options, client openaisdk.Client) *Client {
	return &Client{options: options, client: client}
}

// Model returns the configured LLM model.
func (c *Client) Model() model.Model { return c.options.model }

// SupportsStructuredOutput reports whether the configured model supports structured output.
func (c *Client) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *Client) convertMessages(
	messages []message.Message,
) (openaiMessages []openaisdk.ChatCompletionMessageParamUnion) {
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			openaiMessages = append(
				openaiMessages,
				openaisdk.SystemMessage(msg.Content().String()),
			)
		case message.User:
			var content []openaisdk.ChatCompletionContentPartUnionParam
			textBlock := openaisdk.ChatCompletionContentPartTextParam{
				Text: msg.Content().String(),
			}
			content = append(
				content,
				openaisdk.ChatCompletionContentPartUnionParam{
					OfText: &textBlock,
				},
			)

			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openaisdk.ChatCompletionContentPartImageImageURLParam{
					URL: binaryContent.String(model.ProviderOpenAI),
				}
				imageBlock := openaisdk.ChatCompletionContentPartImageParam{
					ImageURL: imageURL,
				}
				content = append(
					content,
					openaisdk.ChatCompletionContentPartUnionParam{
						OfImageURL: &imageBlock,
					},
				)
			}

			for _, imageURLContent := range msg.ImageURLContent() {
				imageURL := openaisdk.ChatCompletionContentPartImageImageURLParam{
					URL: imageURLContent.URL,
				}
				if imageURLContent.Detail != "" {
					imageURL.Detail = imageURLContent.Detail
				}
				imageBlock := openaisdk.ChatCompletionContentPartImageParam{
					ImageURL: imageURL,
				}
				content = append(
					content,
					openaisdk.ChatCompletionContentPartUnionParam{
						OfImageURL: &imageBlock,
					},
				)
			}

			openaiMessages = append(
				openaiMessages,
				openaisdk.UserMessage(content),
			)

		case message.Assistant:
			assistantMsg := openaisdk.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openaisdk.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make(
					[]openaisdk.ChatCompletionMessageToolCallUnionParam,
					len(msg.ToolCalls()),
				)
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openaisdk.ChatCompletionMessageToolCallUnionParam{
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

			openaiMessages = append(
				openaiMessages,
				openaisdk.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				},
			)

		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages,
					openaisdk.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (c *Client) convertTools(
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

func (c *Client) finishReason(reason string) message.FinishReason {
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

func (c *Client) preparedParams(
	messages []openaisdk.ChatCompletionMessageParamUnion,
	tools []openaisdk.ChatCompletionToolUnionParam,
) openaisdk.ChatCompletionNewParams {
	params := openaisdk.ChatCompletionNewParams{
		Model:    openaisdk.ChatModel(c.options.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}

	if c.options.parallelToolCalls != nil {
		params.ParallelToolCalls = openaisdk.Bool(*c.options.parallelToolCalls)
	}

	pb := llm.NewParameterBuilder(
		c.options.temperature,
		c.options.topP,
		nil,
	)
	pb.ApplyFloat64Temperature(
		func(t *float64) { params.Temperature = openaisdk.Float(*t) },
	)
	pb.ApplyFloat64TopP(func(p *float64) { params.TopP = openaisdk.Float(*p) })

	if len(c.options.stopSequences) > 0 {
		stops := c.options.stopSequences
		if len(stops) > 4 {
			stops = stops[:4]
		}
		params.Stop = openaisdk.ChatCompletionNewParamsStopUnion{
			OfStringArray: stops,
		}
	}

	pb.ApplyFloat64FrequencyPenalty(c.options.frequencyPenalty,
		func(fp *float64) { params.FrequencyPenalty = openaisdk.Float(*fp) })
	pb.ApplyFloat64PresencePenalty(c.options.presencePenalty,
		func(pp *float64) { params.PresencePenalty = openaisdk.Float(*pp) })
	pb.ApplyInt64Seed(c.options.seed,
		func(s *int64) { params.Seed = openaisdk.Int(*s) })

	if c.options.maxTokens > 0 {
		params.MaxCompletionTokens = openaisdk.Int(c.options.maxTokens)
	}
	if c.options.model.CanReason && c.options.reasoningEffort != nil {
		switch *c.options.reasoningEffort {
		case ReasoningEffortLow:
			params.ReasoningEffort = shared.ReasoningEffortLow
		case ReasoningEffortMedium:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case ReasoningEffortHigh:
			params.ReasoningEffort = shared.ReasoningEffortHigh
		}
	}

	return params
}

// requestOptions returns per-call SDK request options derived from Options.
//
// The OpenAI Go SDK has no native top_k field on ChatCompletionNewParams. When
// WithTopK is set, the value is injected directly into the request body, but
// only on the OpenAI-compatible path (when a custom base URL is configured):
// OpenAI's and Azure's own APIs reject top_k as an unrecognized argument
// (HTTP 400), whereas compatible providers that accept it — Together,
// OpenRouter, Fireworks, ... — honor it. Without a custom base URL the target
// is OpenAI/Azure proper, so top_k is omitted rather than triggering a 400.
func (c *Client) requestOptions() []option.RequestOption {
	if c.options.topK == nil || c.options.baseURL == "" {
		return nil
	}
	return []option.RequestOption{option.WithJSONSet("top_k", *c.options.topK)}
}

// SendMessages sends a conversation and returns the complete response.
func (c *Client) SendMessages(
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
			openaiResponse, err := c.client.Chat.Completions.New(
				ctx,
				params,
				c.requestOptions()...)
			if err != nil {
				return nil, wrapError(err)
			}

			if len(openaiResponse.Choices) == 0 {
				return nil, fmt.Errorf(
					"no response choices returned from OpenAI",
				)
			}

			content := openaiResponse.Choices[0].Message.Content
			toolCalls := c.toolCalls(*openaiResponse)
			finishReason := c.finishReason(
				string(openaiResponse.Choices[0].FinishReason),
			)
			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &llm.Response{
				Content:      content,
				ToolCalls:    toolCalls,
				Usage:        c.usage(*openaiResponse),
				FinishReason: finishReason,
			}, nil
		},
	)
}

// StreamResponse sends a conversation and returns a channel of streaming events.
func (c *Client) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.StreamOptions = openaisdk.ChatCompletionStreamOptionsParam{
		IncludeUsage: openaisdk.Bool(true),
	}

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	eventChan := make(chan llm.Event)

	go func() {
		defer close(eventChan)
		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			return c.runStream(ctx, params, eventChan, false)
		}, eventChan)
	}()

	return eventChan
}

func (c *Client) runStream(
	ctx context.Context,
	params openaisdk.ChatCompletionNewParams,
	eventChan chan<- llm.Event,
	structured bool,
) error {
	openaiStream := c.client.Chat.Completions.NewStreaming(
		ctx,
		params,
		c.requestOptions()...)

	acc := openaisdk.ChatCompletionAccumulator{}
	currentContent := ""
	toolCalls := make([]message.ToolCall, 0)

	for openaiStream.Next() {
		chunk := openaiStream.Current()
		acc.AddChunk(chunk)

		for _, choice := range chunk.Choices {
			for _, key := range []string{"reasoning", "reasoning_content"} {
				if field, ok := choice.Delta.JSON.ExtraFields[key]; ok &&
					field.Raw() != "" {
					var rc string
					if json.Unmarshal([]byte(field.Raw()), &rc) == nil &&
						rc != "" {
						eventChan <- llm.Event{
							Type:     types.EventThinkingDelta,
							Thinking: rc,
						}
					}
					break
				}
			}

			if choice.Delta.Content != "" {
				eventChan <- llm.Event{
					Type:    types.EventContentDelta,
					Content: choice.Delta.Content,
				}
				currentContent += choice.Delta.Content
			}
		}
	}

	err := openaiStream.Err()
	if err == nil || errors.Is(err, io.EOF) {
		if len(acc.Choices) == 0 {
			eventChan <- llm.Event{Type: types.EventError, Error: errors.New("no response choices in stream")}
			return errors.New("no response choices in stream")
		}
		finishReason := c.finishReason(string(acc.Choices[0].FinishReason))
		if len(acc.Choices[0].Message.ToolCalls) > 0 {
			toolCalls = append(toolCalls, c.toolCalls(acc.ChatCompletion)...)
		}
		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		resp := &llm.Response{
			Content:      currentContent,
			ToolCalls:    toolCalls,
			Usage:        c.usage(acc.ChatCompletion),
			FinishReason: finishReason,
		}
		if structured {
			resp.StructuredOutput = &currentContent
			resp.UsedNativeStructuredOutput = true
		}
		eventChan <- llm.Event{Type: types.EventComplete, Response: resp}
		return nil
	}
	return wrapError(err)
}

func (c *Client) toolCalls(
	completion openaisdk.ChatCompletion,
) []message.ToolCall {
	var toolCalls []message.ToolCall
	if len(completion.Choices) > 0 &&
		len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			})
		}
	}
	return toolCalls
}

func (c *Client) usage(completion openaisdk.ChatCompletion) llm.TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return llm.TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0,
		CacheReadTokens:     cachedTokens,
	}
}

func (c *Client) responseFormatForSchema(
	outputSchema *schema.StructuredOutputInfo,
) openaisdk.ChatCompletionNewParamsResponseFormatUnion {
	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}

	return openaisdk.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openaisdk.ResponseFormatJSONSchemaParam{
			JSONSchema: openaisdk.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   outputSchema.Name,
				Schema: schemaMap,
				Strict: openaisdk.Bool(true),
			},
		},
	}
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *Client) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.ResponseFormat = c.responseFormatForSchema(outputSchema)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			openaiResponse, err := c.client.Chat.Completions.New(
				ctx,
				params,
				c.requestOptions()...)
			if err != nil {
				return nil, wrapError(err)
			}

			if len(openaiResponse.Choices) == 0 {
				return nil, fmt.Errorf(
					"no response choices returned from OpenAI",
				)
			}

			content := openaiResponse.Choices[0].Message.Content
			toolCalls := c.toolCalls(*openaiResponse)
			finishReason := c.finishReason(
				string(openaiResponse.Choices[0].FinishReason),
			)
			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &llm.Response{
				Content:                    content,
				ToolCalls:                  toolCalls,
				Usage:                      c.usage(*openaiResponse),
				FinishReason:               finishReason,
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
			}, nil
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
	params := c.preparedParams(
		c.convertMessages(messages),
		c.convertTools(tools),
	)
	params.ResponseFormat = c.responseFormatForSchema(outputSchema)
	params.StreamOptions = openaisdk.ChatCompletionStreamOptionsParam{
		IncludeUsage: openaisdk.Bool(true),
	}

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	eventChan := make(chan llm.Event)

	go func() {
		defer close(eventChan)
		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			return c.runStream(ctx, params, eventChan, true)
		}, eventChan)
	}()

	return eventChan
}
