package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// Agent is an AI assistant that can chat with users, use tools, and maintain memory.
// Create one using New() with functional options.
type Agent struct {
	llm              llm.LLM
	memoryLLM        llm.LLM
	tools            []tool.BaseTool
	systemPrompt     string
	maxIterations    int
	autoExecute      bool
	memory           memory.Store
	memoryID         string
	autoExtract      bool
	autoDedup        bool
	session          session.Session
	contextStrategy  tokens.Strategy
	reserveTokens    int64
	maxContextTokens int64
}

func (a *Agent) getMemoryLLM() llm.LLM {
	if a.memoryLLM != nil {
		return a.memoryLLM
	}
	return a.llm
}

// New creates a new Agent with the given LLM client and options.
// The agent can be configured with tools, memory, session persistence, and more.
//
// Example:
//
//	agent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a helpful assistant."),
//	    agent.WithTools(&myTool{}),
//	    agent.WithSession("conv-1", session.FileStore("./sessions")),
//	    agent.WithMemory("user-123", myMemoryStore, memory.AutoExtract()),
//	)
func New(llmClient llm.LLM, opts ...AgentOption) *Agent {
	a := &Agent{
		llm:           llmClient,
		tools:         make([]tool.BaseTool, 0),
		maxIterations: 10,
		autoExecute:   true,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func (a *Agent) getTools() []tool.BaseTool {
	allTools := make([]tool.BaseTool, len(a.tools))
	copy(allTools, a.tools)

	if a.memory != nil && !a.autoExtract && a.memoryID != "" {
		memoryTools := createMemoryTools(a.memory, a.memoryID)
		allTools = append(allTools, memoryTools...)
	}

	return allTools
}

// BuildContextMessages returns the messages that would be sent to the LLM after applying
// the context strategy. This is useful for debugging and testing context management.
// WARNING: This method modifies the session by adding the user message.
func (a *Agent) BuildContextMessages(ctx context.Context, userMessage string) ([]message.Message, error) {
	return a.buildMessages(ctx, userMessage)
}

// PeekContextMessages returns what messages would be sent to the LLM without modifying state.
func (a *Agent) PeekContextMessages(ctx context.Context, userMessage string) ([]message.Message, error) {
	var messages []message.Message

	if a.systemPrompt != "" {
		messages = append(messages, message.NewSystemMessage(a.systemPrompt))
	}

	if a.session != nil {
		sessionMessages, err := a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
		messages = append(messages, sessionMessages...)
	}

	messages = append(messages, message.NewUserMessage(userMessage))

	if a.contextStrategy != nil {
		counter, err := tokens.NewCounter()
		if err != nil {
			return nil, err
		}

		maxTokens := a.maxContextTokens
		if maxTokens == 0 {
			reserveTokens := a.reserveTokens
			if reserveTokens == 0 {
				reserveTokens = 4096
			}
			maxTokens = a.llm.Model().ContextWindow - reserveTokens
		}

		result, err := a.contextStrategy.Fit(ctx, tokens.StrategyInput{
			Messages:     messages,
			SystemPrompt: a.systemPrompt,
			Tools:        a.getTools(),
			Counter:      counter,
			MaxTokens:    maxTokens,
		})
		if err != nil {
			return nil, err
		}

		messages = result.Messages
	}

	return messages, nil
}

func (a *Agent) buildMessages(ctx context.Context, userMessage string) ([]message.Message, error) {
	var messages []message.Message

	systemPrompt := a.systemPrompt
	if a.memory != nil && a.memoryID != "" {
		memories, err := a.memory.Search(ctx, a.memoryID, userMessage, 5)
		if err == nil && len(memories) > 0 {
			var memoryContext string
			for _, m := range memories {
				memoryContext += "- " + m.Content + "\n"
			}
			systemPrompt = systemPrompt + "\n\nRelevant memories about this user:\n" + memoryContext
		}
	}

	if systemPrompt != "" {
		messages = append(messages, message.NewSystemMessage(systemPrompt))
	}

	if a.session != nil {
		sessionMessages, err := a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
		messages = append(messages, sessionMessages...)
	}

	userMsg := message.NewUserMessage(userMessage)
	messages = append(messages, userMsg)

	if a.session != nil {
		if err := a.session.AddMessages(ctx, []message.Message{userMsg}); err != nil {
			return nil, err
		}
	}

	if a.contextStrategy != nil {
		counter, err := tokens.NewCounter()
		if err != nil {
			return nil, fmt.Errorf("failed to create token counter: %w", err)
		}

		maxTokens := a.maxContextTokens
		if maxTokens == 0 {
			reserveTokens := a.reserveTokens
			if reserveTokens == 0 {
				reserveTokens = 4096
			}
			maxTokens = a.llm.Model().ContextWindow - reserveTokens
		}

		result, err := a.contextStrategy.Fit(ctx, tokens.StrategyInput{
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        a.getTools(),
			Counter:      counter,
			MaxTokens:    maxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("context strategy failed: %w", err)
		}

		messages = result.Messages

		if result.SessionUpdate != nil && a.session != nil && len(result.SessionUpdate.AddMessages) > 0 {
			if err := a.session.AddMessages(ctx, result.SessionUpdate.AddMessages); err != nil {
				return nil, fmt.Errorf("failed to save session update: %w", err)
			}
		}
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

func (a *Agent) extractAndStoreMemories(ctx context.Context) error {
	if a.memory == nil || !a.autoExtract || a.memoryID == "" || a.session == nil {
		return nil
	}

	messages, err := a.session.GetMessages(ctx, nil)
	if err != nil {
		return err
	}

	facts, err := memory.ExtractFacts(ctx, a.getMemoryLLM(), messages)
	if err != nil {
		return err
	}

	for _, fact := range facts {
		metadata := map[string]any{
			"source":     "auto_extract",
			"created_at": time.Now().Format(time.RFC3339),
		}
		var storeErr error
		if a.autoDedup {
			storeErr = a.storeWithDedup(ctx, fact, metadata)
		} else {
			storeErr = a.memory.Store(ctx, a.memoryID, fact, metadata)
		}
		if storeErr != nil {
			continue
		}
	}

	return nil
}

func (a *Agent) storeWithDedup(ctx context.Context, fact string, metadata map[string]any) error {
	if !a.autoDedup || a.memory == nil || a.memoryID == "" {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	existing, err := a.memory.Search(ctx, a.memoryID, fact, 5)
	if err != nil {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	result, err := memory.Deduplicate(ctx, a.getMemoryLLM(), fact, existing)
	if err != nil {
		return a.memory.Store(ctx, a.memoryID, fact, metadata)
	}

	for _, decision := range result.Decisions {
		switch decision.Event {
		case memory.DedupEventAdd:
			if err := a.memory.Store(ctx, a.memoryID, decision.Text, metadata); err != nil {
				return err
			}
		case memory.DedupEventUpdate:
			if err := a.memory.Update(ctx, decision.MemoryID, decision.Text, metadata); err != nil {
				return err
			}
		case memory.DedupEventDelete:
			if err := a.memory.Delete(ctx, decision.MemoryID); err != nil {
				return err
			}
		case memory.DedupEventNone:
		}
	}

	return nil
}

// Chat sends a message to the agent and returns the response.
// If the agent has tools configured, it will automatically execute them.
// If memory is configured, relevant memories are injected into the context.
// If a session is configured, the conversation history is persisted.
func (a *Agent) Chat(ctx context.Context, userMessage string) (*ChatResponse, error) {
	messages, err := a.buildMessages(ctx, userMessage)
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
			if a.session != nil && resp.Content != "" {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.AppendContent(resp.Content)
				if err := a.session.AddMessages(ctx, []message.Message{assistantMsg}); err != nil {
					return nil, err
				}
			}

			if a.autoExtract && a.session != nil {
				go a.extractAndStoreMemories(context.Background())
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

		toolMsg := message.Message{Role: message.Tool, CreatedAt: time.Now().UnixNano()}
		for _, result := range toolResults {
			toolMsg.AddToolResult(message.ToolResult{
				ToolCallID: result.ToolCallID,
				Name:       result.ToolName,
				Content:    result.Output,
				IsError:    result.IsError,
			})
		}
		messages = append(messages, toolMsg)

		if a.session != nil {
			if err := a.session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg}); err != nil {
				return nil, err
			}
		}

		iteration++
	}
}

// ChatStream sends a message to the agent and returns a channel of streaming events.
// Events include content deltas, tool calls, and the final response.
// The channel is closed when the response is complete or an error occurs.
func (a *Agent) ChatStream(ctx context.Context, userMessage string) <-chan ChatEvent {
	eventChan := make(chan ChatEvent)

	go func() {
		defer close(eventChan)

		messages, err := a.buildMessages(ctx, userMessage)
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
				if a.session != nil && fullContent != "" {
					assistantMsg := message.NewAssistantMessage()
					assistantMsg.AppendContent(fullContent)
					_ = a.session.AddMessages(ctx, []message.Message{assistantMsg})
				}

				if a.autoExtract && a.session != nil {
					go a.extractAndStoreMemories(context.Background())
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

		for _, tc := range toolCalls {
			eventChan <- ChatEvent{
				Type: types.EventToolUseStart,
				ToolCall: &message.ToolCall{
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				},
			}
		}

		toolResults := a.executeTools(ctx, toolCalls)

			for _, result := range toolResults {
				eventChan <- ChatEvent{
					Type:       types.EventToolUseStop,
					ToolResult: &result,
				}
			}

			toolMsg := message.Message{Role: message.Tool, CreatedAt: time.Now().UnixNano()}
			for _, result := range toolResults {
				toolMsg.AddToolResult(message.ToolResult{
					ToolCallID: result.ToolCallID,
					Name:       result.ToolName,
					Content:    result.Output,
					IsError:    result.IsError,
				})
			}
			messages = append(messages, toolMsg)

			if a.session != nil {
				_ = a.session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg})
			}

			iteration++
		}
	}()

	return eventChan
}

// ParseToolInput parses a JSON tool input string into the specified type.
// This is a helper function for implementing tool.BaseTool.Run().
func ParseToolInput[T any](input string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(input), &result)
	return result, err
}
