# Continue/Resume

`Continue()` lets you manually execute tool calls and feed results back into the agent loop. This is useful when tools require human approval, external API calls, or custom execution logic.

## Setup

```go
myAgent := agent.New(llmClient,
    agent.WithAutoExecute(false), // Don't auto-execute tools
    agent.WithSession("conv-1", session.MemoryStore()),
)
```

## Usage

```go
// First call returns pending tool calls instead of executing them
response, _ := myAgent.Chat(ctx, "Search for flights to Tokyo")

// Inspect what tools the LLM wants to call
for _, tc := range response.ToolCalls {
    fmt.Printf("Tool: %s, Input: %s\n", tc.Name, tc.Input)
}

// Execute tools externally with your own logic
results := []message.ToolResult{
    {
        ToolCallID: response.ToolCalls[0].ID,
        Name:       "search_flights",
        Content:    `{"flights": [{"airline": "JAL", "price": 850}]}`,
    },
}

// Resume the agent loop with results
response, _ = myAgent.Continue(ctx, results)
fmt.Println(response.Content)
```

## Streaming Variant

```go
for event := range myAgent.ContinueStream(ctx, results) {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Println("\nDone!")
    }
}
```

!!! note
    `Continue()` requires a session to be configured, since it needs to restore conversation state from the previous `Chat()` call.
