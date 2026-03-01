# Fan-Out

Fan-out distributes multiple tasks to worker agents in parallel and collects results.

## Setup

```go
researcher := agent.New(llmClient,
    agent.WithSystemPrompt("Research the given topic thoroughly."),
)

coordinator := agent.New(llmClient,
    agent.WithSystemPrompt("You coordinate parallel research tasks."),
    agent.WithFanOut(agent.FanOutConfig{
        Name:           "research",
        Description:    "Research multiple topics in parallel",
        Agent:          researcher,
        MaxConcurrency: 3,
    }),
)

response, _ := coordinator.Chat(ctx, "Compare AI, blockchain, and quantum computing")
```

## How It Works

1. The `FanOutConfig` registers a tool that accepts multiple tasks
2. When the coordinator calls the fan-out tool, all tasks run concurrently
3. `MaxConcurrency` limits how many worker agents run at the same time
4. Results are collected and returned to the coordinator

## FanOutConfig

```go
type FanOutConfig struct {
    Name           string  // Tool name
    Description    string  // Describes when to use fan-out
    Agent          *Agent  // Worker agent (cloned per task)
    MaxConcurrency int     // Max parallel workers (0 = unlimited)
}
```
