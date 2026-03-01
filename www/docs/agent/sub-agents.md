# Sub-Agents

Sub-agents let an orchestrator delegate tasks to specialized child agents. Each sub-agent becomes a callable tool.

## Setup

```go
researcher := agent.New(llmClient,
    agent.WithSystemPrompt("You are a research specialist."),
    agent.WithTools(&webSearchTool{}),
)

writer := agent.New(llmClient,
    agent.WithSystemPrompt("You are a content writer."),
)

orchestrator := agent.New(llmClient,
    agent.WithSystemPrompt("You coordinate research and writing tasks."),
    agent.WithSubAgents(
        agent.SubAgentConfig{Name: "researcher", Description: "Researches topics", Agent: researcher},
        agent.SubAgentConfig{Name: "writer", Description: "Writes content", Agent: writer},
    ),
)

response, _ := orchestrator.Chat(ctx, "Research and write about quantum computing")
```

## How It Works

1. Each `SubAgentConfig` registers a tool named after the sub-agent
2. The orchestrator LLM decides when to delegate a task
3. The sub-agent runs to completion and returns its response
4. The orchestrator continues with the sub-agent's output

## SubAgentConfig

```go
type SubAgentConfig struct {
    Name        string  // Tool name the orchestrator calls
    Description string  // Describes when to use this sub-agent
    Agent       *Agent  // The sub-agent instance
}
```

## Background Execution

Sub-agents can run asynchronously by passing `background: true`. The orchestrator gets a `task_id` immediately and can check status or wait for results later.

```go
orchestrator := agent.New(llmClient,
    agent.WithSystemPrompt(`Launch background tasks, then collect results.`),
    agent.WithSubAgents(
        agent.SubAgentConfig{
            Name:        "researcher",
            Description: "Research a topic. Supports background: true for async execution.",
            Agent:       researcher,
        },
    ),
)
```

When the LLM calls the sub-agent with `background: true`:

1. The task launches in a goroutine and returns `{"task_id": "task-1", "status": "launched"}`
2. Three task management tools are automatically available: `get_task_result`, `stop_task`, `list_tasks`
3. The orchestrator uses `get_task_result` with `wait: true` to collect results

See [Background Agents](background-agents.md) for the full tool reference and examples.
