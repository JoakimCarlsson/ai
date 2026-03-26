package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

func (a *Agent) executeSingleTool(
	ctx context.Context,
	registry *tool.Registry,
	tc message.ToolCall,
) ToolExecutionResult {
	taskID, agentName, branch := a.hookContext(ctx)
	hookTC := ToolUseContext{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Input:      tc.Input,
		AgentName:  agentName,
		TaskID:     taskID,
		Branch:     branch,
	}

	preResult, err := runPreToolUse(ctx, a.hooks, hookTC)
	if err != nil || preResult.Action == HookDeny {
		reason := preResult.DenyReason
		if err != nil {
			reason = err.Error()
		}
		return ToolExecutionResult{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Input:      tc.Input,
			Output:     "Tool call denied: " + reason,
			IsError:    true,
		}
	}
	if preResult.Action == HookModify {
		tc.Input = preResult.Input
	}

	if a.confirmationProvider != nil {
		if t, ok := registry.Get(tc.Name); ok && t.Info().RequireConfirmation {
			req := tool.ConfirmationRequest{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Input:      tc.Input,
			}
			if eventChan := confirmationChanFromContext(ctx); eventChan != nil {
				eventChan <- ChatEvent{
					Type:                types.EventConfirmationRequired,
					ConfirmationRequest: &req,
				}
			}
			approved, providerErr := a.confirmationProvider(ctx, req)
			if providerErr != nil {
				return ToolExecutionResult{
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
					Input:      tc.Input,
					Output: fmt.Sprintf(
						"Confirmation error: %v",
						providerErr,
					),
					IsError: true,
				}
			}
			if !approved {
				return ToolExecutionResult{
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
					Input:      tc.Input,
					Output:     "Tool call requires confirmation — rejected by user",
					IsError:    true,
				}
			}
		}
	}

	execCtx := ctx
	if a.confirmationProvider != nil {
		handler := func(hint string, payload any) error {
			req := tool.ConfirmationRequest{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Input:      tc.Input,
				Hint:       hint,
				Payload:    payload,
			}
			if eventChan := confirmationChanFromContext(ctx); eventChan != nil {
				eventChan <- ChatEvent{
					Type:                types.EventConfirmationRequired,
					ConfirmationRequest: &req,
				}
			}
			approved, providerErr := a.confirmationProvider(ctx, req)
			if providerErr != nil {
				return fmt.Errorf(
					"confirmation provider error: %w",
					providerErr,
				)
			}
			if !approved {
				return tool.ErrConfirmationRejected
			}
			return nil
		}
		execCtx = tool.WithConfirmationHandler(execCtx, handler)
	}

	start := time.Now()
	resp, execErr := registry.Execute(execCtx, tool.Call{
		ID:    tc.ID,
		Name:  tc.Name,
		Input: tc.Input,
	})
	elapsed := time.Since(start)

	result := ToolExecutionResult{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Input:      tc.Input,
		IsError:    resp.IsError || execErr != nil,
		Duration:   elapsed,
	}

	if execErr != nil {
		if errors.Is(execErr, tool.ErrConfirmationRejected) {
			result.Output = "Tool execution halted — confirmation rejected by user"
		} else {
			result.Output = execErr.Error()
		}
	} else {
		result.Output = resp.Content
	}

	postResult, _ := runPostToolUse(ctx, a.hooks, PostToolUseContext{
		ToolUseContext: hookTC,
		Output:         result.Output,
		IsError:        result.IsError,
		Duration:       elapsed,
	})
	if postResult.Action == HookModify {
		result.Output = postResult.Output
	}

	return result
}

func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []message.ToolCall,
) []ToolExecutionResult {
	registry := tool.NewRegistry()
	for _, t := range a.getToolsWithContext(ctx) {
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
