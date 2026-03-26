package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
)

// Chat sends a message to the agent and returns the response.
// If the agent has tools configured, it will automatically execute them.
// If memory is configured, relevant memories are injected into the context.
// If a session is configured, the conversation history is persisted.
// If handoffs are configured, the active agent may change mid-conversation.
func (a *Agent) Chat(
	ctx context.Context,
	userMessage string,
	opts ...ChatOption,
) (*ChatResponse, error) {
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

	return a.runLoop(ctx, messages, cfg)
}

// Continue resumes the agent loop with externally-executed tool results.
// Use this after a Chat() call returned pending ToolCalls (e.g. with autoExecute disabled
// or after hitting the max iteration limit). Requires a session to be configured.
func (a *Agent) Continue(
	ctx context.Context,
	toolResults []message.ToolResult,
	opts ...ChatOption,
) (*ChatResponse, error) {
	if a.session == nil {
		return nil, fmt.Errorf(
			"agent: Continue requires a session to restore conversation state",
		)
	}
	if len(toolResults) == 0 {
		return nil, fmt.Errorf(
			"agent: Continue requires at least one tool result",
		)
	}

	cfg := applyChatOptions(opts)

	if a.taskManager != nil {
		ctx = withTaskManager(ctx, a.taskManager)
		defer func() {
			a.taskManager.CancelAll()
			a.taskManager.WaitAll()
		}()
	}

	messages, err := a.buildContinueMessages(ctx)
	if err != nil {
		return nil, err
	}

	toolMsg := message.Message{
		Role:      message.Tool,
		Model:     a.llm.Model().ID,
		CreatedAt: time.Now().UnixNano(),
	}
	for _, result := range toolResults {
		toolMsg.AddToolResult(result)
	}
	messages = append(messages, toolMsg)

	if err := a.session.AddMessages(ctx, []message.Message{toolMsg}); err != nil {
		return nil, err
	}

	return a.runLoop(ctx, messages, cfg)
}

func (a *Agent) runLoop(
	ctx context.Context,
	messages []message.Message,
	cfg chatConfig,
) (*ChatResponse, error) {
	startTime := time.Now()
	var totalUsage llm.TokenUsage
	var totalToolCalls int
	var turns int

	activeAgent := a
	allTools := activeAgent.getToolsWithContext(ctx)
	iteration := 0

	maxIter := activeAgent.maxIterations
	if cfg.maxIterations > 0 {
		maxIter = cfg.maxIterations
	}

	for {
		turnStart := time.Now()

		taskID, agentName, branch := activeAgent.hookContext(ctx)
		mcResult, err := runPreModelCall(
			ctx,
			activeAgent.hooks,
			ModelCallContext{
				Messages:  messages,
				Tools:     allTools,
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("pre-model-call hook: %w", err)
		}
		if mcResult.Action == HookModify {
			messages = mcResult.Messages
			allTools = mcResult.Tools
		}

		resp, err := activeAgent.llm.SendMessages(ctx, messages, allTools)

		mrResult, hookErr := runPostModelCall(
			ctx,
			activeAgent.hooks,
			ModelResponseContext{
				Response:  resp,
				Duration:  time.Since(turnStart),
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Error:     err,
			},
		)
		if err != nil {
			return nil, err
		}
		if hookErr != nil {
			return nil, fmt.Errorf("post-model-call hook: %w", hookErr)
		}
		if mrResult.Action == HookModify && mrResult.Response != nil {
			resp = mrResult.Response
		}

		turns++
		totalUsage.Add(resp.Usage)

		if len(resp.ToolCalls) == 0 || !activeAgent.autoExecute ||
			(maxIter > 0 && iteration >= maxIter) {
			if activeAgent.session != nil {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.Model = activeAgent.llm.Model().ID
				if resp.Content != "" {
					assistantMsg.AppendContent(resp.Content)
				}
				if len(resp.ToolCalls) > 0 {
					assistantMsg.AppendToolCalls(resp.ToolCalls)
				}
				if resp.Content != "" || len(resp.ToolCalls) > 0 {
					if err := activeAgent.session.AddMessages(ctx, []message.Message{assistantMsg}); err != nil {
						return nil, err
					}
				}
			}

			if activeAgent.autoExtract && activeAgent.session != nil {
				go activeAgent.extractAndStoreMemories(context.Background())
			}

			chatResp := &ChatResponse{
				Content:        resp.Content,
				ToolCalls:      resp.ToolCalls,
				Usage:          totalUsage,
				FinishReason:   resp.FinishReason,
				TotalToolCalls: totalToolCalls,
				TotalDuration:  time.Since(startTime),
				TotalTurns:     turns,
			}
			if activeAgent != a {
				chatResp.AgentName = findAgentName(a, activeAgent)
			}
			return chatResp, nil
		}

		totalToolCalls += len(resp.ToolCalls)

		assistantMsg := message.NewAssistantMessage()
		assistantMsg.Model = activeAgent.llm.Model().ID
		if resp.Content != "" {
			assistantMsg.AppendContent(resp.Content)
		}
		assistantMsg.AppendToolCalls(resp.ToolCalls)
		messages = append(messages, assistantMsg)

		toolResults := activeAgent.executeTools(ctx, resp.ToolCalls)

		toolMsg := message.Message{
			Role:      message.Tool,
			Model:     activeAgent.llm.Model().ID,
			CreatedAt: time.Now().UnixNano(),
		}
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
			messages, err = rebuildMessagesForHandoff(
				ctx,
				activeAgent,
				messages,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"handoff to %s failed: %w",
					handoff.Name,
					err,
				)
			}
			allTools = activeAgent.getToolsWithContext(ctx)
			iteration = 0
			continue
		}

		iteration++
	}
}
