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

type AnthropicOption func(*anthropicOptions)

type anthropicClient struct {
	llmOptions llmClientOptions
	options    anthropicOptions
	client     anthropic.Client
}

type AnthropicClient LLMClient

func newAnthropicClient(opts llmClientOptions) AnthropicClient {
	anthropicOpts := anthropicOptions{}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}

	anthropicClientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey(opts.apiKey))
	}
	if anthropicOpts.useBedrock {
		anthropicClientOptions = append(anthropicClientOptions, bedrock.WithLoadDefaultConfig(context.Background()))
	}

	client := anthropic.NewClient(anthropicClientOptions...)
	return &anthropicClient{
		llmOptions: opts,
		options:    anthropicOpts,
		client:     client,
	}
}

func (a *anthropicClient) convertMessages(messages []message.Message) (anthropicMessages []anthropic.MessageParam, systemMessages []string) {
	for i, msg := range messages {
		cache := false
		if i > len(messages)-3 {
			cache = true
		}
		switch msg.Role {
		case message.System:
			systemMessages = append(systemMessages, msg.Content().String())
		case message.User:
			content := anthropic.NewTextBlock(msg.Content().String())
			if cache && !a.options.disableCache {
				content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			var contentBlocks []anthropic.ContentBlockParamUnion
			contentBlocks = append(contentBlocks, content)
			for _, binaryContent := range msg.BinaryContent() {
				base64Image := binaryContent.String(model.ProviderAnthropic)
				imageBlock := anthropic.NewImageBlockBase64(binaryContent.MIMEType, base64Image)
				contentBlocks = append(contentBlocks, imageBlock)
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(contentBlocks...))

		case message.Assistant:
			blocks := []anthropic.ContentBlockParamUnion{}
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(msg.Content().String())
				if cache && !a.options.disableCache {
					content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					}
				}
				blocks = append(blocks, content)
			}

			for _, toolCall := range msg.ToolCalls() {
				var inputMap map[string]any
				err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
				if err != nil {
					continue
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(toolCall.ID, inputMap, toolCall.Name))
			}

			if len(blocks) == 0 {
				slog.Warn("There is a message without content, investigate, this should not happen")
				continue
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))

		case message.Tool:
			results := make([]anthropic.ContentBlockParamUnion, len(msg.ToolResults()))
			for i, toolResult := range msg.ToolResults() {
				results[i] = anthropic.NewToolResultBlock(toolResult.ToolCallID, toolResult.Content, toolResult.IsError)
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(results...))
		}
	}
	return
}

func (a *anthropicClient) convertTools(tools []tool.BaseTool) []anthropic.ToolUnionParam {
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

func (a *anthropicClient) preparedMessages(messages []anthropic.MessageParam, tools []anthropic.ToolUnionParam, systemMessages []string) anthropic.MessageNewParams {
	var thinkingParam anthropic.ThinkingConfigParamUnion
	lastMessage := messages[len(messages)-1]
	isUser := lastMessage.Role == anthropic.MessageParamRoleUser
	messageContent := ""
	temperature := anthropic.Float(0)
	if a.llmOptions.temperature != nil {
		temperature = anthropic.Float(*a.llmOptions.temperature)
	}
	if isUser {
		for _, m := range lastMessage.Content {
			if m.OfText != nil && m.OfText.Text != "" {
				messageContent = m.OfText.Text
			}
		}
		if messageContent != "" && a.options.shouldThink != nil && a.options.shouldThink(messageContent) {
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

	if a.llmOptions.topP != nil {
		params.TopP = anthropic.Float(*a.llmOptions.topP)
	}

	if a.llmOptions.topK != nil {
		params.TopK = anthropic.Int(*a.llmOptions.topK)
	}

	if len(a.llmOptions.stopSequences) > 0 {
		params.StopSequences = a.llmOptions.stopSequences
	}

	if len(systemMessages) > 0 {
		systemBlocks := make([]anthropic.TextBlockParam, len(systemMessages))
		for i, sysMsg := range systemMessages {
			systemBlocks[i] = anthropic.TextBlockParam{
				Text: sysMsg,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			}
		}
		params.System = systemBlocks
	}

	return params
}

func (a *anthropicClient) send(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (resposne *LLMResponse, err error) {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(anthropicMessages, a.convertTools(tools), systemMessages)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	return ExecuteWithRetry(ctx, AnthropicRetryConfig(), func() (*LLMResponse, error) {
		anthropicResponse, err := a.client.Messages.New(ctx, preparedMessages)
		if err != nil {
			return nil, err
		}

		content := ""
		for _, block := range anthropicResponse.Content {
			if text, ok := block.AsAny().(anthropic.TextBlock); ok {
				content += text.Text
			}
		}

		return &LLMResponse{
			Content:      content,
			ToolCalls:    a.toolCalls(*anthropicResponse),
			Usage:        a.usage(*anthropicResponse),
			FinishReason: a.finishReason(string(anthropicResponse.StopReason)),
		}, nil
	})
}

func (a *anthropicClient) stream(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent {
	anthropicMessages, systemMessages := a.convertMessages(messages)
	preparedMessages := a.preparedMessages(anthropicMessages, a.convertTools(tools), systemMessages)
	eventChan := make(chan LLMEvent)

	ctx, cancel := withTimeout(ctx, a.llmOptions.timeout)
	defer cancel()

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, AnthropicRetryConfig(), func() error {
			anthropicStream := a.client.Messages.NewStreaming(ctx, preparedMessages)
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
						eventChan <- LLMEvent{Type: types.EventContentStart}
					case "tool_use":
						currentToolCallID = event.ContentBlock.ID
						eventChan <- LLMEvent{
							Type: types.EventToolUseStart,
							ToolCall: &message.ToolCall{
								ID:       event.ContentBlock.ID,
								Name:     event.ContentBlock.Name,
								Finished: false,
							},
						}
					}

				case anthropic.ContentBlockDeltaEvent:
					if event.Delta.Type == "thinking_delta" && event.Delta.Thinking != "" {
						eventChan <- LLMEvent{
							Type:     types.EventThinkingDelta,
							Thinking: event.Delta.Thinking,
						}
					} else if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
						eventChan <- LLMEvent{
							Type:    types.EventContentDelta,
							Content: event.Delta.Text,
						}
					} else if event.Delta.Type == "input_json_delta" {
						if currentToolCallID != "" {
							eventChan <- LLMEvent{
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
						eventChan <- LLMEvent{
							Type: types.EventToolUseStop,
							ToolCall: &message.ToolCall{
								ID: currentToolCallID,
							},
						}
						currentToolCallID = ""
					} else {
						eventChan <- LLMEvent{Type: types.EventContentStop}
					}

				case anthropic.MessageStopEvent:
					content := ""
					for _, block := range accumulatedMessage.Content {
						if text, ok := block.AsAny().(anthropic.TextBlock); ok {
							content += text.Text
						}
					}

					eventChan <- LLMEvent{
						Type: types.EventComplete,
						Response: &LLMResponse{
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
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
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

// SupportsStructuredOutput checks if the provider supports structured output
func (a *anthropicClient) supportsStructuredOutput() bool {
	return false
}

// SendMessagesWithStructuredOutput sends messages with a structured output schema
func (a *anthropicClient) sendWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) (*LLMResponse, error) {
	return nil, errors.New("structured output not supported by Anthropic Claude - use tool-based approach instead")
}

// StreamWithStructuredOutput streams messages with a structured output schema
func (a *anthropicClient) streamWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) <-chan LLMEvent {
	errChan := make(chan LLMEvent, 1)
	errChan <- LLMEvent{
		Type:  types.EventError,
		Error: errors.New("structured output not supported by Anthropic Claude - use tool-based approach instead"),
	}
	close(errChan)
	return errChan
}
