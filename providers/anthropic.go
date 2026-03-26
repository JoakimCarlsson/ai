package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type anthropicOptions struct {
	useBedrock   bool
	disableCache bool
	shouldThink  func(userMessage string) bool
}

// AnthropicOption configures optional settings for Anthropic clients.
type AnthropicOption func(*anthropicOptions)

type anthropicClient struct {
	llmOptions llmClientOptions
	options    anthropicOptions
	client     anthropic.Client
}

// AnthropicClient is the Anthropic Client implementation type.
type AnthropicClient Client

func newAnthropicClient(opts llmClientOptions) AnthropicClient {
	anthropicOpts := anthropicOptions{}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}

	anthropicClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		anthropicClientOptions = append(
			anthropicClientOptions,
			option.WithAPIKey(opts.apiKey),
		)
	}
	if anthropicOpts.useBedrock {
		anthropicClientOptions = append(
			anthropicClientOptions,
			bedrock.WithLoadDefaultConfig(context.Background()),
		)
	}

	client := anthropic.NewClient(anthropicClientOptions...)
	return &anthropicClient{
		llmOptions: opts,
		options:    anthropicOpts,
		client:     client,
	}
}

func (a *anthropicClient) convertMessages(
	messages []message.Message,
) (anthropicMessages []anthropic.MessageParam, systemMessages []string) {
	for i, msg := range messages {
		cache := false
		if i == len(messages)-1 && !a.options.disableCache {
			cache = true
		}
		switch msg.Role {
		case message.System:
			systemMessages = append(systemMessages, msg.Content().String())
		case message.User:
			content := anthropic.NewTextBlock(msg.Content().String())
			if cache {
				content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			var contentBlocks []anthropic.ContentBlockParamUnion
			contentBlocks = append(contentBlocks, content)

			for _, binaryContent := range msg.BinaryContent() {
				base64Image := binaryContent.String(model.ProviderAnthropic)
				imageBlock := anthropic.NewImageBlockBase64(
					binaryContent.MIMEType,
					base64Image,
				)
				contentBlocks = append(contentBlocks, imageBlock)
			}

			for _, imageURLContent := range msg.ImageURLContent() {
				imageBlock := anthropic.NewImageBlock(
					anthropic.URLImageSourceParam{
						Type: "url",
						URL:  imageURLContent.URL,
					},
				)
				contentBlocks = append(contentBlocks, imageBlock)
			}

			anthropicMessages = append(
				anthropicMessages,
				anthropic.NewUserMessage(contentBlocks...),
			)

		case message.Assistant:
			blocks := []anthropic.ContentBlockParamUnion{}
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(msg.Content().String())
				blocks = append(blocks, content)
			}

			for _, toolCall := range msg.ToolCalls() {
				var inputMap map[string]any
				err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
				if err != nil {
					continue
				}
				blocks = append(
					blocks,
					anthropic.NewToolUseBlock(
						toolCall.ID,
						inputMap,
						toolCall.Name,
					),
				)
			}

			if len(blocks) == 0 {
				slog.Warn(
					"There is a message without content, investigate, this should not happen",
				)
				continue
			}
			anthropicMessages = append(
				anthropicMessages,
				anthropic.NewAssistantMessage(blocks...),
			)

		case message.Tool:
			results := make(
				[]anthropic.ContentBlockParamUnion,
				len(msg.ToolResults()),
			)
			for i, toolResult := range msg.ToolResults() {
				results[i] = anthropic.NewToolResultBlock(
					toolResult.ToolCallID,
					toolResult.Content,
					toolResult.IsError,
				)
			}
			anthropicMessages = append(
				anthropicMessages,
				anthropic.NewUserMessage(results...),
			)
		}
	}
	return
}

func (a *anthropicClient) convertTools(
	tools []tool.BaseTool,
) []anthropic.ToolUnionParam {
	anthropicTools := make([]anthropic.ToolUnionParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		toolParam := anthropic.ToolParam{
			Name:        info.Name,
			Description: anthropic.String(info.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: info.Parameters,
			},
		}

		if i == len(tools)-1 && !a.options.disableCache {
			toolParam.CacheControl = anthropic.CacheControlEphemeralParam{
				Type: "ephemeral",
			}
		}

		anthropicTools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return anthropicTools
}

func (a *anthropicClient) finishReason(reason string) message.FinishReason {
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

func (a *anthropicClient) preparedMessages(
	messages []anthropic.MessageParam,
	tools []anthropic.ToolUnionParam,
	systemMessages []string,
) anthropic.MessageNewParams {
	var thinkingParam anthropic.ThinkingConfigParamUnion
	lastMessage := messages[len(messages)-1]
	isUser := lastMessage.Role == anthropic.MessageParamRoleUser
	messageContent := ""
	temperature := anthropic.Float(0)
	paramBuilder := newParameterBuilder(a.llmOptions)
	paramBuilder.applyFloat64Temperature(
		func(t *float64) { temperature = anthropic.Float(*t) },
	)
	if isUser {
		for _, m := range lastMessage.Content {
			if m.OfText != nil && m.OfText.Text != "" {
				messageContent = m.OfText.Text
			}
		}
		if messageContent != "" && a.options.shouldThink != nil &&
			a.options.shouldThink(messageContent) {
			thinkingParam = anthropic.ThinkingConfigParamUnion{
				OfEnabled: &anthropic.ThinkingConfigEnabledParam{
					BudgetTokens: int64(float64(a.llmOptions.maxTokens) * 0.8),
					Type:         "enabled",
				},
			}
			if a.llmOptions.temperature == nil {
				temperature = anthropic.Float(1)
			}
		}
	}

	if a.llmOptions.maxTokens == 0 {
		a.llmOptions.maxTokens = a.llmOptions.model.DefaultMaxTokens
	} else {
		a.llmOptions.maxTokens = int64(a.llmOptions.maxTokens)
	}

	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(a.llmOptions.model.APIModel),
		MaxTokens:   a.llmOptions.maxTokens,
		Temperature: temperature,
		Messages:    messages,
		Tools:       tools,
		Thinking:    thinkingParam,
	}

	paramBuilder.applyFloat64TopP(
		func(p *float64) { params.TopP = anthropic.Float(*p) },
	)
	paramBuilder.applyInt64TopK(
		func(k *int64) { params.TopK = anthropic.Int(*k) },
	)

	if len(a.llmOptions.stopSequences) > 0 {
		params.StopSequences = a.llmOptions.stopSequences
	}

	if len(systemMessages) > 0 {
		systemBlocks := make([]anthropic.TextBlockParam, len(systemMessages))
		for i, sysMsg := range systemMessages {
			block := anthropic.TextBlockParam{
				Text: sysMsg,
			}
			if i == len(systemMessages)-1 && !a.options.disableCache {
				block.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			systemBlocks[i] = block
		}
		params.System = systemBlocks
	}

	return params
}

func (a *anthropicClient) send(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (resposne *Response, err error) {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(
		anthropicMessages,
		a.convertTools(tools),
		systemMessages,
	)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	return ExecuteWithRetry(
		ctx,
		AnthropicRetryConfig(),
		func() (*Response, error) {
			anthropicResponse, err := a.client.Messages.New(
				ctx,
				preparedMessages,
			)
			if err != nil {
				return nil, err
			}

			content := ""
			for _, block := range anthropicResponse.Content {
				if text, ok := block.AsAny().(anthropic.TextBlock); ok {
					content += text.Text
				}
			}

			return &Response{
				Content:   content,
				ToolCalls: a.toolCalls(*anthropicResponse),
				Usage:     a.usage(*anthropicResponse),
				FinishReason: a.finishReason(
					string(anthropicResponse.StopReason),
				),
			}, nil
		},
	)
}

func (a *anthropicClient) stream(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan Event {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(
		anthropicMessages,
		a.convertTools(tools),
		systemMessages,
	)
	eventChan := make(chan Event)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, AnthropicRetryConfig(), func() error {
			anthropicStream := a.client.Messages.NewStreaming(
				ctx,
				preparedMessages,
			)
			accumulatedMessage := anthropic.Message{}

			currentToolCallID := ""
			for anthropicStream.Next() {
				event := anthropicStream.Current()
				err := accumulatedMessage.Accumulate(event)
				if err != nil {
					slog.Warn("Error accumulating message", "error", err)
					continue
				}

				switch event := event.AsAny().(type) {
				case anthropic.ContentBlockStartEvent:
					switch event.ContentBlock.Type {
					case "text":
						eventChan <- Event{Type: types.EventContentStart}
					case "tool_use":
						currentToolCallID = event.ContentBlock.ID
						eventChan <- Event{
							Type: types.EventToolUseStart,
							ToolCall: &message.ToolCall{
								ID:       event.ContentBlock.ID,
								Name:     event.ContentBlock.Name,
								Finished: false,
							},
						}
					}

				case anthropic.ContentBlockDeltaEvent:
					switch event.Delta.Type {
					case "thinking_delta":
						if event.Delta.Thinking != "" {
							eventChan <- Event{
								Type:     types.EventThinkingDelta,
								Thinking: event.Delta.Thinking,
							}
						}
					case "text_delta":
						if event.Delta.Text != "" {
							eventChan <- Event{
								Type:    types.EventContentDelta,
								Content: event.Delta.Text,
							}
						}
					case "input_json_delta":
						if currentToolCallID != "" {
							eventChan <- Event{
								Type: types.EventToolUseDelta,
								ToolCall: &message.ToolCall{
									ID:       currentToolCallID,
									Finished: false,
									Input:    event.Delta.JSON.PartialJSON.Raw(),
								},
							}
						}
					}
				case anthropic.ContentBlockStopEvent:
					if currentToolCallID != "" {
						eventChan <- Event{
							Type: types.EventToolUseStop,
							ToolCall: &message.ToolCall{
								ID: currentToolCallID,
							},
						}
						currentToolCallID = ""
					} else {
						eventChan <- Event{Type: types.EventContentStop}
					}

				case anthropic.MessageStopEvent:
					content := ""
					for _, block := range accumulatedMessage.Content {
						if text, ok := block.AsAny().(anthropic.TextBlock); ok {
							content += text.Text
						}
					}

					eventChan <- Event{
						Type: types.EventComplete,
						Response: &Response{
							Content:      content,
							ToolCalls:    a.toolCalls(accumulatedMessage),
							Usage:        a.usage(accumulatedMessage),
							FinishReason: a.finishReason(string(accumulatedMessage.StopReason)),
						},
					}
				}
			}

			err := anthropicStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}, eventChan)
	}()
	return eventChan
}

func (a *anthropicClient) toolCalls(msg anthropic.Message) []message.ToolCall {
	var toolCalls []message.ToolCall

	for _, block := range msg.Content {
		if variant, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			toolCall := message.ToolCall{
				ID:       variant.ID,
				Name:     variant.Name,
				Input:    string(variant.Input),
				Type:     string(variant.Type),
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (a *anthropicClient) usage(msg anthropic.Message) TokenUsage {
	return TokenUsage{
		InputTokens:         msg.Usage.InputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
	}
}

// WithAnthropicBedrock configures whether to use AWS Bedrock for Anthropic models
func WithAnthropicBedrock(useBedrock bool) AnthropicOption {
	return func(options *anthropicOptions) {
		options.useBedrock = useBedrock
	}
}

// WithAnthropicDisableCache disables response caching for Anthropic requests
func WithAnthropicDisableCache() AnthropicOption {
	return func(options *anthropicOptions) {
		options.disableCache = true
	}
}

// DefaultShouldThinkFn checks if the user message contains "think" to enable reasoning mode
func DefaultShouldThinkFn(s string) bool {
	return strings.Contains(strings.ToLower(s), "think")
}

// WithAnthropicShouldThinkFn sets a custom function to determine when to enable reasoning mode
func WithAnthropicShouldThinkFn(fn func(string) bool) AnthropicOption {
	return func(options *anthropicOptions) {
		options.shouldThink = fn
	}
}

func (a *anthropicClient) supportsStructuredOutput() bool {
	return a.llmOptions.model.SupportsStructuredOut
}

func (a *anthropicClient) buildOutputConfig(
	outputSchema *schema.StructuredOutputInfo,
) anthropic.OutputConfigParam {
	schemaMap := map[string]any{
		"type":                 "object",
		"properties":           outputSchema.Parameters,
		"additionalProperties": false,
	}
	if len(outputSchema.Required) > 0 {
		schemaMap["required"] = outputSchema.Required
	}
	return anthropic.OutputConfigParam{
		Format: anthropic.JSONOutputFormatParam{
			Schema: schemaMap,
		},
	}
}

func (a *anthropicClient) sendWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*Response, error) {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(
		anthropicMessages,
		a.convertTools(tools),
		systemMessages,
	)
	preparedMessages.OutputConfig = a.buildOutputConfig(outputSchema)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	return ExecuteWithRetry(
		ctx,
		AnthropicRetryConfig(),
		func() (*Response, error) {
			anthropicResponse, err := a.client.Messages.New(
				ctx,
				preparedMessages,
			)
			if err != nil {
				return nil, err
			}

			content := ""
			for _, block := range anthropicResponse.Content {
				if text, ok := block.AsAny().(anthropic.TextBlock); ok {
					content += text.Text
				}
			}

			return &Response{
				Content:   content,
				ToolCalls: a.toolCalls(*anthropicResponse),
				Usage:     a.usage(*anthropicResponse),
				FinishReason: a.finishReason(
					string(anthropicResponse.StopReason),
				),
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
			}, nil
		},
	)
}

func (a *anthropicClient) streamWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan Event {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(
		anthropicMessages,
		a.convertTools(tools),
		systemMessages,
	)
	preparedMessages.OutputConfig = a.buildOutputConfig(outputSchema)

	eventChan := make(chan Event)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, AnthropicRetryConfig(), func() error {
			anthropicStream := a.client.Messages.NewStreaming(
				ctx,
				preparedMessages,
			)
			accumulatedMessage := anthropic.Message{}

			currentToolCallID := ""
			for anthropicStream.Next() {
				event := anthropicStream.Current()
				err := accumulatedMessage.Accumulate(event)
				if err != nil {
					slog.Warn("Error accumulating message", "error", err)
					continue
				}

				switch event := event.AsAny().(type) {
				case anthropic.ContentBlockStartEvent:
					switch event.ContentBlock.Type {
					case "text":
						eventChan <- Event{Type: types.EventContentStart}
					case "tool_use":
						currentToolCallID = event.ContentBlock.ID
						eventChan <- Event{
							Type: types.EventToolUseStart,
							ToolCall: &message.ToolCall{
								ID:       event.ContentBlock.ID,
								Name:     event.ContentBlock.Name,
								Finished: false,
							},
						}
					}

				case anthropic.ContentBlockDeltaEvent:
					switch event.Delta.Type {
					case "thinking_delta":
						if event.Delta.Thinking != "" {
							eventChan <- Event{
								Type:     types.EventThinkingDelta,
								Thinking: event.Delta.Thinking,
							}
						}
					case "text_delta":
						if event.Delta.Text != "" {
							eventChan <- Event{
								Type:    types.EventContentDelta,
								Content: event.Delta.Text,
							}
						}
					case "input_json_delta":
						if currentToolCallID != "" {
							eventChan <- Event{
								Type: types.EventToolUseDelta,
								ToolCall: &message.ToolCall{
									ID:       currentToolCallID,
									Finished: false,
									Input:    event.Delta.JSON.PartialJSON.Raw(),
								},
							}
						}
					}
				case anthropic.ContentBlockStopEvent:
					if currentToolCallID != "" {
						eventChan <- Event{
							Type: types.EventToolUseStop,
							ToolCall: &message.ToolCall{
								ID: currentToolCallID,
							},
						}
						currentToolCallID = ""
					} else {
						eventChan <- Event{Type: types.EventContentStop}
					}

				case anthropic.MessageStopEvent:
					content := ""
					for _, block := range accumulatedMessage.Content {
						if text, ok := block.AsAny().(anthropic.TextBlock); ok {
							content += text.Text
						}
					}

					eventChan <- Event{
						Type: types.EventComplete,
						Response: &Response{
							Content:                    content,
							ToolCalls:                  a.toolCalls(accumulatedMessage),
							Usage:                      a.usage(accumulatedMessage),
							FinishReason:               a.finishReason(string(accumulatedMessage.StopReason)),
							StructuredOutput:           &content,
							UsedNativeStructuredOutput: true,
						},
					}
				}
			}

			err := anthropicStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}, eventChan)
	}()
	return eventChan
}
