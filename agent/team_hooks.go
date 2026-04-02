package agent

import (
	"context"
	"time"
)

func runOnTeammateJoin(
	ctx context.Context,
	hooks []Hooks,
	tc TeammateEventContext,
) {
	for _, h := range hooks {
		if h.OnTeammateJoin != nil {
			h.OnTeammateJoin(ctx, tc)
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventTeammateJoin,
		Timestamp: time.Now(),
		AgentName: tc.MemberName,
		Input:     tc.Task,
	})
}

func runOnTeammateLeave(
	ctx context.Context,
	hooks []Hooks,
	tc TeammateEventContext,
) {
	for _, h := range hooks {
		if h.OnTeammateLeave != nil {
			h.OnTeammateLeave(ctx, tc)
		}
	}
	evt := HookEvent{
		Type:      HookEventTeammateLeave,
		Timestamp: time.Now(),
		AgentName: tc.MemberName,
		Duration:  tc.Duration,
	}
	if tc.Error != nil {
		evt.IsError = true
		evt.Error = tc.Error.Error()
	}
	runOnEvent(ctx, hooks, evt)
}

func runOnTeamMessage(
	ctx context.Context,
	hooks []Hooks,
	mc TeamMessageContext,
) {
	for _, h := range hooks {
		if h.OnTeamMessage != nil {
			h.OnTeamMessage(ctx, mc)
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventTeamMessage,
		Timestamp: time.Now(),
		AgentName: mc.Message.From,
		Input:     mc.Message.Content,
	})
}

func runOnTeammateComplete(
	ctx context.Context,
	hooks []Hooks,
	tc TeammateEventContext,
) {
	for _, h := range hooks {
		if h.OnTeammateComplete != nil {
			h.OnTeammateComplete(ctx, tc)
		}
	}
	runOnEvent(ctx, hooks, HookEvent{
		Type:      HookEventTeammateComplete,
		Timestamp: time.Now(),
		AgentName: tc.MemberName,
		Output:    tc.Result,
		Duration:  tc.Duration,
	})
}

func runOnTeammateError(
	ctx context.Context,
	hooks []Hooks,
	tc TeammateEventContext,
) {
	for _, h := range hooks {
		if h.OnTeammateError != nil {
			h.OnTeammateError(ctx, tc)
		}
	}
	evt := HookEvent{
		Type:      HookEventTeammateError,
		Timestamp: time.Now(),
		AgentName: tc.MemberName,
		IsError:   true,
		Duration:  tc.Duration,
	}
	if tc.Error != nil {
		evt.Error = tc.Error.Error()
	}
	runOnEvent(ctx, hooks, evt)
}
