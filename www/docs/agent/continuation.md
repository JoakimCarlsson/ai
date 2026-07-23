# Loop Continuation

The continuation protocol lets you control agent execution when it hits its tool iteration limit. The framework provides the mechanism — consumers provide the logic to approve, decline, or timeout the continuation.

## Setup

Register a `ContinuationProvider` on the agent. The provider is called whenever the agent reaches its `MaxIterations` limit while trying to execute tools. It blocks until the consumer provides a decision.

```go
myAgent := agent.New(llmClient,
    agent.WithMaxIterations(5),
    agent.WithContinuationProvider(
        func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationResponse, error) {
            // Present req to the user or apply automated logic
            return agent.ContinuationResponse{
                Decision: agent.ContinuationApprove,
            }, nil
        },
    ),
)
```

Return `agent.ContinuationApprove` inside the response to reset the iteration counter and continue the loop, `agent.ContinuationDecline` to stop execution gracefully, or `agent.ContinuationTimeout` if the wait period expires. If the provider returns an error or anything other than `ContinuationApprove`, the agent halts with an error.

## ContinuationResponse

The provider must return a `ContinuationResponse` struct:

```go
type ContinuationResponse struct {
    Decision         ContinuationDecision // Approve, Decline, or Timeout
    Message          string               // Optional "steering message" to append to the conversation
    DiscardToolCalls bool                 // If true, pending tool calls are skipped
    ToolMessage      string               // Custom error message for skipped tools
}
```

- **Message:** Supply an optional steering message. If the decision is `ContinuationApprove`, this message acts as a steering instruction for the next iteration (e.g., "Please try using the search tool instead"). If `ContinuationDecline` or `ContinuationTimeout`, it overrides the default system halt notification to explain why it was stopped.
- **DiscardToolCalls & ToolMessage:** When approving, you can set `DiscardToolCalls: true` to prevent the pending tools from executing. The agent injects synthetic error results for those tools, optionally using `ToolMessage` (defaults to "Tool execution canceled by user during continuation.").

## ContinuationRequest

The provider receives a `ContinuationRequest` with context about the agent's current state:

```go
type ContinuationRequest struct {
    MaxIterations   int                  // The limit that was reached
    TotalIterations int                  // Total iterations across all continuations
    ToolCalls       []message.ToolCall   // The tools the agent wants to execute next
}
```

## Streaming

In the streaming path (`ChatStream`), an `EventContinuationRequired` event is emitted before the provider blocks. This allows the consumer to present a UI and then unblock the provider:

```go
for event := range myAgent.ChatStream(ctx, "Analyze this large dataset") {
    switch event.Type {
    case types.EventContinuationRequired:
        req := event.ContinuationRequest
        fmt.Printf("Agent reached %d iterations. It wants to call %d tools.\n", req.MaxIterations, len(req.ToolCalls))
        // The provider is blocking — respond via whatever mechanism it uses
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Println("\nDone!")
    }
}
```

A common pattern is to use a channel-based provider that the streaming consumer unblocks, similar to tool confirmation:

```go
type continuationApproval struct {
    response agent.ContinuationResponse
    ch       chan struct{}
}

// In the provider:
// ... wait on channel, return decision ...

// In the stream consumer:
// ... unblock channel with decision ...
```

## Auto-Approve Patterns

The provider is a regular function — implement any continuation logic:

```go
// Always approve (effectively disables max iterations, use with caution!)
agent.WithContinuationProvider(
    func(_ context.Context, _ agent.ContinuationRequest) (agent.ContinuationResponse, error) {
        return agent.ContinuationResponse{Decision: agent.ContinuationApprove}, nil
    },
)

// Approve up to a hard limit
agent.WithContinuationProvider(
    func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationResponse, error) {
        if req.TotalIterations >= 20 {
            return agent.ContinuationResponse{Decision: agent.ContinuationDecline}, nil
        }
        return agent.ContinuationResponse{Decision: agent.ContinuationApprove}, nil
    },
)
```
