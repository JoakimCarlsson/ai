package llm

import (
	"context"
	"errors"
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

type openaiOptions struct {
	baseURL          string
	disableCache     bool
	reasoningEffort  string
	extraHeaders     map[string]string
	frequencyPenalty *float64
	presencePenalty  *float64
	seed             *int64
}

type OpenAIOption func(*openaiOptions)

type openaiClient struct {
	providerOptions llmClientOptions
	options         openaiOptions
	client          openai.Client
}

type OpenAIClient LLMClient

func newOpenAIClient(opts llmClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{
		reasoningEffort: "medium",
	}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	openaiClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(opts.apiKey))
	}
	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithBaseURL(openaiOpts.baseURL))
	}

	if openaiOpts.extraHeaders != nil {
		for key, value := range openaiOpts.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions: opts,
		options:         openaiOpts,
		client:          client,
	}
}

func (o *openaiClient) convertMessages(messages []message.Message) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content().String()))
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(model.ProviderOpenAI)}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}

				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
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
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls()))
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

			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

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

func (o *openaiClient) convertTools(tools []tool.BaseTool) []openai.ChatCompletionToolParam {
	openaiTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		openaiTools[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
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

func (o *openaiClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.providerOptions.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}

	if o.providerOptions.temperature != nil {
		params.Temperature = openai.Float(*o.providerOptions.temperature)
	}

	if o.providerOptions.topP != nil {
		params.TopP = openai.Float(*o.providerOptions.topP)
	}

	if len(o.providerOptions.stopSequences) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfString: openai.String(o.providerOptions.stopSequences[0]),
		}
	}

	if o.options.frequencyPenalty != nil {
		params.FrequencyPenalty = openai.Float(*o.options.frequencyPenalty)
	}

	if o.options.presencePenalty != nil {
		params.PresencePenalty = openai.Float(*o.options.presencePenalty)
	}

	if o.options.seed != nil {
		params.Seed = openai.Int(*o.options.seed)
	}

	if o.providerOptions.model.CanReason {
		params.MaxCompletionTokens = openai.Int(o.providerOptions.maxTokens)
		switch o.options.reasoningEffort {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		}
	} else {
		params.MaxTokens = openai.Int(o.providerOptions.maxTokens)
	}

	return params
}

func (o *openaiClient) send(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (response *LLMResponse, err error) {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))

	if o.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.providerOptions.timeout)
		defer cancel()
	}

	return ExecuteWithRetry(ctx, OpenAIRetryConfig(), func() (*LLMResponse, error) {
		openaiResponse, err := o.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}

		content := ""
		if openaiResponse.Choices[0].Message.Content != "" {
			content = openaiResponse.Choices[0].Message.Content
		}

		toolCalls := o.toolCalls(*openaiResponse)
		finishReason := o.finishReason(string(openaiResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &LLMResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        o.usage(*openaiResponse),
			FinishReason: finishReason,
		}, nil
	})
}

func (o *openaiClient) stream(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	if o.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.providerOptions.timeout)
		defer cancel()
	}

	eventChan := make(chan LLMEvent)

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
					if choice.Delta.Content != "" {
						eventChan <- LLMEvent{
							Type:    types.EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}
			}

			err := openaiStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				finishReason := o.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- LLMEvent{
					Type: types.EventComplete,
					Response: &LLMResponse{
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

func (o *openaiClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
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

// WithReasoningEffort sets the computational effort level for reasoning models (low, medium, high)
func WithReasoningEffort(effort string) OpenAIOption {
	return func(options *openaiOptions) {
		defaultReasoningEffort := "medium"
		switch effort {
		case "low", "medium", "high":
			defaultReasoningEffort = effort
		default:
		}
		options.reasoningEffort = defaultReasoningEffort
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

func (o *openaiClient) supportsStructuredOutput() bool {
	return true
}

func (o *openaiClient) sendWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) (response *LLMResponse, err error) {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))

	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"required":             outputSchema.Required,
		"additionalProperties": false,
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

	if o.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.providerOptions.timeout)
		defer cancel()
	}

	return ExecuteWithRetry(ctx, OpenAIRetryConfig(), func() (*LLMResponse, error) {
		openaiResponse, err := o.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}

		content := ""
		if openaiResponse.Choices[0].Message.Content != "" {
			content = openaiResponse.Choices[0].Message.Content
		}

		toolCalls := o.toolCalls(*openaiResponse)
		finishReason := o.finishReason(string(openaiResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &LLMResponse{
			Content:                    content,
			ToolCalls:                  toolCalls,
			Usage:                      o.usage(*openaiResponse),
			FinishReason:               finishReason,
			StructuredOutput:           &content,
			UsedNativeStructuredOutput: true,
		}, nil
	})
}

func (o *openaiClient) streamWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) <-chan LLMEvent {
	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))

	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"required":             outputSchema.Required,
		"additionalProperties": false,
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

	if o.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.providerOptions.timeout)
		defer cancel()
	}

	eventChan := make(chan LLMEvent)

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
					if choice.Delta.Content != "" {
						eventChan <- LLMEvent{
							Type:    types.EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}
			}

			err := openaiStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				finishReason := o.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- LLMEvent{
					Type: types.EventComplete,
					Response: &LLMResponse{
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
