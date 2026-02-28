package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/prompt"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// Agent is an AI assistant that can chat with users, use tools, and maintain memory.
// Create one using New() with functional options.
type Agent struct {
	llm                 llm.LLM
	memoryLLM           llm.LLM
	tools               []tool.BaseTool
	systemPrompt        string
	maxIterations       int
	autoExecute         bool
	memory              memory.Store
	memoryID            string
	autoExtract         bool
	autoDedup           bool
	session             session.Session
	contextStrategy     tokens.Strategy
	reserveTokens       int64
	maxContextTokens    int64
	parallelTools       bool
	maxParallelTools    int
	state               map[string]any
	instructionProvider func(ctx context.Context, state map[string]any) (string, error)
	handoffs            []HandoffConfig
	taskManager         *TaskManager
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
		maxIterations: 0,
		autoExecute:   true,
		parallelTools: true,
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

	if a.taskManager != nil {
		allTools = append(allTools, createTaskTools()...)
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

	systemPrompt, err := a.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = a.llm.Model().ID
		messages = append(messages, sysMsg)
	}

	if a.session != nil {
		sessionMessages, err := a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
		messages = append(messages, sessionMessages...)
	}

	userMsg := message.NewUserMessage(userMessage)
	userMsg.Model = a.llm.Model().ID
	messages = append(messages, userMsg)

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
			SystemPrompt: systemPrompt,
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

func (a *Agent) resolveSystemPrompt(ctx context.Context) (string, error) {
	if a.instructionProvider != nil {
		return a.instructionProvider(ctx, a.state)
	}

	if a.systemPrompt == "" {
		return "", nil
	}

	return prompt.Process(a.systemPrompt, a.state)
}

func (a *Agent) buildMessages(ctx context.Context, userMessage string) ([]message.Message, error) {
	var messages []message.Message

	systemPrompt, err := a.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

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

	var sessionMessages []message.Message
	if a.session != nil {
		var err error
		sessionMessages, err = a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = a.llm.Model().ID
		messages = append(messages, sysMsg)

		if a.session != nil && len(sessionMessages) == 0 {
			if err := a.session.AddMessages(ctx, []message.Message{sysMsg}); err != nil {
				return nil, err
			}
		}
	}

	messages = append(messages, sessionMessages...)

	userMsg := message.NewUserMessage(userMessage)
	userMsg.Model = a.llm.Model().ID
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

func (a *Agent) executeSingleTool(ctx context.Context, registry *tool.Registry, tc message.ToolCall) ToolExecutionResult {
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

	return result
}

func (a *Agent) executeTools(ctx context.Context, toolCalls []message.ToolCall) []ToolExecutionResult {
	registry := tool.NewRegistry()
	for _, t := range a.getTools() {
		registry.Register(t)
	}

	results := make([]ToolExecutionResult, len(toolCalls))

	if !a.parallelTools {
		for i, tc := range toolCalls {
			results[i] = a.executeSingleTool(ctx, registry, tc)
		}
		return results
	}

	var wg sync.WaitGroup
	var sem chan struct{}

	if a.maxParallelTools > 0 {
		sem = make(chan struct{}, a.maxParallelTools)
	}

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call message.ToolCall) {
			defer wg.Done()

			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			results[idx] = a.executeSingleTool(ctx, registry, call)
		}(i, tc)
	}

	wg.Wait()
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
// If handoffs are configured, the active agent may change mid-conversation.
func (a *Agent) Chat(ctx context.Context, userMessage string, opts ...ChatOption) (*ChatResponse, error) {
	cfg := applyChatOptions(opts)

	if a.taskManager != nil {
		ctx = withTaskManager(ctx, a.taskManager)
		defer func() {
			a.taskManager.CancelAll()
			a.taskManager.WaitAll()
		}()
	}

	messages, err := a.buildMessages(ctx, userMessage)
	if err != nil {
		return nil, err
	}

	activeAgent := a
	allTools := activeAgent.getTools()
	iteration := 0

	maxIter := activeAgent.maxIterations
	if cfg.maxIterations > 0 {
		maxIter = cfg.maxIterations
	}

	for {
		resp, err := activeAgent.llm.SendMessages(ctx, messages, allTools)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 || !activeAgent.autoExecute || (maxIter > 0 && iteration >= maxIter) {
			if activeAgent.session != nil && resp.Content != "" {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.Model = activeAgent.llm.Model().ID
				assistantMsg.AppendContent(resp.Content)
				if err := activeAgent.session.AddMessages(ctx, []message.Message{assistantMsg}); err != nil {
					return nil, err
				}
			}

			if activeAgent.autoExtract && activeAgent.session != nil {
				go activeAgent.extractAndStoreMemories(context.Background())
			}

			chatResp := &ChatResponse{
				Content:      resp.Content,
				ToolCalls:    resp.ToolCalls,
				Usage:        resp.Usage,
				FinishReason: resp.FinishReason,
			}
			if activeAgent != a {
				chatResp.AgentName = findAgentName(a, activeAgent)
			}
			return chatResp, nil
		}

		assistantMsg := message.NewAssistantMessage()
		assistantMsg.Model = activeAgent.llm.Model().ID
		if resp.Content != "" {
			assistantMsg.AppendContent(resp.Content)
		}
		assistantMsg.AppendToolCalls(resp.ToolCalls)
		messages = append(messages, assistantMsg)

		toolResults := activeAgent.executeTools(ctx, resp.ToolCalls)

		toolMsg := message.Message{Role: message.Tool, Model: activeAgent.llm.Model().ID, CreatedAt: time.Now().UnixNano()}
		for _, result := range toolResults {
			toolMsg.AddToolResult(message.ToolResult{
				ToolCallID: result.ToolCallID,
				Name:       result.ToolName,
				Content:    result.Output,
				IsError:    result.IsError,
			})
		}
		messages = append(messages, toolMsg)

		if activeAgent.session != nil {
			if err := activeAgent.session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg}); err != nil {
				return nil, err
			}
		}

		if handoff := detectHandoff(resp.ToolCalls, activeAgent.handoffs); handoff != nil {
			activeAgent = handoff.Agent
			messages, err = rebuildMessagesForHandoff(ctx, activeAgent, messages)
			if err != nil {
				return nil, fmt.Errorf("handoff to %s failed: %w", handoff.Name, err)
			}
			allTools = activeAgent.getTools()
			iteration = 0
			continue
		}

		iteration++
	}
}

// ChatStream sends a message to the agent and returns a channel of streaming events.
// Events include content deltas, tool calls, handoff notifications, and the final response.
// The channel is closed when the response is complete or an error occurs.
func (a *Agent) ChatStream(ctx context.Context, userMessage string, opts ...ChatOption) <-chan ChatEvent {
	eventChan := make(chan ChatEvent)

	go func() {
		defer close(eventChan)

		if a.taskManager != nil {
			ctx = withTaskManager(ctx, a.taskManager)
			defer func() {
				a.taskManager.CancelAll()
				a.taskManager.WaitAll()
			}()
		}

		messages, err := a.buildMessages(ctx, userMessage)
		if err != nil {
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return
		}

		cfg := applyChatOptions(opts)

		activeAgent := a
		allTools := activeAgent.getTools()
		iteration := 0

		maxIter := activeAgent.maxIterations
		if cfg.maxIterations > 0 {
			maxIter = cfg.maxIterations
		}

		for {
			var fullContent string
			var toolCalls []message.ToolCall
			var finalResponse *llm.LLMResponse

			for event := range activeAgent.llm.StreamResponse(ctx, messages, allTools) {
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

			if len(toolCalls) == 0 || !activeAgent.autoExecute || (maxIter > 0 && iteration >= maxIter) {
				if activeAgent.session != nil && fullContent != "" {
					assistantMsg := message.NewAssistantMessage()
					assistantMsg.Model = activeAgent.llm.Model().ID
					assistantMsg.AppendContent(fullContent)
					_ = activeAgent.session.AddMessages(ctx, []message.Message{assistantMsg})
				}

				if activeAgent.autoExtract && activeAgent.session != nil {
					go activeAgent.extractAndStoreMemories(context.Background())
				}

				var usage llm.TokenUsage
				var finishReason message.FinishReason
				if finalResponse != nil {
					usage = finalResponse.Usage
					finishReason = finalResponse.FinishReason
				}

				chatResp := &ChatResponse{
					Content:      fullContent,
					ToolCalls:    toolCalls,
					Usage:        usage,
					FinishReason: finishReason,
				}
				if activeAgent != a {
					chatResp.AgentName = findAgentName(a, activeAgent)
				}

				eventChan <- ChatEvent{
					Type:     types.EventComplete,
					Response: chatResp,
				}
				return
			}

			assistantMsg := message.NewAssistantMessage()
			assistantMsg.Model = activeAgent.llm.Model().ID
			if fullContent != "" {
				assistantMsg.AppendContent(fullContent)
			}
			assistantMsg.AppendToolCalls(toolCalls)
			messages = append(messages, assistantMsg)

			toolResults := activeAgent.executeTools(ctx, toolCalls)

			for _, result := range toolResults {
				eventChan <- ChatEvent{
					Type:       types.EventToolUseStop,
					ToolResult: &result,
				}
			}

			toolMsg := message.Message{Role: message.Tool, Model: activeAgent.llm.Model().ID, CreatedAt: time.Now().UnixNano()}
			for _, result := range toolResults {
				toolMsg.AddToolResult(message.ToolResult{
					ToolCallID: result.ToolCallID,
					Name:       result.ToolName,
					Content:    result.Output,
					IsError:    result.IsError,
				})
			}
			messages = append(messages, toolMsg)

			if activeAgent.session != nil {
				_ = activeAgent.session.AddMessages(ctx, []message.Message{assistantMsg, toolMsg})
			}

			if handoff := detectHandoff(toolCalls, activeAgent.handoffs); handoff != nil {
				eventChan <- ChatEvent{
					Type:      types.EventHandoff,
					AgentName: handoff.Name,
				}

				activeAgent = handoff.Agent
				messages, err = rebuildMessagesForHandoff(ctx, activeAgent, messages)
				if err != nil {
					eventChan <- ChatEvent{Type: types.EventError, Error: err}
					return
				}
				allTools = activeAgent.getTools()
				iteration = 0
				continue
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
