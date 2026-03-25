package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/joakimcarlsson/ai/tool"
)

// FanOutConfig configures a fan-out tool that distributes multiple tasks to
// parallel instances of the same agent. The parent agent invokes this tool with
// a list of tasks, and each task is sent to a separate Chat() call concurrently.
type FanOutConfig struct {
	// Name is the tool name the parent agent uses to invoke this fan-out.
	Name string
	// Description is the tool description shown to the LLM.
	Description string
	// Agent is the agent instance that handles each individual task.
	Agent *Agent
	// MaxConcurrency limits parallel task execution. 0 means unlimited.
	MaxConcurrency int
}

type fanOutInput struct {
	Tasks []string `json:"tasks" desc:"List of tasks to execute in parallel. Each task is sent to a separate agent instance."`
}

type fanOutResult struct {
	Task    string `json:"task"`
	Result  string `json:"result"`
	IsError bool   `json:"is_error,omitempty"`
}

type fanOutTool struct {
	config FanOutConfig
}

func newFanOutTool(config FanOutConfig) *fanOutTool {
	return &fanOutTool{config: config}
}

func (t *fanOutTool) Info() tool.Info {
	return tool.NewInfo(t.config.Name, t.config.Description, fanOutInput{})
}

func (t *fanOutTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input fanOutInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid fan-out parameters: " + err.Error(),
		), nil
	}

	if len(input.Tasks) == 0 {
		return tool.NewTextErrorResponse("at least one task is required"), nil
	}

	results := make([]fanOutResult, len(input.Tasks))
	var wg sync.WaitGroup
	var sem chan struct{}

	if t.config.MaxConcurrency > 0 {
		sem = make(chan struct{}, t.config.MaxConcurrency)
	}

	for i, task := range input.Tasks {
		wg.Add(1)
		go func(idx int, taskStr string) {
			defer wg.Done()

			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			resp, err := t.config.Agent.Chat(ctx, taskStr)
			if err != nil {
				results[idx] = fanOutResult{
					Task:    taskStr,
					Result:  fmt.Sprintf("error: %s", err.Error()),
					IsError: true,
				}
				return
			}

			results[idx] = fanOutResult{
				Task:   taskStr,
				Result: resp.Content,
			}
		}(i, task)
	}

	wg.Wait()

	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "## Task %d: %s\n", i+1, r.Task)
		if r.IsError {
			fmt.Fprintf(&sb, "**Error:** %s\n\n", r.Result)
		} else {
			sb.WriteString(r.Result)
			sb.WriteString("\n\n")
		}
	}

	return tool.NewTextResponse(sb.String()), nil
}
