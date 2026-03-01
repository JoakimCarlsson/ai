# Agent Framework

The agent package provides multi-agent orchestration with automatic tool execution, session management, persistent memory, sub-agents, handoffs, fan-out, and context strategies.

## Basic Agent

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/session"
)

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithTools(&weatherTool{}),
    agent.WithSession("user-123", session.FileStore("./sessions")),
)

response, _ := myAgent.Chat(ctx, "What's the weather in Tokyo?")
fmt.Println(response.Content)
```

## How It Works

When you call `Chat()`, the agent:

1. Builds the message history (system prompt + session messages + user message)
2. Sends messages to the LLM
3. If the LLM requests tool calls, executes them automatically
4. Loops back to step 2 with tool results until the LLM responds with text
5. Persists the conversation to the session store

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithSystemPrompt(prompt)` | Sets the agent's behavior | none |
| `WithTools(tools...)` | Adds tools the agent can use | none |
| `WithSession(id, store)` | Enables conversation persistence | none |
| `WithMemory(id, store, opts...)` | Enables long-term memory | none |
| `WithMaxIterations(n)` | Max tool execution loops | 10 |
| `WithAutoExecute(bool)` | Auto-execute tool calls | true |
| `WithContextStrategy(strategy, maxTokens)` | Context window management | none |
| `WithSequentialToolExecution()` | Disable parallel tool execution | parallel |
| `WithMaxParallelTools(n)` | Limit concurrent tool execution | unlimited |
| `WithState(map)` | Template variables for system prompt | none |
| `WithInstructionProvider(fn)` | Dynamic system prompt generation | none |
| `WithSubAgents(configs...)` | Register child agents | none |
| `WithHandoffs(configs...)` | Register peer agents for transfer | none |
| `WithFanOut(configs...)` | Register parallel task distribution | none |

## ChatResponse

```go
type ChatResponse struct {
    Content        string
    ToolCalls      []message.ToolCall
    ToolResults    []ToolExecutionResult
    Usage          llm.TokenUsage
    FinishReason   message.FinishReason
    AgentName      string         // Set when a handoff occurred
    TotalToolCalls int
    TotalDuration  time.Duration
    TotalTurns     int
}
```

All metrics are aggregated across the full agent loop, not just the final LLM call:

| Field | Description |
|-------|-------------|
| `TotalTurns` | Number of LLM round-trips (API calls) made |
| `TotalDuration` | Wall-clock time from `Chat()` entry to return |
| `TotalToolCalls` | Total tool invocations across all iterations |
| `ToolResults` | Results of every tool execution during the conversation |

## Debug APIs

Inspect the messages that would be sent to the LLM after applying context strategies:

```go
// Non-destructive — does not modify the session
messages, err := myAgent.PeekContextMessages(ctx, "Hello")

// Modifying — adds the user message to the session
messages, err := myAgent.BuildContextMessages(ctx, "Hello")
```

Use `PeekContextMessages` to debug context window management without side effects.
