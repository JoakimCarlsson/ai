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

// ToolErrorContext provides context about a tool execution error to error-recovery hooks.
type ToolErrorContext struct {
	ToolUseContext
	Error    error
	Output   string
	Duration time.Duration
}

// ToolErrorResult is the decision returned by a tool-error hook.
type ToolErrorResult struct {
	Action HookAction
	Output string
}

// ModelErrorContext provides context about an LLM call failure to error-recovery hooks.
type ModelErrorContext struct {
	Messages  []message.Message
	Tools     []tool.BaseTool
	Error     error
	AgentName string
	TaskID    string
	Branch    string
}

// ModelErrorResult is the decision returned by a model-error hook.
type ModelErrorResult struct {
	Action   HookAction
	Response *llm.Response
}

// LifecycleContext provides context about an agent lifecycle event to before/after-agent hooks.
type LifecycleContext struct {
	AgentName string
	TaskID    string
	Branch    string
	Input     string
	Response  *ChatResponse
}

// LifecycleResult is the decision returned by a before/after-agent hook.
type LifecycleResult struct {
	Action   HookAction
	Response *ChatResponse
}

// RunContext provides context about a run lifecycle event to before/after-run hooks.
type RunContext struct {
	AgentName string
	TaskID    string
	Branch    string
	Input     string
	Response  *ChatResponse
	Error     error
	Duration  time.Duration
}

// UserMessageContext provides context about an incoming user message to input hooks.
type UserMessageContext struct {
	Message   string
	AgentName string
	TaskID    string
	Branch    string
}

// UserMessageResult is the decision returned by a user-message hook.
type UserMessageResult struct {
	Action     HookAction
	Message    string
	DenyReason string
}

// Hooks defines callback functions that intercept and optionally modify agent execution events.
type Hooks struct {
	PreToolUse      func(ctx context.Context, tc ToolUseContext) (PreToolUseResult, error)
	PostToolUse     func(ctx context.Context, tc PostToolUseContext) (PostToolUseResult, error)
	PreModelCall    func(ctx context.Context, mc ModelCallContext) (ModelCallResult, error)
	PostModelCall   func(ctx context.Context, mc ModelResponseContext) (ModelResponseResult, error)
	OnSubagentStart func(ctx context.Context, sc SubagentEventContext)
	OnSubagentStop  func(ctx context.Context, sc SubagentEventContext)
	OnToolError     func(ctx context.Context, tc ToolErrorContext) (ToolErrorResult, error)
	OnModelError    func(ctx context.Context, mc ModelErrorContext) (ModelErrorResult, error)
	BeforeAgent     func(ctx context.Context, ac LifecycleContext) (LifecycleResult, error)
	AfterAgent      func(ctx context.Context, ac LifecycleContext) (LifecycleResult, error)
	BeforeRun       func(ctx context.Context, rc RunContext)
	AfterRun        func(ctx context.Context, rc RunContext)
	OnUserMessage   func(ctx context.Context, uc UserMessageContext) (UserMessageResult, error)
	OnEvent         func(ctx context.Context, evt HookEvent)
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
	HookEventToolError     HookEventType = "tool_error"
	HookEventModelError    HookEventType = "model_error"
	HookEventBeforeAgent   HookEventType = "before_agent"
	HookEventAfterAgent    HookEventType = "after_agent"
	HookEventBeforeRun     HookEventType = "before_run"
	HookEventAfterRun      HookEventType = "after_run"
	HookEventUserMessage   HookEventType = "user_message"
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
		OnToolError: func(_ context.Context, tc ToolErrorContext) (ToolErrorResult, error) {
			evt := HookEvent{
				Type:       HookEventToolError,
				Timestamp:  time.Now(),
				AgentName:  tc.AgentName,
				TaskID:     tc.TaskID,
				Branch:     tc.Branch,
				ToolCallID: tc.ToolCallID,
				ToolName:   tc.ToolName,
				Input:      tc.Input,
				Output:     tc.Output,
				IsError:    true,
				Duration:   tc.Duration,
			}
			if tc.Error != nil {
				evt.Error = tc.Error.Error()
			}
			fn(evt)
			return ToolErrorResult{Action: HookAllow}, nil
		},
		OnModelError: func(_ context.Context, mc ModelErrorContext) (ModelErrorResult, error) {
			fn(HookEvent{
				Type:      HookEventModelError,
				Timestamp: time.Now(),
				AgentName: mc.AgentName,
				TaskID:    mc.TaskID,
				Branch:    mc.Branch,
				IsError:   true,
				Error:     mc.Error.Error(),
			})
			return ModelErrorResult{Action: HookAllow}, nil
		},
		BeforeAgent: func(_ context.Context, ac LifecycleContext) (LifecycleResult, error) {
			fn(HookEvent{
				Type:      HookEventBeforeAgent,
				Timestamp: time.Now(),
				AgentName: ac.AgentName,
				TaskID:    ac.TaskID,
				Branch:    ac.Branch,
				Input:     ac.Input,
			})
			return LifecycleResult{Action: HookAllow}, nil
		},
		AfterAgent: func(_ context.Context, ac LifecycleContext) (LifecycleResult, error) {
			fn(HookEvent{
				Type:      HookEventAfterAgent,
				Timestamp: time.Now(),
				AgentName: ac.AgentName,
				TaskID:    ac.TaskID,
				Branch:    ac.Branch,
			})
			return LifecycleResult{Action: HookAllow}, nil
		},
		BeforeRun: func(_ context.Context, rc RunContext) {
			fn(HookEvent{
				Type:      HookEventBeforeRun,
				Timestamp: time.Now(),
				AgentName: rc.AgentName,
				TaskID:    rc.TaskID,
				Branch:    rc.Branch,
				Input:     rc.Input,
			})
		},
		AfterRun: func(_ context.Context, rc RunContext) {
			evt := HookEvent{
				Type:      HookEventAfterRun,
				Timestamp: time.Now(),
				AgentName: rc.AgentName,
				TaskID:    rc.TaskID,
				Branch:    rc.Branch,
				Duration:  rc.Duration,
			}
			if rc.Error != nil {
				evt.IsError = true
				evt.Error = rc.Error.Error()
			}
			fn(evt)
		},
		OnUserMessage: func(_ context.Context, uc UserMessageContext) (UserMessageResult, error) {
			fn(HookEvent{
				Type:      HookEventUserMessage,
				Timestamp: time.Now(),
				AgentName: uc.AgentName,
				TaskID:    uc.TaskID,
				Branch:    uc.Branch,
				Input:     uc.Message,
			})
			return UserMessageResult{
				Action:  HookAllow,
				Message: uc.Message,
			}, nil
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
	runOnEvent(ctx, hooks, HookEvent{
		Type:       HookEventPreToolUse,
		Timestamp:  time.Now(),
		AgentName:  tc.AgentName,
		TaskID:     tc.TaskID,
		Branch:     tc.Branch,
		ToolCallID: tc.ToolCallID,
		ToolName:   tc.ToolName,
		Input:      tc.Input,
	})
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
	runOnEvent(ctx, hooks, HookEvent{
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
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventPreModelCall,
		Timestamp: time.Now(),
		AgentName: mc.AgentName,
		TaskID:    mc.TaskID,
		Branch:    mc.Branch,
	})
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
	runOnEvent(ctx, hooks, evt)
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
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventSubagentStart,
		Timestamp: time.Now(),
		AgentName: sc.AgentName,
		TaskID:    sc.TaskID,
		Branch:    sc.Branch,
		Input:     sc.Task,
	})
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
	runOnEvent(ctx, hooks, evt)
}

func runOnToolError(
	ctx context.Context,
	hooks []Hooks,
	tc ToolErrorContext,
) (ToolErrorResult, error) {
	result := ToolErrorResult{Action: HookAllow}
	recovered := false
	for _, h := range hooks {
		if h.OnToolError == nil {
			continue
		}
		r, err := h.OnToolError(ctx, tc)
		if err != nil {
			return ToolErrorResult{Action: HookAllow}, err
		}
		if r.Action == HookModify && !recovered {
			result.Action = HookModify
			result.Output = r.Output
			recovered = true
		}
	}
	evt := HookEvent{
		Type:       HookEventToolError,
		Timestamp:  time.Now(),
		AgentName:  tc.AgentName,
		TaskID:     tc.TaskID,
		Branch:     tc.Branch,
		ToolCallID: tc.ToolCallID,
		ToolName:   tc.ToolName,
		Input:      tc.Input,
		Output:     tc.Output,
		IsError:    true,
		Duration:   tc.Duration,
	}
	if tc.Error != nil {
		evt.Error = tc.Error.Error()
	}
	runOnEvent(ctx, hooks, evt)
	return result, nil
}

func runOnModelError(
	ctx context.Context,
	hooks []Hooks,
	mc ModelErrorContext,
) (ModelErrorResult, error) {
	result := ModelErrorResult{Action: HookAllow}
	recovered := false
	for _, h := range hooks {
		if h.OnModelError == nil {
			continue
		}
		r, err := h.OnModelError(ctx, mc)
		if err != nil {
			return ModelErrorResult{Action: HookAllow}, err
		}
		if r.Action == HookModify && !recovered {
			result.Action = HookModify
			result.Response = r.Response
			recovered = true
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventModelError,
		Timestamp: time.Now(),
		AgentName: mc.AgentName,
		TaskID:    mc.TaskID,
		Branch:    mc.Branch,
		IsError:   true,
		Error:     mc.Error.Error(),
	})
	return result, nil
}

func runBeforeAgent(
	ctx context.Context,
	hooks []Hooks,
	ac LifecycleContext,
) (LifecycleResult, error) {
	result := LifecycleResult{Action: HookAllow}
	for _, h := range hooks {
		if h.BeforeAgent == nil {
			continue
		}
		r, err := h.BeforeAgent(ctx, ac)
		if err != nil {
			return LifecycleResult{Action: HookDeny}, err
		}
		if r.Action == HookDeny {
			return r, nil
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Response = r.Response
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventBeforeAgent,
		Timestamp: time.Now(),
		AgentName: ac.AgentName,
		TaskID:    ac.TaskID,
		Branch:    ac.Branch,
		Input:     ac.Input,
	})
	return result, nil
}

func runAfterAgent(
	ctx context.Context,
	hooks []Hooks,
	ac LifecycleContext,
) (LifecycleResult, error) {
	result := LifecycleResult{Action: HookAllow}
	for _, h := range hooks {
		if h.AfterAgent == nil {
			continue
		}
		r, err := h.AfterAgent(ctx, ac)
		if err != nil {
			return result, err
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Response = r.Response
			ac.Response = r.Response
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventAfterAgent,
		Timestamp: time.Now(),
		AgentName: ac.AgentName,
		TaskID:    ac.TaskID,
		Branch:    ac.Branch,
	})
	return result, nil
}

func runBeforeRun(
	ctx context.Context,
	hooks []Hooks,
	rc RunContext,
) {
	for _, h := range hooks {
		if h.BeforeRun != nil {
			h.BeforeRun(ctx, rc)
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventBeforeRun,
		Timestamp: time.Now(),
		AgentName: rc.AgentName,
		TaskID:    rc.TaskID,
		Branch:    rc.Branch,
		Input:     rc.Input,
	})
}

func runAfterRun(
	ctx context.Context,
	hooks []Hooks,
	rc RunContext,
) {
	for _, h := range hooks {
		if h.AfterRun != nil {
			h.AfterRun(ctx, rc)
		}
	}
	evt := HookEvent{
		Type:      HookEventAfterRun,
		Timestamp: time.Now(),
		AgentName: rc.AgentName,
		TaskID:    rc.TaskID,
		Branch:    rc.Branch,
		Duration:  rc.Duration,
	}
	if rc.Error != nil {
		evt.IsError = true
		evt.Error = rc.Error.Error()
	}
	runOnEvent(ctx, hooks, evt)
}

func runOnUserMessage(
	ctx context.Context,
	hooks []Hooks,
	uc UserMessageContext,
) (UserMessageResult, error) {
	result := UserMessageResult{Action: HookAllow, Message: uc.Message}
	for _, h := range hooks {
		if h.OnUserMessage == nil {
			continue
		}
		r, err := h.OnUserMessage(ctx, uc)
		if err != nil {
			return UserMessageResult{
				Action:     HookDeny,
				DenyReason: err.Error(),
			}, err
		}
		if r.Action == HookDeny {
			return r, nil
		}
		if r.Action == HookModify {
			result.Action = HookModify
			result.Message = r.Message
			uc.Message = r.Message
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventUserMessage,
		Timestamp: time.Now(),
		AgentName: uc.AgentName,
		TaskID:    uc.TaskID,
		Branch:    uc.Branch,
		Input:     uc.Message,
	})
	return result, nil
}

func runOnEvent(
	ctx context.Context,
	hooks []Hooks,
	evt HookEvent,
) {
	for _, h := range hooks {
		if h.OnEvent != nil {
			h.OnEvent(ctx, evt)
		}
	}
}
