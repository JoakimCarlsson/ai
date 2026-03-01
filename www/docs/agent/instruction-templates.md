# Instruction Templates

Dynamic system prompts using template variables or runtime-generated instructions.

## Static Templates

Use Go template syntax (`{{.var}}`) with `WithState`:

```go
myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are {{.role}}. Help {{.user_name}} with their tasks."),
    agent.WithState(map[string]any{
        "role":      "a coding assistant",
        "user_name": "Alice",
    }),
)
```

## Conditional Templates

```go
myAgent := agent.New(llmClient,
    agent.WithSystemPrompt(`You are a helpful assistant.
{{if .extra_context}}
Additional context: {{.extra_context}}
{{end}}`),
    agent.WithState(map[string]any{
        "extra_context": "The user prefers concise answers.",
    }),
)
```

## Dynamic Provider

For fully dynamic prompts generated at runtime:

```go
myAgent := agent.New(llmClient,
    agent.WithInstructionProvider(func(ctx context.Context, state map[string]any) (string, error) {
        return fmt.Sprintf(
            "Current time: %s\nYou are a helpful assistant.",
            time.Now().Format(time.RFC3339),
        ), nil
    }),
)
```

The instruction provider receives the state map and can use it alongside any other runtime data (database lookups, feature flags, etc.).
