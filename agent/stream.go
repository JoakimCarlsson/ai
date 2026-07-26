package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/agent/team"
	llm "github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
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

		if err := a.session.AddMessages(
			ctx,
			[]message.Message{toolMsg},
		); err != nil {
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

func (a *Agent) executeStreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	eventChan chan<- ChatEvent,
) (*llm.Response, []message.Message, []tool.BaseTool, string, string, error) {
	var fullContent string
	var fullReasoning string
	var finalResponse *llm.Response
	var streamErr error
	var streamRecovered bool
	seenToolStarts := make(map[string]bool)

	turnStart := time.Now()
	taskID, agentName, branch := a.hookContext(ctx)
	mcResult, hookErr := runPreModelCall(
		ctx,
		a.hooks,
		ModelCallContext{
			Messages:  messages,
			Tools:     tools,
			AgentName: agentName,
			TaskID:    taskID,
			Branch:    branch,
		},
	)
	if hookErr != nil {
		err := fmt.Errorf("pre-model-call hook: %w", hookErr)
		eventChan <- ChatEvent{Type: types.EventError, Error: err}
		return nil, nil, nil, "", "", err
	}
	if mcResult.Action == HookModify {
		messages = mcResult.Messages
		tools = mcResult.Tools
	}

	for event := range a.llm.StreamResponse(ctx, messages, tools) {
		switch event.Type {
		case types.EventContentDelta:
			fullContent += event.Content
			eventChan <- ChatEvent{Type: types.EventContentDelta, Content: event.Content}
		case types.EventThinkingDelta:
			fullReasoning += event.Thinking
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
			}
		case types.EventError:
			runPostModelCall(ctx, a.hooks, ModelResponseContext{
				Duration:  time.Since(turnStart),
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
				Error:     event.Error,
			})
			meResult, meErr := runOnModelError(
				ctx,
				a.hooks,
				ModelErrorContext{
					Messages:  messages,
					Tools:     tools,
					Error:     event.Error,
					AgentName: agentName,
					TaskID:    taskID,
					Branch:    branch,
				},
			)
			if meErr == nil && meResult.Action == HookModify &&
				meResult.Response != nil {
				finalResponse = meResult.Response
				streamRecovered = true
			} else {
				streamErr = event.Error
			}
		}
	}

	if streamErr != nil && !streamRecovered {
		eventChan <- ChatEvent{Type: types.EventError, Error: streamErr}
		return nil, nil, nil, "", "", streamErr
	}

	if finalResponse != nil && !streamRecovered {
		mrResult, hookErr := runPostModelCall(
			ctx,
			a.hooks,
			ModelResponseContext{
				Response:  finalResponse,
				Duration:  time.Since(turnStart),
				AgentName: agentName,
				TaskID:    taskID,
				Branch:    branch,
			},
		)
		if hookErr != nil {
			err := fmt.Errorf("post-model-call hook: %w", hookErr)
			eventChan <- ChatEvent{Type: types.EventError, Error: err}
			return nil, nil, nil, "", "", err
		}
		if mrResult.Action == HookModify && mrResult.Response != nil {
			finalResponse = mrResult.Response
			fullContent = finalResponse.Content
			fullReasoning = finalResponse.Reasoning
		}
	}

	if streamRecovered && finalResponse != nil {
		fullContent = finalResponse.Content
		fullReasoning = finalResponse.Reasoning
	}

	if finalResponse != nil {
		for i := range finalResponse.ToolCalls {
			tc := &finalResponse.ToolCalls[i]
			if !seenToolStarts[tc.ID] {
				seenToolStarts[tc.ID] = true
				eventChan <- ChatEvent{
					Type:     types.EventToolUseStart,
					ToolCall: tc,
				}
			}
		}
	}

	return finalResponse, messages, tools, fullContent, fullReasoning, nil
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
	totalIterations := 0

	maxIter := activeAgent.maxIterations
	if cfg.maxIterations > 0 {
		maxIter = cfg.maxIterations
	}

	for {
		var pendingSteeringMessage string
		var fullContent string
		var fullReasoning string
		var toolCalls []message.ToolCall
		var finalResponse *llm.Response

		allTools := activeAgent.getToolsWithContext(ctx)

		var streamErr error
		finalResponse, messages, _, fullContent, fullReasoning, streamErr = activeAgent.executeStreamResponse(
			ctx,
			messages,
			allTools,
			eventChan,
		)
		if streamErr != nil {
			return nil, streamErr
		}
		turns++
		if finalResponse != nil {
			toolCalls = finalResponse.ToolCalls
			totalUsage.Add(finalResponse.Usage)
		}

		if (maxIter > 0 && iteration >= maxIter) && len(toolCalls) > 0 {
			if activeAgent.continuationProvider != nil {
				toolCallsCopy := make([]message.ToolCall, len(toolCalls))
				copy(toolCallsCopy, toolCalls)
				req := ContinuationRequest{
					MaxIterations:   maxIter,
					TotalIterations: totalIterations + iteration,
					ToolCalls:       toolCallsCopy,
				}

				eventChan <- ChatEvent{
					Type:                types.EventContinuationRequired,
					ContinuationRequest: &req,
				}

				contResp, pErr := activeAgent.continuationProvider(ctx, req)
				if pErr == nil && contResp.Decision == ContinuationApprove {
					totalIterations += iteration
					iteration = 0

					if contResp.DiscardToolCalls {
						assistantMsg := message.NewAssistantMessage()
						assistantMsg.Model = activeAgent.llm.Model().ID
						if fullContent != "" {
							assistantMsg.AppendContent(fullContent)
						}
						if fullReasoning != "" {
							assistantMsg.AppendReasoningContent(fullReasoning)
						}
						assistantMsg.AppendToolCalls(toolCalls)

						toolMsg := message.Message{
							Role:      message.Tool,
							Model:     activeAgent.llm.Model().ID,
							CreatedAt: time.Now().UnixNano(),
						}

						errText := contResp.ToolMessage
						if errText == "" {
							errText = "Tool execution canceled by user during continuation."
						}

						for _, tc := range toolCalls {
							toolMsg.AddToolResult(message.ToolResult{
								ToolCallID: tc.ID,
								Name:       tc.Name,
								Content:    errText,
								IsError:    true,
							})
						}

						newMessages := []message.Message{assistantMsg, toolMsg}
						if contResp.Message != "" {
							sysMsg := message.NewUserMessage(contResp.Message)
							newMessages = append(newMessages, sysMsg)
						}

						messages = append(messages, newMessages...)
						if activeAgent.session != nil {
							if err := activeAgent.session.AddMessages(
								ctx,
								newMessages,
							); err != nil {
								eventChan <- ChatEvent{Type: types.EventError, Error: err}
								return nil, err
							}
						}
						continue
					}
					if contResp.Message != "" {
						pendingSteeringMessage = contResp.Message
					}
				} else {
					var errText string
					switch {
					case pErr != nil:
						errText = pErr.Error()
					case contResp.Message != "":
						errText = contResp.Message
					case contResp.Decision == ContinuationTimeout:
						errText = "Continuation request timed out."
					default:
						errText = "Maximum iteration limit reached. Continuation declined by user."
					}

					assistantMsg := message.NewAssistantMessage()
					assistantMsg.Model = activeAgent.llm.Model().ID
					if fullContent != "" {
						assistantMsg.AppendContent(fullContent)
					}
					if fullReasoning != "" {
						assistantMsg.AppendReasoningContent(fullReasoning)
					}
					assistantMsg.AppendToolCalls(toolCalls)

					toolMsg := message.Message{
						Role:      message.Tool,
						Model:     activeAgent.llm.Model().ID,
						CreatedAt: time.Now().UnixNano(),
					}
					for _, tc := range toolCalls {
						toolMsg.AddToolResult(message.ToolResult{
							ToolCallID: tc.ID,
							Name:       tc.Name,
							Content:    errText,
							IsError:    true,
						})
					}

					sysMsg := message.NewUserMessage(
						"System Notification: " + errText,
					)

					messages = append(messages, assistantMsg, toolMsg, sysMsg)
					if activeAgent.session != nil {
						if err := activeAgent.session.AddMessages(
							ctx,
							[]message.Message{assistantMsg, toolMsg, sysMsg},
						); err != nil {
							eventChan <- ChatEvent{Type: types.EventError, Error: err}
							return nil, err
						}
					}

					var finalResp *llm.Response
					var streamErr error
					finalResp, _, _, fullContent, fullReasoning, streamErr = activeAgent.executeStreamResponse(
						ctx,
						messages,
						nil,
						eventChan,
					)
					if streamErr != nil {
						return nil, streamErr
					}
					turns++

					if activeAgent.session != nil && finalResp != nil {
						finalAssistantMsg := message.NewAssistantMessage()

						finalAssistantMsg.Model = activeAgent.llm.Model().ID
						if finalResp.Content != "" {
							finalAssistantMsg.AppendContent(finalResp.Content)
						}
						if finalResp.Reasoning != "" {
							finalAssistantMsg.AppendReasoningContent(
								finalResp.Reasoning,
							)
						}
						if finalResp.Content != "" ||
							finalResp.Reasoning != "" {
							if err := activeAgent.session.AddMessages(
								ctx,
								[]message.Message{finalAssistantMsg},
							); err != nil {
								eventChan <- ChatEvent{Type: types.EventError, Error: err}
								return nil, err
							}
						}
					}

					chatResp := &ChatResponse{
						Content:            fullContent,
						Reasoning:          fullReasoning,
						ToolCalls:          nil,
						Usage:              totalUsage,
						FinishReason:       message.FinishReasonMaxIterations,
						ProviderResponseID: "",
						TotalToolCalls:     totalToolCalls,
						TotalDuration:      time.Since(startTime),
						TotalTurns:         turns,
					}
					if finalResp != nil {
						chatResp.Usage.Add(finalResp.Usage)
						chatResp.ProviderResponseID = finalResp.ProviderResponseID
					}
					if activeAgent != a {
						chatResp.AgentName = findAgentName(a, activeAgent)
					}
					return chatResp, nil
				}
			}
		}

		if len(toolCalls) == 0 || !activeAgent.autoExecute ||
			(maxIter > 0 && iteration >= maxIter) {

			if activeAgent.session != nil {
				assistantMsg := message.NewAssistantMessage()
				assistantMsg.Model = activeAgent.llm.Model().ID
				if fullContent != "" {
					assistantMsg.AppendContent(fullContent)
				}
				if fullReasoning != "" {
					assistantMsg.AppendReasoningContent(fullReasoning)
				}
				if len(toolCalls) > 0 && !activeAgent.autoExecute {
					assistantMsg.AppendToolCalls(toolCalls)
				}
				if fullContent != "" || fullReasoning != "" ||
					len(toolCalls) > 0 && !activeAgent.autoExecute {
					if err := activeAgent.session.AddMessages(
						ctx,
						[]message.Message{assistantMsg},
					); err != nil {
						eventChan <- ChatEvent{Type: types.EventError, Error: err}
						return nil, err
					}
				}

				if pendingSteeringMessage != "" {
					sysMsg := message.NewUserMessage(pendingSteeringMessage)
					if err := activeAgent.session.AddMessages(
						ctx,
						[]message.Message{sysMsg},
					); err != nil {
						eventChan <- ChatEvent{Type: types.EventError, Error: err}
						return nil, err
					}
				}
			}

			if activeAgent.autoExtract && activeAgent.session != nil {
				go activeAgent.extractAndStoreMemories(context.Background())
			}

			var finishReason message.FinishReason
			var providerResponseID string
			if finalResponse != nil {
				finishReason = finalResponse.FinishReason
				providerResponseID = finalResponse.ProviderResponseID
			}
			if maxIter > 0 && iteration >= maxIter && len(toolCalls) > 0 {
				finishReason = message.FinishReasonMaxIterations
			}

			chatResp := &ChatResponse{
				Content:            fullContent,
				Reasoning:          fullReasoning,
				ToolCalls:          toolCalls,
				Usage:              totalUsage,
				FinishReason:       finishReason,
				ProviderResponseID: providerResponseID,
				TotalToolCalls:     totalToolCalls,
				TotalDuration:      time.Since(startTime),
				TotalTurns:         turns,
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
		if fullReasoning != "" {
			assistantMsg.AppendReasoningContent(fullReasoning)
		}
		assistantMsg.AppendToolCalls(toolCalls)
		messages = append(messages, assistantMsg)

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
			if err := activeAgent.session.AddMessages(
				ctx,
				[]message.Message{assistantMsg, toolMsg},
			); err != nil {
				eventChan <- ChatEvent{Type: types.EventError, Error: err}
				return nil, err
			}
		}

		if pendingSteeringMessage != "" {
			sysMsg := message.NewUserMessage(pendingSteeringMessage)
			messages = append(messages, sysMsg)
			if activeAgent.session != nil {
				if err := activeAgent.session.AddMessages(
					ctx,
					[]message.Message{sysMsg},
				); err != nil {
					eventChan <- ChatEvent{Type: types.EventError, Error: err}
					return nil, err
				}
			}
		}

		if handoff := detectHandoff(
			toolCalls,
			activeAgent.handoffs,
		); handoff != nil {
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
