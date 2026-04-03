package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tracing"
	"github.com/joakimcarlsson/ai/types"
)

// ChatStream sends a message to the agent and returns a channel of streaming events.
// Events include content deltas, tool calls, handoff notifications, and the final response.
// The channel is closed when the response is complete or an error occurs.
func (a *Agent) ChatStream(
	ctx context.Context,
	userMessage string,
	opts ...ChatOption,
) <-chan ChatEvent {
	eventChan := make(chan ChatEvent)

	go func() {
		defer close(eventChan)

		startTime := time.Now()
		taskID, agentName, branch := a.hookContext(ctx)

		ctx, span := tracing.StartAgentSpan(ctx, agentName)
		defer span.End()

		runBeforeRun(ctx, a.hooks, RunContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
			Input:     userMessage,
		})

		if a.taskManager != nil {
			ctx = withTaskManager(ctx, a.taskManager)
			defer func() {
				a.taskManager.CancelAll()
				a.taskManager.WaitAll()
			}()
		}

		if a.team != nil {
			ctx = team.WithContext(ctx, a.team)
			ctx = team.WithLeadContext(ctx)
			ctx = withTeamEventChan(ctx, eventChan)
			ctx = withTeamHooks(ctx, a.hooks)
			a.team.Mailbox.RegisterRecipient("__lead__")
			defer func() {
				a.team.WaitAll()
				a.team.Mailbox.Close()
			}()
		}

		umResult, umErr := runOnUserMessage(ctx, a.hooks, UserMessageContext{
			Message:   userMessage,
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
		})
		if umErr != nil {
			tracing.SetError(span, umErr)
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("on-user-message hook: %w", umErr),
			}
			return
		}
		if umResult.Action == HookDeny {
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("message denied: %s", umResult.DenyReason),
			}
			return
		}
		if umResult.Action == HookModify {
			userMessage = umResult.Message
		}

		baResult, baErr := runBeforeAgent(ctx, a.hooks, LifecycleContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
			Input:     userMessage,
		})
		if baErr != nil {
			tracing.SetError(span, baErr)
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("before-agent hook: %w", baErr),
			}
			return
		}
		if baResult.Action == HookDeny ||
			(baResult.Action == HookModify && baResult.Response != nil) {
			resp := baResult.Response
			runAfterAgent(ctx, a.hooks, LifecycleContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Response:  resp,
			})
			runAfterRun(ctx, a.hooks, RunContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Input:     userMessage,
				Response:  resp,
				Duration:  time.Since(startTime),
			})
			eventChan <- ChatEvent{
				Type:     types.EventComplete,
				Response: resp,
			}
			return
		}

		messages, err := a.buildMessages(ctx, userMessage)
		if err != nil {
			tracing.SetError(span, err)
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return
		}

		cfg := applyChatOptions(opts)
		resp, loopErr := a.runLoopStream(ctx, messages, cfg, eventChan)

		if loopErr == nil && resp != nil {
			aaResult, aaErr := runAfterAgent(ctx, a.hooks, LifecycleContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Response:  resp,
			})
			if aaErr != nil {
				tracing.SetError(span, aaErr)
				eventChan <- ChatEvent{
					Type:  types.EventError,
					Error: fmt.Errorf("after-agent hook: %w", aaErr),
				}
				runAfterRun(ctx, a.hooks, RunContext{
					AgentName: agentName,
					TaskID:    taskID,
					Branch:    branch,
					Input:     userMessage,
					Error:     aaErr,
					Duration:  time.Since(startTime),
				})
				return
			}
			if aaResult.Action == HookModify && aaResult.Response != nil {
				resp = aaResult.Response
			}
			tracing.SetResponseAttrs(span,
				tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
				tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
				tracing.AttrAgentTotalTurns.Int(resp.TotalTurns),
				tracing.AttrAgentTotalToolCalls.Int(resp.TotalToolCalls),
			)
			eventChan <- ChatEvent{
				Type:     types.EventComplete,
				Response: resp,
			}
		}

		if loopErr != nil {
			tracing.SetError(span, loopErr)
		}

		runAfterRun(ctx, a.hooks, RunContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
			Input:     userMessage,
			Response:  resp,
			Error:     loopErr,
			Duration:  time.Since(startTime),
		})
	}()

	return eventChan
}

// ContinueStream is the streaming variant of Continue. It resumes the agent loop
// with externally-executed tool results and returns a channel of streaming events.
func (a *Agent) ContinueStream(
	ctx context.Context,
	toolResults []message.ToolResult,
	opts ...ChatOption,
) <-chan ChatEvent {
	eventChan := make(chan ChatEvent)

	go func() {
		defer close(eventChan)

		if a.session == nil {
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("agent: ContinueStream requires a session to restore conversation state"),
			}
			return
		}
		if len(toolResults) == 0 {
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("agent: ContinueStream requires at least one tool result"),
			}
			return
		}

		startTime := time.Now()
		taskID, agentName, branch := a.hookContext(ctx)

		ctx, span := tracing.StartAgentSpan(ctx, agentName)
		defer span.End()

		runBeforeRun(ctx, a.hooks, RunContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
		})

		if a.taskManager != nil {
			ctx = withTaskManager(ctx, a.taskManager)
			defer func() {
				a.taskManager.CancelAll()
				a.taskManager.WaitAll()
			}()
		}

		if a.team != nil {
			ctx = team.WithContext(ctx, a.team)
			ctx = team.WithLeadContext(ctx)
			ctx = withTeamEventChan(ctx, eventChan)
			ctx = withTeamHooks(ctx, a.hooks)
			a.team.Mailbox.RegisterRecipient("__lead__")
			defer func() {
				a.team.WaitAll()
				a.team.Mailbox.Close()
			}()
		}

		baResult, baErr := runBeforeAgent(ctx, a.hooks, LifecycleContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
		})
		if baErr != nil {
			tracing.SetError(span, baErr)
			eventChan <- ChatEvent{
				Type:  types.EventError,
				Error: fmt.Errorf("before-agent hook: %w", baErr),
			}
			return
		}
		if baResult.Action == HookDeny ||
			(baResult.Action == HookModify && baResult.Response != nil) {
			resp := baResult.Response
			runAfterAgent(ctx, a.hooks, LifecycleContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Response:  resp,
			})
			runAfterRun(ctx, a.hooks, RunContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Response:  resp,
				Duration:  time.Since(startTime),
			})
			eventChan <- ChatEvent{
				Type:     types.EventComplete,
				Response: resp,
			}
			return
		}

		messages, err := a.buildContinueMessages(ctx)
		if err != nil {
			tracing.SetError(span, err)
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return
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
			tracing.SetError(span, err)
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return
		}

		cfg := applyChatOptions(opts)
		resp, loopErr := a.runLoopStream(ctx, messages, cfg, eventChan)

		if loopErr == nil && resp != nil {
			aaResult, aaErr := runAfterAgent(ctx, a.hooks, LifecycleContext{
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Response:  resp,
			})
			if aaErr != nil {
				tracing.SetError(span, aaErr)
				eventChan <- ChatEvent{
					Type:  types.EventError,
					Error: fmt.Errorf("after-agent hook: %w", aaErr),
				}
				runAfterRun(ctx, a.hooks, RunContext{
					AgentName: agentName,
					TaskID:    taskID,
					Branch:    branch,
					Error:     aaErr,
					Duration:  time.Since(startTime),
				})
				return
			}
			if aaResult.Action == HookModify && aaResult.Response != nil {
				resp = aaResult.Response
			}
			tracing.SetResponseAttrs(span,
				tracing.AttrUsageInputTokens.Int64(resp.Usage.InputTokens),
				tracing.AttrUsageOutputTokens.Int64(resp.Usage.OutputTokens),
				tracing.AttrAgentTotalTurns.Int(resp.TotalTurns),
				tracing.AttrAgentTotalToolCalls.Int(resp.TotalToolCalls),
			)
			eventChan <- ChatEvent{
				Type:     types.EventComplete,
				Response: resp,
			}
		}

		if loopErr != nil {
			tracing.SetError(span, loopErr)
		}

		runAfterRun(ctx, a.hooks, RunContext{
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
			Response:  resp,
			Error:     loopErr,
			Duration:  time.Since(startTime),
		})
	}()

	return eventChan
}

func (a *Agent) runLoopStream(
	ctx context.Context,
	messages []message.Message,
	cfg chatConfig,
	eventChan chan<- ChatEvent,
) (*ChatResponse, error) {
	startTime := time.Now()
	var totalUsage llm.TokenUsage
	var totalToolCalls int
	var turns int

	activeAgent := a
	iteration := 0

	maxIter := activeAgent.maxIterations
	if cfg.maxIterations > 0 {
		maxIter = cfg.maxIterations
	}

	for {
		var fullContent string
		var toolCalls []message.ToolCall
		var finalResponse *llm.Response
		seenToolStarts := make(map[string]bool)

		turnStart := time.Now()
		allTools := activeAgent.getToolsWithContext(ctx)

		taskID, agentName, branch := activeAgent.hookContext(ctx)
		mcResult, hookErr := runPreModelCall(
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
		if hookErr != nil {
			eventChan <- ChatEvent{Type: types.EventError, Error: fmt.Errorf("pre-model-call hook: %w", hookErr)}
			return nil, hookErr
		}
		if mcResult.Action == HookModify {
			messages = mcResult.Messages
			allTools = mcResult.Tools
		}

		var streamErr error
		var streamRecovered bool

		for event := range activeAgent.llm.StreamResponse(ctx, messages, allTools) {
			switch event.Type {
			case types.EventContentDelta:
				fullContent += event.Content
				eventChan <- ChatEvent{Type: types.EventContentDelta, Content: event.Content}
			case types.EventThinkingDelta:
				eventChan <- ChatEvent{Type: types.EventThinkingDelta, Thinking: event.Thinking}
			case types.EventToolUseStart,
				types.EventToolUseDelta,
				types.EventToolUseStop:
				if event.ToolCall != nil {
					if event.Type == types.EventToolUseStart {
						seenToolStarts[event.ToolCall.ID] = true
					}
					eventChan <- ChatEvent{Type: event.Type, ToolCall: event.ToolCall}
				}
			case types.EventComplete:
				if event.Response != nil {
					finalResponse = event.Response
					toolCalls = event.Response.ToolCalls
				}
			case types.EventError:
				runPostModelCall(ctx, activeAgent.hooks, ModelResponseContext{
					Duration:  time.Since(turnStart),
					AgentName: agentName,
					TaskID:    taskID,
					Branch:    branch,
					Error:     event.Error,
				})
				meResult, meErr := runOnModelError(
					ctx,
					activeAgent.hooks,
					ModelErrorContext{
						Messages:  messages,
						Tools:     allTools,
						Error:     event.Error,
						AgentName: agentName,
						TaskID:    taskID,
						Branch:    branch,
					},
				)
				if meErr == nil && meResult.Action == HookModify &&
					meResult.Response != nil {
					finalResponse = meResult.Response
					toolCalls = meResult.Response.ToolCalls
					streamRecovered = true
				} else {
					streamErr = event.Error
				}
			}
		}

		if streamErr != nil && !streamRecovered {
			eventChan <- ChatEvent{Type: types.EventError, Error: streamErr}
			return nil, streamErr
		}

		turns++
		if finalResponse != nil {
			totalUsage.Add(finalResponse.Usage)
			if !streamRecovered {
				mrResult, hookErr := runPostModelCall(
					ctx,
					activeAgent.hooks,
					ModelResponseContext{
						Response:  finalResponse,
						Duration:  time.Since(turnStart),
						AgentName: agentName,
						TaskID:    taskID,
						Branch:    branch,
					},
				)
				if hookErr != nil {
					eventChan <- ChatEvent{Type: types.EventError, Error: fmt.Errorf("post-model-call hook: %w", hookErr)}
					return nil, hookErr
				}
				if mrResult.Action == HookModify && mrResult.Response != nil {
					finalResponse = mrResult.Response
					toolCalls = finalResponse.ToolCalls
				}
			}
		}

		if streamRecovered && finalResponse != nil {
			fullContent = finalResponse.Content
		}

		if len(toolCalls) == 0 || !activeAgent.autoExecute ||
			(maxIter > 0 && iteration >= maxIter) {
			if activeAgent.session != nil {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.Model = activeAgent.llm.Model().ID
				if fullContent != "" {
					assistantMsg.AppendContent(fullContent)
				}
				if len(toolCalls) > 0 {
					assistantMsg.AppendToolCalls(toolCalls)
				}
				if fullContent != "" || len(toolCalls) > 0 {
					_ = activeAgent.session.AddMessages(
						ctx,
						[]message.Message{assistantMsg},
					)
				}
			}

			if activeAgent.autoExtract && activeAgent.session != nil {
				go activeAgent.extractAndStoreMemories(context.Background())
			}

			var finishReason message.FinishReason
			if finalResponse != nil {
				finishReason = finalResponse.FinishReason
			}

			chatResp := &ChatResponse{
				Content:        fullContent,
				ToolCalls:      toolCalls,
				Usage:          totalUsage,
				FinishReason:   finishReason,
				TotalToolCalls: totalToolCalls,
				TotalDuration:  time.Since(startTime),
				TotalTurns:     turns,
			}
			if activeAgent != a {
				chatResp.AgentName = findAgentName(a, activeAgent)
			}

			return chatResp, nil
		}

		totalToolCalls += len(toolCalls)

		assistantMsg := message.NewAssistantMessage()
		assistantMsg.Model = activeAgent.llm.Model().ID
		if fullContent != "" {
			assistantMsg.AppendContent(fullContent)
		}
		assistantMsg.AppendToolCalls(toolCalls)
		messages = append(messages, assistantMsg)

		for i := range toolCalls {
			if !seenToolStarts[toolCalls[i].ID] {
				eventChan <- ChatEvent{
					Type:     types.EventToolUseStart,
					ToolCall: &toolCalls[i],
				}
			}
		}

		execCtx := withConfirmationChan(ctx, eventChan)
		toolResults := activeAgent.executeTools(execCtx, toolCalls)

		for _, result := range toolResults {
			eventChan <- ChatEvent{
				Type:       types.EventToolUseStop,
				ToolResult: &result,
			}
		}

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
			_ = activeAgent.session.AddMessages(
				ctx,
				[]message.Message{assistantMsg, toolMsg},
			)
		}

		if handoff := detectHandoff(toolCalls, activeAgent.handoffs); handoff != nil {
			eventChan <- ChatEvent{
				Type:      types.EventHandoff,
				AgentName: handoff.Name,
			}

			activeAgent = handoff.Agent
			var err error
			messages, err = rebuildMessagesForHandoff(
				ctx,
				activeAgent,
				messages,
			)
			if err != nil {
				eventChan <- ChatEvent{Type: types.EventError, Error: err}
				return nil, err
			}
			iteration = 0
			continue
		}

		iteration++
	}
}
