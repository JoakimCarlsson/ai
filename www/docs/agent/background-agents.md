# Background Agents

Background agents let the orchestrator launch sub-agents asynchronously. Tasks run in goroutines and the orchestrator can continue working, check status, or wait for results.

## Setup

```go
researcher := agent.New(llmClient,
    agent.WithSystemPrompt("You are a concise research assistant."),
)

orchestrator := agent.New(llmClient,
    agent.WithSystemPrompt(`You coordinate research tasks.
1. Launch background tasks with background: true
2. Collect results with get_task_result (wait: true)
3. Synthesize the results`),
    agent.WithSubAgents(
        agent.SubAgentConfig{
            Name:        "researcher",
            Description: "Research a topic. Supports background: true for async execution.",
            Agent:       researcher,
        },
    ),
)
```

## How It Works

1. The orchestrator calls a sub-agent tool with `background: true`
2. The sub-agent launches in a goroutine and returns a `task_id` immediately
3. Three task management tools are auto-registered for the orchestrator:

| Tool | Description |
|------|-------------|
| `get_task_result` | Check status or wait for a background task to complete |
| `stop_task` | Cancel a running background task |
| `list_tasks` | List all background tasks and their status |

## Task Lifecycle

Tasks move through these states:

| Status | Description |
|--------|-------------|
| `running` | Task is currently executing |
| `completed` | Task finished successfully |
| `failed` | Task encountered an error |
| `cancelled` | Task was explicitly cancelled |

## Tool Reference

### get_task_result

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_id` | string | yes | The task ID returned when the task was launched |
| `wait` | bool | no | If true, block until the task completes |
| `timeout` | int | no | Max wait time in milliseconds. 0 means no timeout |

### stop_task

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_id` | string | yes | The task ID to cancel |

### list_tasks

No parameters. Returns all tasks with their ID, agent name, and status.

## Streaming Example

```go
for event := range orchestrator.ChatStream(ctx, "Compare Go and Rust. Research each in the background.") {
    switch event.Type {
    case types.EventContentDelta:
        fmt.Print(event.Content)
    case types.EventError:
        log.Fatal(event.Error)
    }
    if event.ToolResult != nil {
        fmt.Printf("\n[Tool: %s → %s]\n", event.ToolResult.ToolName, event.ToolResult.Output)
    }
}
```

## Sub-Agent Input

When a sub-agent is called, it accepts these parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task` | string | yes | The task or question to send to the sub-agent |
| `background` | bool | no | If true, run in background and return a task ID |
| `max_turns` | int | no | Maximum tool-execution turns. 0 uses the agent default |
