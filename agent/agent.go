package agent

import (
	"context"
	"encoding/json"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type Agent struct {
	llm           llm.LLM
	memoryLLM     llm.LLM
	tools         []tool.BaseTool
	systemPrompt  string
	maxIterations int
	autoExecute   bool
	memory        Memory
	userIDKey     string
	autoExtract   bool
	autoDedup     bool
}

func (a *Agent) getMemoryLLM() llm.LLM {
	if a.memoryLLM != nil {
		return a.memoryLLM
	}
	return a.llm
}

func New(llmClient llm.LLM, opts ...AgentOption) *Agent {
	a := &Agent{
		llm:           llmClient,
		tools:         make([]tool.BaseTool, 0),
		maxIterations: 10,
		autoExecute:   true,
		userIDKey:     "user_id",
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func (a *Agent) getTools() []tool.BaseTool {
	allTools := make([]tool.BaseTool, len(a.tools))
	copy(allTools, a.tools)

	if a.memory != nil {
		memoryTools := createMemoryTools(a.memory, a.userIDKey)
		allTools = append(allTools, memoryTools...)
	}

	return allTools
}

func (a *Agent) buildMessages(ctx context.Context, session Session, userMessage string) ([]message.Message, error) {
	var messages []message.Message

	if a.systemPrompt != "" {
		messages = append(messages, message.NewSystemMessage(a.systemPrompt))
	}

	sessionMessages, err := session.GetMessages(ctx, nil)
	if err != nil {
		return nil, err
	}
	messages = append(messages, sessionMessages...)

	userMsg := message.NewUserMessage(userMessage)
	messages = append(messages, userMsg)

	if err := session.AddMessages(ctx, []message.Message{userMsg}); err != nil {
		return nil, err
	}

	return messages, nil
}

func (a *Agent) executeTools(ctx context.Context, toolCalls []message.ToolCall) []ToolExecutionResult {
	registry := tool.NewRegistry()
	for _, t := range a.getTools() {
		registry.Register(t)
	}

	var results []ToolExecutionResult
	for _, tc := range toolCalls {
		resp, err := registry.Execute(ctx, tool.ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: tc.Input,
		})

		result := ToolExecutionResult{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Input:      tc.Input,
			IsError:    resp.IsError || err != nil,
		}

		if err != nil {
			result.Output = err.Error()
		} else {
			result.Output = resp.Content
		}

		results = append(results, result)
	}

	return results
}

func (a *Agent) Chat(ctx context.Context, session Session, userMessage string) (*ChatResponse, error) {
	messages, err := a.buildMessages(ctx, session, userMessage)
	if err != nil {
		return nil, err
	}

	allTools := a.getTools()
	iteration := 0

	for {
		resp, err := a.llm.SendMessages(ctx, messages, allTools)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 || !a.autoExecute || iteration >= a.maxIterations {
			assistantMsg := message.NewAssistantMessage()
			assistantMsg.AppendContent(resp.Content)
			if err := session.AddMessages(ctx, []message.Message{assistantMsg}); err != nil {
				return nil, err
			}

			if a.autoExtract {
				_ = a.extractAndStoreMemories(ctx, session)
			}

			return &ChatResponse{
				Content:      resp.Content,
				ToolCalls:    resp.ToolCalls,
				Usage:        resp.Usage,
				FinishReason: resp.FinishReason,
			}, nil
		}

		assistantMsg := message.NewAssistantMessage()
		assistantMsg.SetToolCalls(resp.ToolCalls)
		messages = append(messages, assistantMsg)

		toolResults := a.executeTools(ctx, resp.ToolCalls)

		toolMsg := message.Message{Role: message.Tool}
		for _, result := range toolResults {
			toolMsg.AddToolResult(message.ToolResult{
				ToolCallID: result.ToolCallID,
				Name:       result.ToolName,
				Content:    result.Output,
				IsError:    result.IsError,
			})
		}
		messages = append(messages, toolMsg)

		if err := session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg}); err != nil {
			return nil, err
		}

		iteration++
	}
}

func (a *Agent) ChatStream(ctx context.Context, session Session, userMessage string) <-chan ChatEvent {
	eventChan := make(chan ChatEvent)

	go func() {
		defer close(eventChan)

		messages, err := a.buildMessages(ctx, session, userMessage)
		if err != nil {
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return
		}

		allTools := a.getTools()
		iteration := 0

		for {
			var fullContent string
			var toolCalls []message.ToolCall
			var finalResponse *llm.LLMResponse

			for event := range a.llm.StreamResponse(ctx, messages, allTools) {
				switch event.Type {
				case types.EventContentDelta:
					fullContent += event.Content
					eventChan <- ChatEvent{Type: types.EventContentDelta, Content: event.Content}
				case types.EventThinkingDelta:
					eventChan <- ChatEvent{Type: types.EventThinkingDelta, Thinking: event.Thinking}
				case types.EventToolUseStart, types.EventToolUseDelta, types.EventToolUseStop:
					if event.ToolCall != nil {
						eventChan <- ChatEvent{Type: event.Type, ToolCall: event.ToolCall}
					}
				case types.EventComplete:
					if event.Response != nil {
						finalResponse = event.Response
						toolCalls = event.Response.ToolCalls
					}
				case types.EventError:
					eventChan <- ChatEvent{Type: types.EventError, Error: event.Error}
					return
				}
			}

			if len(toolCalls) == 0 || !a.autoExecute || iteration >= a.maxIterations {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.AppendContent(fullContent)
				_ = session.AddMessages(ctx, []message.Message{assistantMsg})

				if a.autoExtract {
					_ = a.extractAndStoreMemories(ctx, session)
				}

				var usage llm.TokenUsage
				var finishReason message.FinishReason
				if finalResponse != nil {
					usage = finalResponse.Usage
					finishReason = finalResponse.FinishReason
				}

				eventChan <- ChatEvent{
					Type: types.EventComplete,
					Response: &ChatResponse{
						Content:      fullContent,
						ToolCalls:    toolCalls,
						Usage:        usage,
						FinishReason: finishReason,
					},
				}
				return
			}

			assistantMsg := message.NewAssistantMessage()
			assistantMsg.SetToolCalls(toolCalls)
			messages = append(messages, assistantMsg)

			toolResults := a.executeTools(ctx, toolCalls)

			for _, result := range toolResults {
				eventChan <- ChatEvent{
					Type:       types.EventToolUseStop,
					ToolResult: &result,
				}
			}

			toolMsg := message.Message{Role: message.Tool}
			for _, result := range toolResults {
				toolMsg.AddToolResult(message.ToolResult{
					ToolCallID: result.ToolCallID,
					Name:       result.ToolName,
					Content:    result.Output,
					IsError:    result.IsError,
				})
			}
			messages = append(messages, toolMsg)

			_ = session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg})

			iteration++
		}
	}()

	return eventChan
}

func (a *Agent) ChatWithMemoryContext(ctx context.Context, session Session, userMessage string) (*ChatResponse, error) {
	if a.memory != nil {
		userID, ok := ctx.Value(a.userIDKey).(string)
		if ok && userID != "" {
			memories, err := a.memory.Search(ctx, userID, userMessage, 5)
			if err == nil && len(memories) > 0 {
				var memoryContext string
				for _, m := range memories {
					memoryContext += "- " + m.Content + "\n"
				}

				originalPrompt := a.systemPrompt
				a.systemPrompt = a.systemPrompt + "\n\nRelevant memories about this user:\n" + memoryContext
				defer func() { a.systemPrompt = originalPrompt }()
			}
		}
	}

	return a.Chat(ctx, session, userMessage)
}

func ParseToolInput[T any](input string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(input), &result)
	return result, err
}

