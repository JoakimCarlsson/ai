package voice

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// runToolCalls executes each tool call sequentially against the given tool
// list. For every call it emits an EventToolCallStart before invoking the tool
// and an EventToolCallEnd after. The result is appended to history as a
// message.Tool message carrying message.ToolResult parts.
//
// Unknown tools and tool.Run errors are surfaced as a tool.NewTextErrorResponse
// rather than aborting the turn so the LLM can recover on the next iteration.
func runToolCalls(
	ctx context.Context,
	tools []tool.BaseTool,
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

		start := time.Now()
		t := findTool(tools, call.Name)
		var resp tool.Response
		var runErr error
		if t == nil {
			resp = tool.NewTextErrorResponse("tool not available: " + call.Name)
		} else {
			resp, runErr = t.Run(ctx, tool.Call{
				ID:    call.ID,
				Name:  call.Name,
				Input: call.Input,
			})
			if runErr != nil {
				resp = tool.NewTextErrorResponse(runErr.Error())
			}
		}
		elapsed := time.Since(start)

		emit(Event{
			Type:      EventToolCallEnd,
			Timestamp: time.Now(),
			ToolCall:  &call,
			ToolResult: &ToolExecutionResult{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Input:      call.Input,
				Output:     resp.Content,
				IsError:    resp.IsError || runErr != nil,
				Duration:   elapsed,
			},
		})

		*history = append(*history, message.NewMessage(message.Tool, []message.ContentPart{
			message.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    resp.Content,
				IsError:    resp.IsError || runErr != nil,
			},
		}))
	}
	return nil
}

func findTool(tools []tool.BaseTool, name string) tool.BaseTool {
	for _, t := range tools {
		if t.Info().Name == name {
			return t
		}
	}
	return nil
}
