package agent

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

// HookAction represents the action a hook wants to take on an intercepted event.
type HookAction int

// HookAction values control whether an event is allowed, denied, or modified.
const (
	HookAllow HookAction = iota
	HookDeny
	HookModify
)

// ToolUseContext provides context about a tool invocation to pre-tool-use hooks.
type ToolUseContext struct {
	ToolCallID string
	ToolName   string
	Input      string
	AgentName  string
	TaskID     string
	Branch     string
}

// PreToolUseResult is the decision returned by a pre-tool-use hook.
type PreToolUseResult struct {
	Action     HookAction
	DenyReason string
	Input      string
}

// PostToolUseContext provides context about a completed tool invocation to post-tool-use hooks.
type PostToolUseContext struct {
	ToolUseContext
	Output   string
	IsError  bool
	Duration time.Duration
}

// PostToolUseResult is the decision returned by a post-tool-use hook.
type PostToolUseResult struct {
	Action HookAction
	Output string
}

// ModelCallContext provides context about an upcoming LLM call to pre-model-call hooks.
type ModelCallContext struct {
	Messages  []message.Message
	Tools     []tool.BaseTool
	AgentName string
	TaskID    string
	Branch    string
}

// ModelCallResult is the decision returned by a pre-model-call hook.
type ModelCallResult struct {
	Action   HookAction
	Messages []message.Message
	Tools    []tool.BaseTool
}

// ModelResponseContext provides context about a completed LLM response to post-model-call hooks.
type ModelResponseContext struct {
	Response  *llm.Response
	Duration  time.Duration
	AgentName string
	TaskID    string
	Branch    string
	Error     error
}

// ModelResponseResult is the decision returned by a post-model-call hook.
type ModelResponseResult struct {
	Action   HookAction
	Response *llm.Response
}

// SubagentEventContext provides context about a sub-agent lifecycle event.
type SubagentEventContext struct {
	TaskID    string
	AgentName string
	Task      string
	Branch    string
	Result    string
	Error     error
	Duration  time.Duration
}

// Hooks defines callback functions that intercept and optionally modify agent execution events.
type Hooks struct {
	PreToolUse      func(ctx context.Context, tc ToolUseContext) (PreToolUseResult, error)
	PostToolUse     func(ctx context.Context, tc PostToolUseContext) (PostToolUseResult, error)
	PreModelCall    func(ctx context.Context, mc ModelCallContext) (ModelCallResult, error)
	PostModelCall   func(ctx context.Context, mc ModelResponseContext) (ModelResponseResult, error)
	OnSubagentStart func(ctx context.Context, sc SubagentEventContext)
	OnSubagentStop  func(ctx context.Context, sc SubagentEventContext)
}

// HookEventType identifies the kind of hook event being emitted.
type HookEventType string

// HookEventType values for each stage of the agent execution pipeline.
const (
	HookEventPreToolUse    HookEventType = "pre_tool_use"
	HookEventPostToolUse   HookEventType = "post_tool_use"
	HookEventPreModelCall  HookEventType = "pre_model_call"
	HookEventPostModelCall HookEventType = "post_model_call"
	HookEventSubagentStart HookEventType = "subagent_start"
	HookEventSubagentStop  HookEventType = "subagent_stop"
)

// HookEvent is a structured record of an agent execution event emitted by observing hooks.
type HookEvent struct {
	Type       HookEventType
	Timestamp  time.Time
	AgentName  string
	TaskID     string
	Branch     string
	ToolCallID string
	ToolName   string
	Input      string
	Output     string
	IsError    bool
	Duration   time.Duration
	Usage      llm.TokenUsage
	Error      string
}

// NewObservingHooks creates a Hooks instance that emits read-only HookEvent records to fn.
func NewObservingHooks(fn func(HookEvent)) Hooks {
	return Hooks{
		PreToolUse: func(_ context.Context, tc ToolUseContext) (PreToolUseResult, error) {
			fn(HookEvent{
				Type:       HookEventPreToolUse,
				Timestamp:  time.Now(),
				AgentName:  tc.AgentName,
				TaskID:     tc.TaskID,
				Branch:     tc.Branch,
				ToolCallID: tc.ToolCallID,
				ToolName:   tc.ToolName,
				Input:      tc.Input,
			})
			return PreToolUseResult{Action: HookAllow}, nil
		},
		PostToolUse: func(_ context.Context, tc PostToolUseContext) (PostToolUseResult, error) {
			fn(HookEvent{
				Type:       HookEventPostToolUse,
				Timestamp:  time.Now(),
				AgentName:  tc.AgentName,
				TaskID:     tc.TaskID,
				Branch:     tc.Branch,
				ToolCallID: tc.ToolCallID,
				ToolName:   tc.ToolName,
				Input:      tc.Input,
				Output:     tc.Output,
				IsError:    tc.IsError,
				Duration:   tc.Duration,
			})
			return PostToolUseResult{Action: HookAllow}, nil
		},
		PreModelCall: func(_ context.Context, mc ModelCallContext) (ModelCallResult, error) {
			fn(HookEvent{
				Type:      HookEventPreModelCall,
				Timestamp: time.Now(),
				AgentName: mc.AgentName,
				TaskID:    mc.TaskID,
				Branch:    mc.Branch,
			})
			return ModelCallResult{Action: HookAllow}, nil
		},
		PostModelCall: func(_ context.Context, mc ModelResponseContext) (ModelResponseResult, error) {
			evt := HookEvent{
				Type:      HookEventPostModelCall,
				Timestamp: time.Now(),
				AgentName: mc.AgentName,
				TaskID:    mc.TaskID,
				Branch:    mc.Branch,
				Duration:  mc.Duration,
			}
			if mc.Error != nil {
				evt.IsError = true
				evt.Error = mc.Error.Error()
			} else if mc.Response != nil {
				evt.Usage = mc.Response.Usage
			}
			fn(evt)
			return ModelResponseResult{Action: HookAllow}, nil
		},
		OnSubagentStart: func(_ context.Context, sc SubagentEventContext) {
			fn(HookEvent{
				Type:      HookEventSubagentStart,
				Timestamp: time.Now(),
				AgentName: sc.AgentName,
				TaskID:    sc.TaskID,
				Branch:    sc.Branch,
				Input:     sc.Task,
			})
		},
		OnSubagentStop: func(_ context.Context, sc SubagentEventContext) {
			evt := HookEvent{
				Type:      HookEventSubagentStop,
				Timestamp: time.Now(),
				AgentName: sc.AgentName,
				TaskID:    sc.TaskID,
				Branch:    sc.Branch,
				Output:    sc.Result,
				Duration:  sc.Duration,
			}
			if sc.Error != nil {
				evt.IsError = true
				evt.Error = sc.Error.Error()
			}
			fn(evt)
		},
	}
}

// Task scope context helpers for correlating events across nested sub-agents.

type taskScopeKey struct{}

type taskScope struct {
	TaskID    string
	AgentName string
	Branch    string
}

func withTaskScope(
	ctx context.Context,
	taskID, agentName string,
) context.Context {
	var branch string
	if existing, ok := ctx.Value(taskScopeKey{}).(taskScope); ok {
		branch = existing.Branch + "/" + agentName
	} else {
		branch = agentName
	}
	return context.WithValue(ctx, taskScopeKey{}, taskScope{
		TaskID:    taskID,
		AgentName: agentName,
		Branch:    branch,
	})
}

func taskScopeFromContext(
	ctx context.Context,
) (taskID, agentName, branch string) {
	if s, ok := ctx.Value(taskScopeKey{}).(taskScope); ok {
		return s.TaskID, s.AgentName, s.Branch
	}
	return "", "", ""
}

// Chain runners for composing multiple hooks.

func runPreToolUse(
	ctx context.Context,
	hooks []Hooks,
	tc ToolUseContext,
) (PreToolUseResult, error) {
	result := PreToolUseResult{Action: HookAllow, Input: tc.Input}
	for _, h := range hooks {
		if h.PreToolUse == nil {
			continue
		}
		r, err := h.PreToolUse(ctx, tc)
		if err != nil {
			return PreToolUseResult{
				Action:     HookDeny,
				DenyReason: err.Error(),
			}, err
		}
		switch r.Action {
		case HookDeny:
			return r, nil
		case HookModify:
			result.Action = HookModify
			result.Input = r.Input
			tc.Input = r.Input
		}
	}
	return result, nil
}

func runPostToolUse(
	ctx context.Context,
	hooks []Hooks,
	tc PostToolUseContext,
) (PostToolUseResult, error) {
	result := PostToolUseResult{Action: HookAllow}
	for _, h := range hooks {
		if h.PostToolUse == nil {
			continue
		}
		r, err := h.PostToolUse(ctx, tc)
		if err != nil {
			return result, err
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Output = r.Output
			tc.Output = r.Output
		}
	}
	return result, nil
}

func runPreModelCall(
	ctx context.Context,
	hooks []Hooks,
	mc ModelCallContext,
) (ModelCallResult, error) {
	result := ModelCallResult{
		Action:   HookAllow,
		Messages: mc.Messages,
		Tools:    mc.Tools,
	}
	for _, h := range hooks {
		if h.PreModelCall == nil {
			continue
		}
		r, err := h.PreModelCall(ctx, mc)
		if err != nil {
			return result, err
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Messages = r.Messages
			result.Tools = r.Tools
			mc.Messages = r.Messages
			mc.Tools = r.Tools
		}
	}
	return result, nil
}

func runPostModelCall(
	ctx context.Context,
	hooks []Hooks,
	mc ModelResponseContext,
) (ModelResponseResult, error) {
	result := ModelResponseResult{Action: HookAllow, Response: mc.Response}
	for _, h := range hooks {
		if h.PostModelCall == nil {
			continue
		}
		r, err := h.PostModelCall(ctx, mc)
		if err != nil {
			return result, err
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Response = r.Response
			mc.Response = r.Response
		}
	}
	return result, nil
}

func runSubagentStart(
	ctx context.Context,
	hooks []Hooks,
	sc SubagentEventContext,
) {
	for _, h := range hooks {
		if h.OnSubagentStart != nil {
			h.OnSubagentStart(ctx, sc)
		}
	}
}

func runSubagentStop(
	ctx context.Context,
	hooks []Hooks,
	sc SubagentEventContext,
) {
	for _, h := range hooks {
		if h.OnSubagentStop != nil {
			h.OnSubagentStop(ctx, sc)
		}
	}
}
