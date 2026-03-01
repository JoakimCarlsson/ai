# Streaming

`ChatStream` returns a channel of events for real-time response handling.

## Basic Usage

```go
for event := range myAgent.ChatStream(ctx, "Tell me a story") {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventThinkingDelta:
        // Extended thinking content (if supported)
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

## ChatEvent

```go
type ChatEvent struct {
    Type       types.EventType
    Content    string
    Thinking   string
    ToolCall   *message.ToolCall
    ToolResult *ToolExecutionResult
    Response   *ChatResponse
    Error      error
    AgentName  string
}
```
