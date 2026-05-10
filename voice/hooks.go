package voice

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// HookAction tells a hook callback's caller whether to proceed, abort, or
// adopt the modified values returned by the hook.
type HookAction int

const (
	// HookAllow keeps the original values; the operation proceeds.
	HookAllow HookAction = iota
	// HookDeny aborts the operation. The caller substitutes a deny outcome
	// (skip the tool, drop the user message, etc.) and continues.
	HookDeny
	// HookModify replaces the in-flight values with the result fields.
	HookModify
)

// ConversationLifecycleContext is passed to OnConversationStart and
// OnConversationEnd. ConversationID matches Conversation.ID().
type ConversationLifecycleContext struct {
	ConversationID string
}

// UserMessageContext is passed to OnUserMessage when STT commits a final
// user transcript and before the runner appends it to history.
type UserMessageContext struct {
	ConversationID string
	Text           string
}

// UserMessageResult is returned by OnUserMessage. On HookModify the runner
// uses Text instead of the original transcript. On HookDeny the runner
// drops the user turn entirely; DenyReason is surfaced via EventError.
type UserMessageResult struct {
	Action     HookAction
	Text       string
	DenyReason string
}

// ModelCallContext is passed to PreModelCall right before the runner calls
// the LLM. Messages reflects the strategy-trimmed history.
type ModelCallContext struct {
	ConversationID string
	Messages       []message.Message
	Tools          []tool.BaseTool
}

// ModelCallResult is returned by PreModelCall. On HookModify the runner
// substitutes Messages and Tools for the upcoming LLM call.
type ModelCallResult struct {
	Action   HookAction
	Messages []message.Message
	Tools    []tool.BaseTool
}

// ModelResponseContext is passed to PostModelCall after the LLM stream
// closes. Response is nil if the call ended with an error.
type ModelResponseContext struct {
	ConversationID string
	Response       *llm.Response
	Duration       time.Duration
	Error          error
}

// ToolUseContext is passed to PreToolUse before a tool runs.
type ToolUseContext struct {
	ConversationID string
	ToolCallID     string
	ToolName       string
	Input          string
}

// PreToolUseResult is returned by PreToolUse. On HookModify the runner
// invokes the tool with Input. On HookDeny the runner skips the tool and
// records DenyReason as the tool result (IsError = true).
type PreToolUseResult struct {
	Action     HookAction
	Input      string
	DenyReason string
}

// PostToolUseContext is passed to PostToolUse after a tool returns.
type PostToolUseContext struct {
	ToolUseContext
	Output   string
	IsError  bool
	Duration time.Duration
}

// PostToolUseResult is returned by PostToolUse. On HookModify the runner
// substitutes Output before the tool result lands in history.
type PostToolUseResult struct {
	Action HookAction
	Output string
}

// ToolErrorContext is passed to OnToolError when a tool run errors.
type ToolErrorContext struct {
	ToolUseContext
	Error    error
	Duration time.Duration
}

// ToolErrorResult is returned by OnToolError. On HookModify the runner
// replaces the failed output with Output and clears the error flag, so the
// LLM sees a successful tool result.
type ToolErrorResult struct {
	Action HookAction
	Output string
}

// Hooks defines optional callbacks invoked at synchronous interception
// points during a conversation. Any field may be nil. Multiple Hooks
// passed via WithHooks run in registration order; HookModify mutations
// chain (later hooks see earlier hooks' edits).
type Hooks struct {
	// OnConversationStart fires once when the runner begins. Observe-only.
	OnConversationStart func(ctx context.Context, c ConversationLifecycleContext)
	// OnConversationEnd fires once when the runner returns. Observe-only.
	OnConversationEnd func(ctx context.Context, c ConversationLifecycleContext)

	// OnUserMessage fires after STT commits a final transcript, before the
	// text is appended to history. May allow, modify, or deny.
	OnUserMessage func(ctx context.Context, uc UserMessageContext) (UserMessageResult, error)

	// PreModelCall fires before each LLM call (after context-window
	// strategy runs). May allow or modify Messages and Tools.
	PreModelCall func(ctx context.Context, mc ModelCallContext) (ModelCallResult, error)

	// PostModelCall fires after each LLM call returns or errors.
	// Observe-only.
	PostModelCall func(ctx context.Context, mc ModelResponseContext)

	// PreToolUse fires before each tool invocation. May allow, modify, or
	// deny.
	PreToolUse func(ctx context.Context, tc ToolUseContext) (PreToolUseResult, error)

	// PostToolUse fires after each successful tool invocation. May allow or
	// modify the tool output.
	PostToolUse func(ctx context.Context, tc PostToolUseContext) (PostToolUseResult, error)

	// OnToolError fires when a tool run errors. May allow (propagate) or
	// modify (recover with a fallback output).
	OnToolError func(ctx context.Context, tc ToolErrorContext) (ToolErrorResult, error)
}

type convIDKey struct{}

func withConversationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, convIDKey{}, id)
}

func conversationIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(convIDKey{}).(string); ok {
		return v
	}
	return ""
}

func runOnConversationStart(
	ctx context.Context,
	hooks []Hooks,
	c ConversationLifecycleContext,
) {
	for _, h := range hooks {
		if h.OnConversationStart != nil {
			h.OnConversationStart(ctx, c)
		}
	}
}

func runOnConversationEnd(
	ctx context.Context,
	hooks []Hooks,
	c ConversationLifecycleContext,
) {
	for _, h := range hooks {
		if h.OnConversationEnd != nil {
			h.OnConversationEnd(ctx, c)
		}
	}
}

func runOnUserMessage(
	ctx context.Context,
	hooks []Hooks,
	uc UserMessageContext,
) (UserMessageResult, error) {
	result := UserMessageResult{Action: HookAllow, Text: uc.Text}
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
		switch r.Action {
		case HookDeny:
			return r, nil
		case HookModify:
			result.Action = HookModify
			result.Text = r.Text
			uc.Text = r.Text
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
) {
	for _, h := range hooks {
		if h.PostModelCall != nil {
			h.PostModelCall(ctx, mc)
		}
	}
}

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
	result := PostToolUseResult{Action: HookAllow, Output: tc.Output}
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
			return result, err
		}
		if r.Action == HookModify && !recovered {
			result.Action = HookModify
			result.Output = r.Output
			recovered = true
		}
	}
	return result, nil
}
