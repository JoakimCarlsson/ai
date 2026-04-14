package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// OpenAIReasoningEffort controls reasoning depth for OpenAI o-series models.
type OpenAIReasoningEffort string

// OpenAIReasoningEffort values.
const (
	OpenAIReasoningEffortLow    OpenAIReasoningEffort = "low"
	OpenAIReasoningEffortMedium OpenAIReasoningEffort = "medium"
	OpenAIReasoningEffortHigh   OpenAIReasoningEffort = "high"
)

type openaiOptions struct {
	baseURL           string
	disableCache      bool
	reasoningEffort   *OpenAIReasoningEffort
	extraHeaders      map[string]string
	frequencyPenalty  *float64
	presencePenalty   *float64
	seed              *int64
	parallelToolCalls *bool
}

// OpenAIOption configures optional settings for OpenAI clients.
type OpenAIOption func(*openaiOptions)

type openaiClient struct {
	providerOptions llmClientOptions
	options         openaiOptions
	client          openai.Client
}

// OpenAIClient is the OpenAI Client implementation type.
type OpenAIClient Client

func newOpenAIClient(opts llmClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithAPIKey(opts.apiKey),
		)
	}
	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(
			openaiClientOptions,
			option.WithBaseURL(openaiOpts.baseURL),
		)
	}

	if openaiOpts.extraHeaders != nil {
		for key, value := range openaiOpts.extraHeaders {
			openaiClientOptions = append(
				openaiClientOptions,
				option.WithHeader(key, value),
			)
		}
	}

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          client,
	}
}

func (o *openaiClient) convertMessages(
	messages []message.Message,
) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			openaiMessages = append(
				openaiMessages,
				openai.SystemMessage(msg.Content().String()),
			)
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{
				Text: msg.Content().String(),
			}
			content = append(
				content,
				openai.ChatCompletionContentPartUnionParam{OfText: &textBlock},
			)

			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{
					URL: binaryContent.String(model.ProviderOpenAI),
				}
				imageBlock := openai.ChatCompletionContentPartImageParam{
					ImageURL: imageURL,
				}
				content = append(
					content,
					openai.ChatCompletionContentPartUnionParam{
						OfImageURL: &imageBlock,
					},
				)
			}

			for _, imageURLContent := range msg.ImageURLContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{
					URL: imageURLContent.URL,
				}
				if imageURLContent.Detail != "" {
					imageURL.Detail = imageURLContent.Detail
				}
				imageBlock := openai.ChatCompletionContentPartImageParam{
					ImageURL: imageURL,
				}
				content = append(
					content,
					openai.ChatCompletionContentPartUnionParam{
						OfImageURL: &imageBlock,
					},
				)
			}

			openaiMessages = append(openaiMessages, openai.UserMessage(content))

		case message.Assistant:
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make(
					[]openai.ChatCompletionMessageToolCallParam,
					len(msg.ToolCalls()),
				)
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID:   call.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					}
				}
			}

			openaiMessages = append(
				openaiMessages,
				openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				},
			)

		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (o *openaiClient) convertTools(
	tools []tool.BaseTool,
) []openai.ChatCompletionToolParam {
	openaiTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		params := openai.FunctionParameters{
			"type":       "object",
			"properties": info.Parameters,
		}
		if len(info.Required) > 0 {
			params["required"] = info.Required
		}
		openaiTools[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters:  params,
			},
		}
	}

	return openaiTools
}

func (o *openaiClient) finishReason(reason string) message.FinishReason {
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

func (o *openaiClient) preparedParams(
	messages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolParam,
) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.providerOptions.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}

	if o.options.parallelToolCalls != nil {
		params.ParallelToolCalls = openai.Bool(*o.options.parallelToolCalls)
	}

	paramBuilder := newParameterBuilder(o.providerOptions)
	paramBuilder.applyFloat64Temperature(
		func(t *float64) { params.Temperature = openai.Float(*t) },
	)
	paramBuilder.applyFloat64TopP(
		func(p *float64) { params.TopP = openai.Float(*p) },
	)

	if len(o.providerOptions.stopSequences) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfString: openai.String(o.providerOptions.stopSequences[0]),
		}
	}

	paramBuilder.applyFloat64FrequencyPenalty(
		o.options.frequencyPenalty,
		func(fp *float64) { params.FrequencyPenalty = openai.Float(*fp) },
	)
	paramBuilder.applyFloat64PresencePenalty(
		o.options.presencePenalty,
		func(pp *float64) { params.PresencePenalty = openai.Float(*pp) },
	)
	paramBuilder.applyInt64Seed(
		o.options.seed,
		func(s *int64) { params.Seed = openai.Int(*s) },
	)

	if o.providerOptions.model.CanReason {
		params.MaxCompletionTokens = openai.Int(o.providerOptions.maxTokens)
		if o.options.reasoningEffort != nil {
			switch *o.options.reasoningEffort {
			case OpenAIReasoningEffortLow:
				params.ReasoningEffort = shared.ReasoningEffortLow
			case OpenAIReasoningEffortMedium:
				params.ReasoningEffort = shared.ReasoningEffortMedium
			case OpenAIReasoningEffortHigh:
				params.ReasoningEffort = shared.ReasoningEffortHigh
			}
		}
	} else {
		params.MaxTokens = openai.Int(o.providerOptions.maxTokens)
	}

	return params
}

func (o *openaiClient) send(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (response *Response, err error) {
	params := o.preparedParams(
		o.convertMessages(messages),
		o.convertTools(tools),
	)

	ctx, cancel := withTimeout(ctx, o.providerOptions.timeout)
	defer cancel()

	return ExecuteWithRetry(
		ctx,
		OpenAIRetryConfig(),
		func() (*Response, error) {
			openaiResponse, err := o.client.Chat.Completions.New(ctx, params)
			if err != nil {
				return nil, err
			}

			if len(openaiResponse.Choices) == 0 {
				return nil, fmt.Errorf(
					"no response choices returned from OpenAI",
				)
			}

			content := ""
			if openaiResponse.Choices[0].Message.Content != "" {
				content = openaiResponse.Choices[0].Message.Content
			}

			toolCalls := o.toolCalls(*openaiResponse)
			finishReason := o.finishReason(
				string(openaiResponse.Choices[0].FinishReason),
			)

			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &Response{
				Content:      content,
				ToolCalls:    toolCalls,
				Usage:        o.usage(*openaiResponse),
				FinishReason: finishReason,
			}, nil
		},
	)
}

func (o *openaiClient) stream(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan Event {
	params := o.preparedParams(
		o.convertMessages(messages),
		o.convertTools(tools),
	)
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	ctx, cancel := withTimeout(ctx, o.providerOptions.timeout)
	defer cancel()

	eventChan := make(chan Event)

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, OpenAIRetryConfig(), func() error {
			openaiStream := o.client.Chat.Completions.NewStreaming(ctx, params)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			for openaiStream.Next() {
				chunk := openaiStream.Current()
				acc.AddChunk(chunk)

				for _, choice := range chunk.Choices {
					for _, key := range []string{"reasoning", "reasoning_content"} {
						if field, ok := choice.Delta.JSON.ExtraFields[key]; ok && field.Raw() != "" {
							var rc string
							if json.Unmarshal([]byte(field.Raw()), &rc) == nil && rc != "" {
								eventChan <- Event{
									Type:     types.EventThinkingDelta,
									Thinking: rc,
								}
							}
							break
						}
					}

					if choice.Delta.Content != "" {
						eventChan <- Event{
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
					eventChan <- Event{Type: types.EventError, Error: errors.New("no response choices in stream")}
					return errors.New("no response choices in stream")
				}
				finishReason := o.finishReason(
					string(acc.Choices[0].FinishReason),
				)
				if len(acc.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(
						toolCalls,
						o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- Event{
					Type: types.EventComplete,
					Response: &Response{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        o.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				return nil
			}
			return err
		}, eventChan)
	}()

	return eventChan
}

func (o *openaiClient) toolCalls(
	completion openai.ChatCompletion,
) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 &&
		len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (o *openaiClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0,
		CacheReadTokens:     cachedTokens,
	}
}

// WithOpenAIBaseURL sets a custom API endpoint for OpenAI-compatible services
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(options *openaiOptions) {
		options.baseURL = baseURL
	}
}

// WithOpenAIExtraHeaders adds custom HTTP headers to API requests
func WithOpenAIExtraHeaders(headers map[string]string) OpenAIOption {
	return func(options *openaiOptions) {
		options.extraHeaders = headers
	}
}

// WithOpenAIDisableCache disables response caching for OpenAI requests
func WithOpenAIDisableCache() OpenAIOption {
	return func(options *openaiOptions) {
		options.disableCache = true
	}
}

// WithReasoningEffort sets the reasoning effort level for OpenAI o-series models.
func WithReasoningEffort(effort OpenAIReasoningEffort) OpenAIOption {
	return func(options *openaiOptions) {
		options.reasoningEffort = &effort
	}
}

// WithOpenAIFrequencyPenalty sets the frequency penalty to reduce repetition in responses
func WithOpenAIFrequencyPenalty(frequencyPenalty float64) OpenAIOption {
	return func(options *openaiOptions) {
		options.frequencyPenalty = &frequencyPenalty
	}
}

// WithOpenAIPresencePenalty sets the presence penalty to encourage topic diversity
func WithOpenAIPresencePenalty(presencePenalty float64) OpenAIOption {
	return func(options *openaiOptions) {
		options.presencePenalty = &presencePenalty
	}
}

// WithOpenAISeed sets a random seed for deterministic response generation
func WithOpenAISeed(seed int64) OpenAIOption {
	return func(options *openaiOptions) {
		options.seed = &seed
	}
}

// WithOpenAIParallelToolCalls controls whether OpenAI can return multiple tool calls in a single response.
// Default is true (parallel tool calls enabled). Set to false to force sequential tool calls from the model.
func WithOpenAIParallelToolCalls(enabled bool) OpenAIOption {
	return func(options *openaiOptions) {
		options.parallelToolCalls = &enabled
	}
}

func (o *openaiClient) supportsStructuredOutput() bool {
	return o.providerOptions.model.SupportsStructuredOut
}

func (o *openaiClient) sendWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (response *Response, err error) {
	params := o.preparedParams(
		o.convertMessages(messages),
		o.convertTools(tools),
	)

	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}

	params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   outputSchema.Name,
				Schema: schemaMap,
				Strict: openai.Bool(true),
			},
		},
	}

	ctx, cancel := withTimeout(ctx, o.providerOptions.timeout)
	defer cancel()

	return ExecuteWithRetry(
		ctx,
		OpenAIRetryConfig(),
		func() (*Response, error) {
			openaiResponse, err := o.client.Chat.Completions.New(ctx, params)
			if err != nil {
				return nil, err
			}

			if len(openaiResponse.Choices) == 0 {
				return nil, fmt.Errorf(
					"no response choices returned from OpenAI",
				)
			}

			content := ""
			if openaiResponse.Choices[0].Message.Content != "" {
				content = openaiResponse.Choices[0].Message.Content
			}

			toolCalls := o.toolCalls(*openaiResponse)
			finishReason := o.finishReason(
				string(openaiResponse.Choices[0].FinishReason),
			)

			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &Response{
				Content:                    content,
				ToolCalls:                  toolCalls,
				Usage:                      o.usage(*openaiResponse),
				FinishReason:               finishReason,
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
			}, nil
		},
	)
}

func (o *openaiClient) streamWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan Event {
	params := o.preparedParams(
		o.convertMessages(messages),
		o.convertTools(tools),
	)

	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}

	params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   outputSchema.Name,
				Schema: schemaMap,
				Strict: openai.Bool(true),
			},
		},
	}

	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	ctx, cancel := withTimeout(ctx, o.providerOptions.timeout)
	defer cancel()

	eventChan := make(chan Event)

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, OpenAIRetryConfig(), func() error {
			openaiStream := o.client.Chat.Completions.NewStreaming(ctx, params)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			for openaiStream.Next() {
				chunk := openaiStream.Current()
				acc.AddChunk(chunk)

				for _, choice := range chunk.Choices {
					for _, key := range []string{"reasoning", "reasoning_content"} {
						if field, ok := choice.Delta.JSON.ExtraFields[key]; ok && field.Raw() != "" {
							var rc string
							if json.Unmarshal([]byte(field.Raw()), &rc) == nil && rc != "" {
								eventChan <- Event{
									Type:     types.EventThinkingDelta,
									Thinking: rc,
								}
							}
							break
						}
					}

					if choice.Delta.Content != "" {
						eventChan <- Event{
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
					eventChan <- Event{Type: types.EventError, Error: errors.New("no response choices in stream")}
					return errors.New("no response choices in stream")
				}
				finishReason := o.finishReason(
					string(acc.Choices[0].FinishReason),
				)
				if len(acc.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(
						toolCalls,
						o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- Event{
					Type: types.EventComplete,
					Response: &Response{
						Content:                    currentContent,
						ToolCalls:                  toolCalls,
						Usage:                      o.usage(acc.ChatCompletion),
						FinishReason:               finishReason,
						StructuredOutput:           &currentContent,
						UsedNativeStructuredOutput: true,
					},
				}
				return nil
			}
			return err
		}, eventChan)
	}()

	return eventChan
}
