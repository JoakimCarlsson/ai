# Tool Confirmation

The confirmation protocol lets tools require human approval before executing. The framework provides the mechanism — consumers provide the UI/interaction layer.

## Setup

Register a `ConfirmationProvider` on the agent. The provider is called whenever a tool requires confirmation and blocks until the consumer provides a decision.

```go
myAgent := agent.New(llmClient,
    agent.WithTools(&DeleteTool{}),
    agent.WithConfirmationProvider(
        func(ctx context.Context, req tool.ConfirmationRequest) (bool, error) {
            // Present req to the user, wait for their decision
            return askUser(req.ToolName, req.Input, req.Hint), nil
        },
    ),
)
```

Return `true` to approve, `false` to reject. If the provider returns an error, the tool call fails with that error.

## Declarative Confirmation

Set `RequireConfirmation` on a tool's `Info` to require approval before `Run()` is called:

```go
func (t *DeleteTool) Info() tool.Info {
    info := tool.NewInfo("delete_records", "Delete database records", DeleteParams{})
    info.RequireConfirmation = true
    return info
}
```

When the agent encounters this tool, it calls the `ConfirmationProvider` before executing. If no provider is configured, the tool runs normally — confirmation is opt-in.

## Dynamic Confirmation

Tools can request confirmation from within `Run()` for conditional approval:

```go
func (t *TransferTool) Run(ctx context.Context, params tool.Call) (tool.Response, error) {
    var input TransferParams
    json.Unmarshal([]byte(params.Input), &input)

    if input.Amount > 10000 {
        err := tool.RequestConfirmation(ctx, "Large transfer exceeding $10,000", input)
        if err != nil {
            return tool.Response{}, err
        }
    }

    // Proceed with transfer
    return tool.NewTextResponse("Transfer complete"), nil
}
```

`RequestConfirmation` blocks until the consumer decides. If rejected, it returns `tool.ErrConfirmationRejected` — propagate this error to halt execution. If no `ConfirmationProvider` is configured, `RequestConfirmation` is a no-op (auto-approve).

## ConfirmationRequest

The provider receives a `ConfirmationRequest` with context about the tool call:

```go
type ConfirmationRequest struct {
    ToolCallID string // Unique ID of this tool call
    ToolName   string // Name of the tool
    Input      string // JSON-encoded arguments
    Hint       string // Human-readable description (dynamic confirmation only)
    Payload    any    // Arbitrary structured data (dynamic confirmation only)
}
```

For declarative confirmation (`RequireConfirmation` flag), `Hint` and `Payload` are empty. For dynamic confirmation (`RequestConfirmation`), they carry the values passed by the tool.

## Toolset-Level Confirmation

Use `tool.WithConfirmation` to mark all tools in a toolset as requiring confirmation:

```go
dangerousTools := tool.NewToolset("dangerous",
    &DeleteTool{},
    &DropTableTool{},
    &FormatDiskTool{},
)

confirmed := tool.WithConfirmation(dangerousTools)

myAgent := agent.New(llmClient,
    agent.WithToolsets(confirmed),
    agent.WithConfirmationProvider(myProvider),
)
```

This sets `RequireConfirmation = true` on every tool in the toolset without modifying the originals.

## Streaming

In the streaming path (`ChatStream`), an `EventConfirmationRequired` event is emitted before the provider blocks. This allows the consumer to present a UI and then unblock the provider:

```go
for event := range myAgent.ChatStream(ctx, "Delete old records") {
    switch event.Type {
    case types.EventConfirmationRequired:
        req := event.ConfirmationRequest
        fmt.Printf("Tool %q wants to run with input: %s\n", req.ToolName, req.Input)
        // The provider is blocking — respond via whatever mechanism it uses
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventComplete:
        fmt.Println("\nDone!")
    }
}
```

A common pattern is to use a channel-based provider that the streaming consumer unblocks:

```go
type approval struct {
    approved bool
    ch       chan struct{}
}

pending := make(map[string]*approval)
var mu sync.Mutex

provider := func(ctx context.Context, req tool.ConfirmationRequest) (bool, error) {
    a := &approval{ch: make(chan struct{})}
    mu.Lock()
    pending[req.ToolCallID] = a
    mu.Unlock()
    <-a.ch // Block until consumer decides
    return a.approved, nil
}

// In the stream consumer, when EventConfirmationRequired arrives:
// mu.Lock()
// a := pending[req.ToolCallID]
// mu.Unlock()
// a.approved = userClickedApprove
// close(a.ch)
```

## Interaction with Hooks

`PreToolUse` hooks run before confirmation. If a hook denies the tool, the confirmation provider is never called:

```
PreToolUse hooks → Confirmation check → tool.Run()
```

This means hooks enforce policy (rate limits, blocklists), while confirmation handles human approval.

## Handoffs

Each agent has its own `ConfirmationProvider`. When a handoff occurs, the new agent's provider is used. If the target agent has no provider, its tools run without confirmation.

## Auto-Approve Patterns

The provider is a regular function — implement any approval logic:

```go
// Always approve (useful for testing)
agent.WithConfirmationProvider(
    func(_ context.Context, _ tool.ConfirmationRequest) (bool, error) {
        return true, nil
    },
)

// Check a database of pre-approved tools
agent.WithConfirmationProvider(
    func(ctx context.Context, req tool.ConfirmationRequest) (bool, error) {
        return db.IsToolPreApproved(ctx, userID, req.ToolName)
    },
)

// Approve safe tools, prompt for dangerous ones
agent.WithConfirmationProvider(
    func(ctx context.Context, req tool.ConfirmationRequest) (bool, error) {
        if req.ToolName == "read_file" {
            return true, nil
        }
        return promptUser(ctx, req)
    },
)
```
