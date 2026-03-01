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

Sub-agents can also run as background tasks using the task manager. See the `example/background_agents/` example.
