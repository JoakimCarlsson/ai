package voice

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// runToolCalls executes each tool call sequentially against the agent's
// configured tool list. For every call it emits EventToolCallStart before
// invoking the tool and EventToolCallEnd after, runs the configured hooks
// (PreToolUse / OnToolError / PostToolUse), and appends a message.Tool
// history entry carrying the final ToolResult.
//
// Unknown tools and tool.Run errors are surfaced as a tool.NewTextErrorResponse
// (and routed through OnToolError if any hooks are configured) rather than
// aborting the turn so the LLM can recover on the next iteration.
func runToolCalls(
	ctx context.Context,
	v *VoiceAgent,
	calls []message.ToolCall,
	history *[]message.Message,
	emit func(Event),
) error {
	for _, call := range calls {
		emit(Event{
			Type:      EventToolCallStart,
			Timestamp: time.Now(),
			ToolCall:  &call,
		})

		input := call.Input
		toolUse := ToolUseContext{
			ConversationID: conversationIDFromCtx(ctx),
			ToolCallID:     call.ID,
			ToolName:       call.Name,
			Input:          input,
		}

		if len(v.hooks) > 0 {
			pre, err := runPreToolUse(ctx, v.hooks, toolUse)
			if err != nil {
				return err
			}
			if pre.Action == HookDeny {
				appendToolDeny(call, pre.DenyReason, history, emit)
				continue
			}
			if pre.Action == HookModify {
				input = pre.Input
				toolUse.Input = pre.Input
			}
		}

		start := time.Now()
		t := findTool(v.tools, call.Name)
		var resp tool.Response
		var runErr error
		if t == nil {
			resp = tool.NewTextErrorResponse("tool not available: " + call.Name)
		} else {
			resp, runErr = t.Run(ctx, tool.Call{
				ID:    call.ID,
				Name:  call.Name,
				Input: input,
			})
			if runErr != nil {
				resp = tool.NewTextErrorResponse(runErr.Error())
			}
		}
		elapsed := time.Since(start)

		isError := resp.IsError || runErr != nil
		output := resp.Content

		if isError && len(v.hooks) > 0 {
			errCtx := ToolErrorContext{
				ToolUseContext: toolUse,
				Error:          runErr,
				Duration:       elapsed,
			}
			recoverRes, err := runOnToolError(ctx, v.hooks, errCtx)
			if err != nil {
				return err
			}
			if recoverRes.Action == HookModify {
				output = recoverRes.Output
				isError = false
			}
		}

		if !isError && len(v.hooks) > 0 {
			postCtx := PostToolUseContext{
				ToolUseContext: toolUse,
				Output:         output,
				IsError:        false,
				Duration:       elapsed,
			}
			post, err := runPostToolUse(ctx, v.hooks, postCtx)
			if err != nil {
				return err
			}
			if post.Action == HookModify {
				output = post.Output
			}
		}

		emit(Event{
			Type:      EventToolCallEnd,
			Timestamp: time.Now(),
			ToolCall:  &call,
			ToolResult: &ToolExecutionResult{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Input:      input,
				Output:     output,
				IsError:    isError,
				Duration:   elapsed,
			},
		})

		*history = append(*history, message.NewMessage(message.Tool, []message.ContentPart{
			message.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    output,
				IsError:    isError,
			},
		}))
	}
	return nil
}

func appendToolDeny(
	call message.ToolCall,
	reason string,
	history *[]message.Message,
	emit func(Event),
) {
	if reason == "" {
		reason = "tool call denied by hook"
	}
	emit(Event{
		Type:      EventToolCallEnd,
		Timestamp: time.Now(),
		ToolCall:  &call,
		ToolResult: &ToolExecutionResult{
			ToolCallID: call.ID,
			ToolName:   call.Name,
			Input:      call.Input,
			Output:     reason,
			IsError:    true,
		},
	})
	*history = append(*history, message.NewMessage(message.Tool, []message.ContentPart{
		message.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    reason,
			IsError:    true,
		},
	}))
}

func findTool(tools []tool.BaseTool, name string) tool.BaseTool {
	for _, t := range tools {
		if t.Info().Name == name {
			return t
		}
	}
	return nil
}
