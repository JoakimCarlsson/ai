package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/tool"
)

type getTaskResultInput struct {
	TaskID  string `json:"task_id" desc:"The ID of the background task to check"`
	Wait    bool   `json:"wait" desc:"If true, block until the task completes" required:"false"`
	Timeout int    `json:"timeout" desc:"Max wait time in milliseconds. 0 means no timeout." required:"false"`
}

type getTaskResultTool struct{}

func (t *getTaskResultTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(
		"get_task_result",
		"Check the status or wait for the result of a background sub-agent task. Use the task_id returned when you launched the task with background: true.",
		getTaskResultInput{},
	)
}

func (t *getTaskResultTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input getTaskResultInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}
	if input.TaskID == "" {
		return tool.NewTextErrorResponse("task_id is required"), nil
	}

	tm := taskManagerFromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no task manager available"), nil
	}

	bt, err := tm.GetResult(ctx, input.TaskID, input.Wait, time.Duration(input.Timeout)*time.Millisecond)
	if err != nil {
		return tool.NewTextErrorResponse(fmt.Sprintf("failed to get task result: %s", err.Error())), nil
	}

	type taskResultOutput struct {
		TaskID    string     `json:"task_id"`
		AgentName string     `json:"agent_name"`
		Status    TaskStatus `json:"status"`
		Result    string     `json:"result,omitempty"`
		Error     string     `json:"error,omitempty"`
	}

	return tool.NewJSONResponse(taskResultOutput{
		TaskID:    bt.ID,
		AgentName: bt.AgentName,
		Status:    bt.Status,
		Result:    bt.Result,
		Error:     bt.Error,
	}), nil
}

type stopTaskInput struct {
	TaskID string `json:"task_id" desc:"The ID of the background task to cancel"`
}

type stopTaskTool struct{}

func (t *stopTaskTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(
		"stop_task",
		"Cancel a running background sub-agent task.",
		stopTaskInput{},
	)
}

func (t *stopTaskTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input stopTaskInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}
	if input.TaskID == "" {
		return tool.NewTextErrorResponse("task_id is required"), nil
	}

	tm := taskManagerFromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no task manager available"), nil
	}

	if err := tm.Stop(input.TaskID); err != nil {
		return tool.NewTextErrorResponse(fmt.Sprintf("failed to stop task: %s", err.Error())), nil
	}

	type stopTaskOutput struct {
		TaskID  string `json:"task_id"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	return tool.NewJSONResponse(stopTaskOutput{
		TaskID:  input.TaskID,
		Status:  "cancelled",
		Message: fmt.Sprintf("Task %s has been cancelled.", input.TaskID),
	}), nil
}

type listTasksInput struct{}

type listTasksTool struct{}

func (t *listTasksTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(
		"list_tasks",
		"List all background sub-agent tasks and their current status.",
		listTasksInput{},
	)
}

func (t *listTasksTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	tm := taskManagerFromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no task manager available"), nil
	}

	tasks := tm.ListAll()

	type taskSummary struct {
		TaskID    string     `json:"task_id"`
		AgentName string     `json:"agent_name"`
		Status    TaskStatus `json:"status"`
	}

	summaries := make([]taskSummary, len(tasks))
	for i, bt := range tasks {
		summaries[i] = taskSummary{
			TaskID:    bt.ID,
			AgentName: bt.AgentName,
			Status:    bt.Status,
		}
	}

	return tool.NewJSONResponse(summaries), nil
}

func createTaskTools() []tool.BaseTool {
	return []tool.BaseTool{
		&getTaskResultTool{},
		&stopTaskTool{},
		&listTasksTool{},
	}
}
