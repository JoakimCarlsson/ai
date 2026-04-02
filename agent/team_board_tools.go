package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/tool"
)

type createBoardTaskInput struct {
	Title string `json:"title" desc:"Title of the task to create"`
}

type createBoardTaskTool struct{}

func (t *createBoardTaskTool) Info() tool.Info {
	return tool.NewInfo(
		"create_board_task",
		"Create a new task on the team's shared task board. Tasks start with 'open' status and can be claimed by any teammate.",
		createBoardTaskInput{},
	)
}

func (t *createBoardTaskTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input createBoardTaskInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.Title == "" {
		return tool.NewTextErrorResponse("title is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil || tm.TaskBoard == nil {
		return tool.NewTextErrorResponse("no task board available"), nil
	}

	_, creator, _ := taskScopeFromContext(ctx)
	if creator == "" {
		if team.IsLead(ctx) {
			creator = "__lead__"
		}
	}

	task := tm.TaskBoard.Create(input.Title, creator)
	return tool.NewJSONResponse(task), nil
}

type claimBoardTaskInput struct {
	TaskID string `json:"task_id" desc:"ID of the task to claim"`
}

type claimBoardTaskTool struct{}

func (t *claimBoardTaskTool) Info() tool.Info {
	return tool.NewInfo(
		"claim_board_task",
		"Claim an open task from the team's shared task board. Only open tasks can be claimed.",
		claimBoardTaskInput{},
	)
}

func (t *claimBoardTaskTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input claimBoardTaskInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.TaskID == "" {
		return tool.NewTextErrorResponse("task_id is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil || tm.TaskBoard == nil {
		return tool.NewTextErrorResponse("no task board available"), nil
	}

	_, assignee, _ := taskScopeFromContext(ctx)
	if assignee == "" {
		if team.IsLead(ctx) {
			assignee = "__lead__"
		}
	}

	if err := tm.TaskBoard.Claim(input.TaskID, assignee); err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to claim task: %s", err.Error()),
		), nil
	}

	return tool.NewTextResponse(
		fmt.Sprintf("Task %s claimed by %s.", input.TaskID, assignee),
	), nil
}

type completeBoardTaskInput struct {
	TaskID string `json:"task_id" desc:"ID of the task to complete"`
	Result string `json:"result"  desc:"Result or summary of the completed work"`
}

type completeBoardTaskTool struct{}

func (t *completeBoardTaskTool) Info() tool.Info {
	return tool.NewInfo(
		"complete_board_task",
		"Mark a claimed task as completed with a result. Only the assignee can complete their own tasks.",
		completeBoardTaskInput{},
	)
}

func (t *completeBoardTaskTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input completeBoardTaskInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.TaskID == "" {
		return tool.NewTextErrorResponse("task_id is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil || tm.TaskBoard == nil {
		return tool.NewTextErrorResponse("no task board available"), nil
	}

	_, assignee, _ := taskScopeFromContext(ctx)
	if assignee == "" {
		if team.IsLead(ctx) {
			assignee = "__lead__"
		}
	}

	if err := tm.TaskBoard.Complete(input.TaskID, assignee, input.Result); err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to complete task: %s", err.Error()),
		), nil
	}

	return tool.NewTextResponse(
		fmt.Sprintf("Task %s completed.", input.TaskID),
	), nil
}

type listBoardTasksInput struct{}

type listBoardTasksTool struct{}

func (t *listBoardTasksTool) Info() tool.Info {
	return tool.NewInfo(
		"list_board_tasks",
		"List all tasks on the team's shared task board with their current status and assignees.",
		listBoardTasksInput{},
	)
}

func (t *listBoardTasksTool) Run(
	ctx context.Context,
	_ tool.Call,
) (tool.Response, error) {
	tm := team.FromContext(ctx)
	if tm == nil || tm.TaskBoard == nil {
		return tool.NewTextErrorResponse("no task board available"), nil
	}

	tasks := tm.TaskBoard.List()
	if len(tasks) == 0 {
		return tool.NewTextResponse("No tasks on the board."), nil
	}

	return tool.NewJSONResponse(tasks), nil
}

func createTaskBoardTools() []tool.BaseTool {
	return []tool.BaseTool{
		&createBoardTaskTool{},
		&claimBoardTaskTool{},
		&completeBoardTaskTool{},
		&listBoardTasksTool{},
	}
}
