# Streaming

`ChatStream` returns a channel of events for real-time response handling.

## Basic Usage

```go
for event := range myAgent.ChatStream(ctx, "Tell me a story") {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventThinkingDelta:
        fmt.Print(event.Thinking)
    case types.EventToolUseStart:
        fmt.Printf("\nUsing tool: %s\n", event.ToolCall.Name)
    case types.EventToolUseStop:
        if event.ToolResult != nil {
            fmt.Printf("Tool result: %s\n", event.ToolResult.Output)
        }
    case types.EventHandoff:
        fmt.Printf("Handed off to: %s\n", event.AgentName)
    case types.EventComplete:
        fmt.Printf("\nDone! Tokens: %d\n", event.Response.Usage.InputTokens)
    case types.EventError:
        log.Fatal(event.Error)
    }
}
```

## ContinueStream

The streaming variant of `Continue()`:

```go
for event := range myAgent.ContinueStream(ctx, toolResults) {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Println("\nDone!")
    }
}
```

## Event Types

| Event | Field | Description |
|-------|-------|-------------|
| `EventContentStart` | — | Content generation is beginning |
| `EventContentDelta` | `Content` | Partial text token |
| `EventContentStop` | — | Content generation finished |
| `EventToolUseStart` | `ToolCall` | Tool invocation starting (name, ID) |
| `EventToolUseDelta` | `ToolCall` | Partial tool input JSON |
| `EventToolUseStop` | `ToolResult` | Tool execution completed with result |
| `EventThinkingDelta` | `Thinking` | Chain-of-thought reasoning (if model supports it) |
| `EventHandoff` | `AgentName` | Control transferred to another agent |
| `EventConfirmationRequired` | `ConfirmationRequest` | Tool awaiting human approval ([details](confirmation.md)) |
| `EventTeammateSpawned` | `AgentName` | A new teammate was spawned in a [team](team-coordination.md) |
| `EventTeamMessage` | `AgentName` | A message was sent between team members |
| `EventTeammateComplete` | `AgentName` | A teammate finished its task successfully |
| `EventTeammateError` | `AgentName`, `Error` | A teammate encountered an error |
| `EventComplete` | `Response` | Streaming finished — contains the full `ChatResponse` |
| `EventError` | `Error` | An error occurred during streaming |
| `EventWarning` | `Error` | A non-fatal warning |

## ChatEvent

```go
type ChatEvent struct {
    Type       types.EventType
    Content    string              // EventContentDelta
    Thinking   string              // EventThinkingDelta
    ToolCall   *message.ToolCall   // EventToolUseStart/Delta
    ToolResult *ToolExecutionResult // EventToolUseStop
    Response   *ChatResponse       // EventComplete
    Error               error                    // EventError, EventWarning
    AgentName           string                   // EventHandoff
    ConfirmationRequest *tool.ConfirmationRequest // EventConfirmationRequired
}
```
