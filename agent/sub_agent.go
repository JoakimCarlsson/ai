package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/tool"
)

type SubAgentConfig struct {
	Name        string
	Description string
	Agent       *Agent
}

type subAgentInput struct {
	Task       string `json:"task" desc:"The task or question to send to the sub-agent"`
	Background bool   `json:"background" desc:"If true, run in background and return a task ID immediately" required:"false"`
}

type subAgentTool struct {
	config SubAgentConfig
}

func newSubAgentTool(config SubAgentConfig) *subAgentTool {
	return &subAgentTool{config: config}
}

func (t *subAgentTool) Info() tool.ToolInfo {
	return tool.NewToolInfo(t.config.Name, t.config.Description, subAgentInput{})
}

func (t *subAgentTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
	var input subAgentInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse("invalid sub-agent parameters: " + err.Error()), nil
	}

	if input.Task == "" {
		return tool.NewTextErrorResponse("task is required"), nil
	}

	if input.Background {
		tm := taskManagerFromContext(ctx)
		if tm == nil {
			return t.runSync(ctx, input.Task)
		}
		taskID := tm.Launch(ctx, t.config.Name, t.config.Agent, input.Task)

		type launchOutput struct {
			TaskID    string `json:"task_id"`
			AgentName string `json:"agent_name"`
			Status    string `json:"status"`
		}
		return tool.NewJSONResponse(launchOutput{
			TaskID:    taskID,
			AgentName: t.config.Name,
			Status:    "launched",
		}), nil
	}

	return t.runSync(ctx, input.Task)
}

func (t *subAgentTool) runSync(ctx context.Context, task string) (tool.ToolResponse, error) {
	resp, err := t.config.Agent.Chat(ctx, task)
	if err != nil {
		return tool.NewTextErrorResponse(fmt.Sprintf("sub-agent %q failed: %s", t.config.Name, err.Error())), nil
	}
	return tool.NewTextResponse(resp.Content), nil
}
