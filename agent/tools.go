package agent

import (
	"context"
	"sync"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

func (a *Agent) executeSingleTool(
	ctx context.Context,
	registry *tool.Registry,
	tc message.ToolCall,
) ToolExecutionResult {
	resp, err := registry.Execute(ctx, tool.ToolCall{
		ID:    tc.ID,
		Name:  tc.Name,
		Input: tc.Input,
	})

	result := ToolExecutionResult{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Input:      tc.Input,
		IsError:    resp.IsError || err != nil,
	}

	if err != nil {
		result.Output = err.Error()
	} else {
		result.Output = resp.Content
	}

	return result
}

func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []message.ToolCall,
) []ToolExecutionResult {
	registry := tool.NewRegistry()
	for _, t := range a.getTools() {
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
