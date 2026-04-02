# Hooks

Hooks let you observe, modify, or block agent behavior at key points in the execution pipeline. They cover tool calls, model interactions, error recovery, agent lifecycle, input validation, and cross-cutting event observation.

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
| `OnToolError` | When a tool returns an error | Allow (re-raise) or Modify (recover) |
| `OnModelError` | When an LLM call fails | Allow (re-raise) or Modify (recover) |
| `BeforeAgent` | Before an agent starts its run | Allow, Deny, or Modify (short-circuit) |
| `AfterAgent` | After an agent completes its run | Allow or Modify response |
| `BeforeRun` | At the start of Chat/ChatStream | Observe only |
| `AfterRun` | At the end of Chat/ChatStream | Observe only |
| `OnUserMessage` | When a user message arrives | Allow, Deny, or Modify message |
| `OnEvent` | On every hook event emitted | Observe only |
| `OnTeammateJoin` | When a teammate is spawned | Observe only |
| `OnTeammateLeave` | When a teammate leaves (stopped) | Observe only |
| `OnTeammateComplete` | When a teammate finishes successfully | Observe only |
| `OnTeammateError` | When a teammate encounters an error | Observe only |
| `OnTeamMessage` | When a message is sent between members | Observe only |

## HookAction

Every hook returns a `HookAction` that controls what happens next:

| Action | Behavior |
|--------|----------|
| `HookAllow` | Continue normally (default) |
| `HookDeny` | Block execution (PreToolUse, BeforeAgent, OnUserMessage) |
| `HookModify` | Replace input, output, messages, response, or recover from errors |

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

## Error Recovery

### Tool Error Recovery

`OnToolError` fires when a tool returns an error, before the error reaches `PostToolUse`. Return `HookModify` with replacement output to recover:

```go
agent.Hooks{
    OnToolError: func(_ context.Context, tc agent.ToolErrorContext) (agent.ToolErrorResult, error) {
        if tc.ToolName == "flaky_api" {
            return agent.ToolErrorResult{
                Action: agent.HookModify,
                Output: "API temporarily unavailable, using cached data",
            }, nil
        }
        return agent.ToolErrorResult{Action: agent.HookAllow}, nil
    },
}
```

When recovery succeeds, the error flag is cleared and `PostToolUse` sees a non-error result. Multiple error callbacks chain — the first recovery wins.

### Model Error Recovery

`OnModelError` fires when an LLM call fails. Return `HookModify` with a replacement response to recover:

```go
agent.Hooks{
    OnModelError: func(_ context.Context, mc agent.ModelErrorContext) (agent.ModelErrorResult, error) {
        return agent.ModelErrorResult{
            Action: agent.HookModify,
            Response: &llm.Response{
                Content: "Service temporarily unavailable. Please try again.",
            },
        }, nil
    },
}
```

This works in both `Chat()` and `ChatStream()` paths.

## Agent Lifecycle

### Short-Circuiting with BeforeAgent

`BeforeAgent` fires before an agent starts its run. Return `HookModify` with a response to skip the agent entirely:

```go
agent.Hooks{
    BeforeAgent: func(_ context.Context, ac agent.LifecycleContext) (agent.LifecycleResult, error) {
        if cached, ok := cache.Get(ac.Input); ok {
            return agent.LifecycleResult{
                Action:   agent.HookModify,
                Response: &agent.ChatResponse{Content: cached},
            }, nil
        }
        return agent.LifecycleResult{Action: agent.HookAllow}, nil
    },
}
```

Return `HookDeny` to block the agent run with a nil response.

### Modifying with AfterAgent

`AfterAgent` fires after an agent completes. Modify the response before it reaches the caller:

```go
agent.Hooks{
    AfterAgent: func(_ context.Context, ac agent.LifecycleContext) (agent.LifecycleResult, error) {
        modified := *ac.Response
        modified.Content = sanitize(modified.Content)
        return agent.LifecycleResult{
            Action:   agent.HookModify,
            Response: &modified,
        }, nil
    },
}
```

## Run Lifecycle

`BeforeRun` and `AfterRun` are observation-only hooks that fire at the very start and end of `Chat()`/`ChatStream()`:

```go
agent.Hooks{
    BeforeRun: func(_ context.Context, rc agent.RunContext) {
        metrics.StartTimer(rc.AgentName)
    },
    AfterRun: func(_ context.Context, rc agent.RunContext) {
        metrics.RecordDuration(rc.AgentName, rc.Duration)
        if rc.Error != nil {
            metrics.RecordError(rc.AgentName, rc.Error)
        }
    },
}
```

`AfterRun` receives the final response, any error, and the total duration.

## Input Validation

`OnUserMessage` fires when a user message arrives, before it reaches any agent logic. Use it to preprocess, validate, or reject messages:

```go
agent.Hooks{
    OnUserMessage: func(_ context.Context, uc agent.UserMessageContext) (agent.UserMessageResult, error) {
        if containsPII(uc.Message) {
            return agent.UserMessageResult{
                Action:     agent.HookDeny,
                DenyReason: "message contains PII",
            }, nil
        }
        return agent.UserMessageResult{
            Action:  agent.HookModify,
            Message: sanitizeInput(uc.Message),
        }, nil
    },
}
```

`OnUserMessage` does not fire for `Continue()`/`ContinueStream()` since those resume with tool results, not user messages.

## Cross-Cutting Event Observation

`OnEvent` fires on every hook event emitted during execution. Use it for logging, analytics, or event transformation:

```go
agent.Hooks{
    OnEvent: func(_ context.Context, evt agent.HookEvent) {
        log.Printf("[%s] agent=%s tool=%s", evt.Type, evt.AgentName, evt.ToolName)
    },
}
```

`OnEvent` fires once per hook-point invocation (after all hooks in the chain have run), not once per registered hook. It covers all event types except itself.

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
- **First recovery wins** — for error callbacks (`OnToolError`, `OnModelError`), the first `HookModify` response is used
- **nil fields are skipped** — you only need to set the hooks you care about

## Observation with NewObservingHooks

For pure observation (logging, metrics, streaming to a UI), use the `NewObservingHooks` helper. It wires all hooks to emit structured `HookEvent` values to a single callback:

```go
myAgent := agent.New(llmClient,
    agent.WithHooks(agent.NewObservingHooks(func(evt agent.HookEvent) {
        log.Printf("[%s] agent=%s branch=%s tool=%s",
            evt.Type, evt.AgentName, evt.Branch, evt.ToolName)
    })),
)
```

All observing hooks return `HookAllow` — they never block or modify execution. `OnEvent` is left nil in observing hooks to avoid double-emission.

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
| `Input` | `string` | Tool input, sub-agent task, or user message |
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
| `HookEventToolError` | `"tool_error"` | Tool returned an error |
| `HookEventModelError` | `"model_error"` | LLM call failed |
| `HookEventBeforeAgent` | `"before_agent"` | Before agent starts |
| `HookEventAfterAgent` | `"after_agent"` | After agent completes |
| `HookEventBeforeRun` | `"before_run"` | Start of Chat/ChatStream |
| `HookEventAfterRun` | `"after_run"` | End of Chat/ChatStream |
| `HookEventUserMessage` | `"user_message"` | User message received |
| `HookEventTeammateJoin` | `"teammate_join"` | Teammate spawned in a team |
| `HookEventTeammateLeave` | `"teammate_leave"` | Teammate left (stopped) |
| `HookEventTeamMessage` | `"team_message"` | Message sent between team members |
| `HookEventTeammateComplete` | `"teammate_complete"` | Teammate finished successfully |
| `HookEventTeammateError` | `"teammate_error"` | Teammate encountered an error |

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

Passed to `PreToolUse` and embedded in `PostToolUseContext` and `ToolErrorContext`:

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

### ToolErrorContext

Passed to `OnToolError`:

```go
type ToolErrorContext struct {
    ToolUseContext        // Embeds all fields from ToolUseContext
    Error    error
    Output   string
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

### ModelErrorContext

Passed to `OnModelError`:

```go
type ModelErrorContext struct {
    Messages  []message.Message
    Tools     []tool.BaseTool
    Error     error
    AgentName string
    TaskID    string
    Branch    string
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

### TeammateEventContext

Passed to `OnTeammateJoin`, `OnTeammateLeave`, `OnTeammateComplete`, and `OnTeammateError`:

```go
type TeammateEventContext struct {
    TeamName   string
    MemberID   string
    MemberName string
    Task       string
    Result     string
    Error      error
    Duration   time.Duration
}
```

### TeamMessageContext

Passed to `OnTeamMessage`:

```go
type TeamMessageContext struct {
    TeamName string
    Message  team.Message
}
```

See [Team Coordination](team-coordination.md) for full details on team hooks and messaging.

### LifecycleContext

Passed to `BeforeAgent` and `AfterAgent`:

```go
type LifecycleContext struct {
    AgentName string
    TaskID    string
    Branch    string
    Input     string
    Response  *ChatResponse   // nil for BeforeAgent, set for AfterAgent
}
```

### RunContext

Passed to `BeforeRun` and `AfterRun`:

```go
type RunContext struct {
    AgentName string
    TaskID    string
    Branch    string
    Input     string
    Response  *ChatResponse   // nil for BeforeRun, set for AfterRun
    Error     error           // nil for BeforeRun, set for AfterRun if failed
    Duration  time.Duration   // zero for BeforeRun
}
```

### UserMessageContext

Passed to `OnUserMessage`:

```go
type UserMessageContext struct {
    Message   string
    AgentName string
    TaskID    string
    Branch    string
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

This gives the UI real-time visibility into tool calls, model interactions, error recovery, and agent lifecycle — including nested agent hierarchies via `Branch`.
