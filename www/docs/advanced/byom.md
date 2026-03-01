# BYOM (Bring Your Own Model)

Use Ollama, LocalAI, vLLM, LM Studio, or any OpenAI-compatible inference server.

## Setup

```go
// 1. Create model
llamaModel := model.NewCustomModel(
    model.WithModelID("llama3.2"),
    model.WithAPIModel("llama3.2:latest"),
)

// 2. Register provider
ollama := llm.RegisterCustomProvider("ollama", llm.CustomProviderConfig{
    BaseURL:      "http://localhost:11434/v1",
    DefaultModel: llamaModel,
})

// 3. Use it
client, _ := llm.NewLLM(ollama)
response, _ := client.SendMessages(ctx, messages, nil)
```

## Supported Servers

Any server that implements the OpenAI-compatible API:

- **Ollama** — `http://localhost:11434/v1`
- **LocalAI** — `http://localhost:8080/v1`
- **vLLM** — `http://localhost:8000/v1`
- **LM Studio** — `http://localhost:1234/v1`

See `example/byom/main.go` for a complete example.
