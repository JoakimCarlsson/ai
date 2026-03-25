# Hooks

Hooks let you observe, modify, or block agent behavior at key points in the execution pipeline. They cover tool calls, model interactions, and sub-agent lifecycle events.

## Setup

```go
myAgent := agent.New(llmClient,
    agent.WithHooks(agent.Hooks{
        PreToolUse: func(ctx context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
            log.Printf("Tool call: %s (branch: %s)", tc.ToolName, tc.Branch)
            return agent.PreToolUseResult{Action: agent.HookAllow}, nil
        },
    }),
)
```

## Hook Types

| Hook | Fires | Can |
|------|-------|-----|
| `PreToolUse` | Before a tool executes | Allow, Deny, or Modify input |
| `PostToolUse` | After a tool executes | Allow or Modify output |
| `PreModelCall` | Before an LLM request | Allow or Modify messages/tools |
| `PostModelCall` | After an LLM response | Allow or Modify response |
| `OnSubagentStart` | When a background sub-agent launches | Observe only |
| `OnSubagentStop` | When a background sub-agent finishes | Observe only |

## HookAction

Every hook returns a `HookAction` that controls what happens next:

| Action | Behavior |
|--------|----------|
| `HookAllow` | Continue normally (default) |
| `HookDeny` | Block execution (PreToolUse only) |
| `HookModify` | Replace input, output, messages, or response |

## Denying a Tool Call

Return `HookDeny` from `PreToolUse` to block a tool before it runs:

```go
agent.Hooks{
    PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
        if tc.ToolName == "dangerous_tool" {
            return agent.PreToolUseResult{
                Action:     agent.HookDeny,
                DenyReason: "this tool is not allowed",
            }, nil
        }
        return agent.PreToolUseResult{Action: agent.HookAllow}, nil
    },
}
```

The agent receives a tool error result with the deny reason.

## Modifying Tool Input

Return `HookModify` from `PreToolUse` to rewrite the input before execution:

```go
agent.Hooks{
    PreToolUse: func(_ context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
        modified := strings.ReplaceAll(tc.Input, "SECRET", "[REDACTED]")
        return agent.PreToolUseResult{
            Action: agent.HookModify,
            Input:  modified,
        }, nil
    },
}
```

## Modifying Model Messages

Return `HookModify` from `PreModelCall` to inject or filter messages before they reach the LLM:

```go
agent.Hooks{
    PreModelCall: func(_ context.Context, mc agent.ModelCallContext) (agent.ModelCallResult, error) {
        extra := message.NewUserMessage("Remember: always respond in JSON.")
        return agent.ModelCallResult{
            Action:   agent.HookModify,
            Messages: append(mc.Messages, extra),
            Tools:    mc.Tools,
        }, nil
    },
}
```

## Chaining Multiple Hooks

Pass multiple `Hooks` to `WithHooks`, or call `WithHooks` multiple times. Hooks run in registration order.

```go
myAgent := agent.New(llmClient,
    agent.WithHooks(loggingHooks, guardRailHooks, metricsHooks),
)
```

Chain rules:

- **Deny wins immediately** — if any hook returns `HookDeny`, later hooks are skipped
- **Last Modify wins** — if multiple hooks return `HookModify`, the last one's value is used
- **nil fields are skipped** — you only need to set the hooks you care about

## Observation with NewObservingHooks

For pure observation (logging, metrics, streaming to a UI), use the `NewObservingHooks` helper. It wires all 6 hooks to emit structured `HookEvent` values to a single callback:

```go
myAgent := agent.New(llmClient,
    agent.WithHooks(agent.NewObservingHooks(func(evt agent.HookEvent) {
        log.Printf("[%s] agent=%s branch=%s tool=%s",
            evt.Type, evt.AgentName, evt.Branch, evt.ToolName)
    })),
)
```

All observing hooks return `HookAllow` — they never block or modify execution.

### HookEvent

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `HookEventType` | Event type (see below) |
| `Timestamp` | `time.Time` | When the event fired |
| `AgentName` | `string` | Name of the agent |
| `TaskID` | `string` | Background task ID (if applicable) |
| `Branch` | `string` | Agent hierarchy path (e.g. `"orchestrator/researcher"`) |
| `ToolCallID` | `string` | Tool call ID (tool events only) |
| `ToolName` | `string` | Tool name (tool events only) |
| `Input` | `string` | Tool input or sub-agent task |
| `Output` | `string` | Tool output or sub-agent result |
| `IsError` | `bool` | Whether an error occurred |
| `Duration` | `time.Duration` | Execution duration (post-events only) |
| `Usage` | `llm.TokenUsage` | Token usage (post model call only) |
| `Error` | `string` | Error message (if `IsError` is true) |

### Event Types

| Constant | Value | When |
|----------|-------|------|
| `HookEventPreToolUse` | `"pre_tool_use"` | Before tool execution |
| `HookEventPostToolUse` | `"post_tool_use"` | After tool execution |
| `HookEventPreModelCall` | `"pre_model_call"` | Before LLM request |
| `HookEventPostModelCall` | `"post_model_call"` | After LLM response |
| `HookEventSubagentStart` | `"subagent_start"` | Background sub-agent launched |
| `HookEventSubagentStop` | `"subagent_stop"` | Background sub-agent finished |

## Branch

The `Branch` field on all hook contexts gives you the agent hierarchy as a `/`-separated path. For a nested setup where an orchestrator delegates to a researcher which delegates to a scraper:

```
Branch: "orchestrator/researcher/scraper"
```

This lets you immediately see which agent in the hierarchy produced an event, without cross-referencing task IDs.

## Hook Propagation

Hooks set on a parent agent automatically propagate to sub-agents that don't have their own hooks:

```go
orchestrator := agent.New(llmClient,
    agent.WithHooks(myHooks),
    agent.WithSubAgents(
        agent.SubAgentConfig{Name: "worker", Agent: worker},
    ),
)
// worker inherits myHooks since it has none of its own
```

If a sub-agent already has hooks configured, the parent's hooks are not applied.

## Context Structs

### ToolUseContext

Passed to `PreToolUse` and embedded in `PostToolUseContext`:

```go
type ToolUseContext struct {
    ToolCallID string
    ToolName   string
    Input      string
    AgentName  string
    TaskID     string
    Branch     string
}
```

### PostToolUseContext

Passed to `PostToolUse`:

```go
type PostToolUseContext struct {
    ToolUseContext        // Embeds all fields from ToolUseContext
    Output   string
    IsError  bool
    Duration time.Duration
}
```

### ModelCallContext

Passed to `PreModelCall`:

```go
type ModelCallContext struct {
    Messages  []message.Message
    Tools     []tool.BaseTool
    AgentName string
    TaskID    string
    Branch    string
}
```

### ModelResponseContext

Passed to `PostModelCall`:

```go
type ModelResponseContext struct {
    Response  *llm.Response
    Duration  time.Duration
    AgentName string
    TaskID    string
    Branch    string
    Error     error
}
```

### SubagentEventContext

Passed to `OnSubagentStart` and `OnSubagentStop`:

```go
type SubagentEventContext struct {
    TaskID    string
    AgentName string
    Task      string
    Branch    string
    Result    string
    Error     error
    Duration  time.Duration
}
```

## Streaming to a UI

A common use case is forwarding hook events to a frontend over WebSocket or SSE:

```go
agent.NewObservingHooks(func(evt agent.HookEvent) {
    data, _ := json.Marshal(evt)
    websocket.Send(data)
})
```

This gives the UI real-time visibility into tool calls, model interactions, and sub-agent lifecycle — including nested agent hierarchies via `Branch`.
