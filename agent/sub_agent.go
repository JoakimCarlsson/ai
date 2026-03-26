package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/tool"
)

// SubAgentConfig configures a sub-agent tool that the parent agent can delegate
// tasks to. Unlike handoffs, sub-agents run independently and return their result
// to the parent. They can run synchronously or in the background.
type SubAgentConfig struct {
	// Name is the tool name the parent agent uses to invoke this sub-agent.
	Name string
	// Description is the tool description shown to the LLM.
	Description string
	// Agent is the sub-agent instance that handles delegated tasks.
	Agent *Agent
}

type subAgentInput struct {
	Task       string `json:"task"       desc:"The task or question to send to the sub-agent"`
	Background bool   `json:"background" desc:"If true, run in background and return a task ID immediately"       required:"false"`
	MaxTurns   int    `json:"max_turns"  desc:"Maximum number of tool-execution turns. 0 uses the agent default." required:"false"`
}

type subAgentTool struct {
	config SubAgentConfig
}

func newSubAgentTool(config SubAgentConfig) *subAgentTool {
	return &subAgentTool{config: config}
}

func (t *subAgentTool) Info() tool.Info {
	return tool.NewInfo(
		t.config.Name,
		t.config.Description,
		subAgentInput{},
	)
}

func (t *subAgentTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input subAgentInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid sub-agent parameters: " + err.Error(),
		), nil
	}

	if input.Task == "" {
		return tool.NewTextErrorResponse("task is required"), nil
	}

	var opts []ChatOption
	if input.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(input.MaxTurns))
	}

	if input.Background {
		tm := taskManagerFromContext(ctx)
		if tm == nil {
			return t.runSync(ctx, input.Task, opts...)
		}
		taskID := tm.Launch(
			ctx,
			t.config.Name,
			t.config.Agent,
			input.Task,
			opts...)

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

	return t.runSync(ctx, input.Task, opts...)
}

func (t *subAgentTool) runSync(
	ctx context.Context,
	task string,
	opts ...ChatOption,
) (tool.Response, error) {
	resp, err := t.config.Agent.Chat(ctx, task, opts...)
	if err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("sub-agent %q failed: %s", t.config.Name, err.Error()),
		), nil
	}
	return tool.NewTextResponse(resp.Content), nil
}
