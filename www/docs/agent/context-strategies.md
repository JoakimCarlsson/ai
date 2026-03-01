# Context Strategies

Context strategies automatically manage the context window when conversations grow beyond token limits.

## Available Strategies

### Sliding Window

Keep only the last N messages:

```go
import "github.com/joakimcarlsson/ai/tokens/sliding"

myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are a helpful assistant."),
    agent.WithSession("conv-1", store),
    agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(10)), 0),
)
```

### Truncate

Remove oldest messages to fit the token budget:

```go
import "github.com/joakimcarlsson/ai/tokens/truncate"

myAgent := agent.New(llmClient,
    agent.WithContextStrategy(truncate.Strategy(), 0),
)
```

### Summarize

Use an LLM to compress older messages into a summary:

```go
import "github.com/joakimcarlsson/ai/tokens/summarize"

myAgent := agent.New(llmClient,
    agent.WithContextStrategy(summarize.Strategy(llmClient), 0),
)
```

## How It Works

Before each LLM call, the agent:

1. Counts tokens for all messages + system prompt + tools
2. If total exceeds the limit, applies the strategy
3. The strategy reduces messages while preserving recent context
4. The session is updated if the strategy produces a session update (e.g., summary message)

## Custom Max Tokens

The second argument to `WithContextStrategy` sets a custom max token limit. Pass `0` to auto-calculate from the model's context window minus a 4096-token reserve.

```go
// Custom limit: 50k tokens
agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(20)), 50000)
```

## Custom Strategy

Implement the `tokens.Strategy` interface:

```go
type Strategy interface {
    Fit(ctx context.Context, input StrategyInput) (*StrategyResult, error)
}
```
