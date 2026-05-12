# Tool Calling

## Defining a Tool

```go
import "github.com/joakimcarlsson/ai/tool"

type WeatherParams struct {
    Location string `json:"location" desc:"City name"`
    Units    string `json:"units" desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

type WeatherTool struct{}

func (w *WeatherTool) Info() tool.Info {
    return tool.NewInfo("get_weather", "Get current weather for a location", WeatherParams{})
}

func (w *WeatherTool) Run(ctx context.Context, params tool.Call) (tool.Response, error) {
    var input WeatherParams
    json.Unmarshal([]byte(params.Input), &input)
    return tool.NewTextResponse("Sunny, 22°C"), nil
}
```

## Function Tools

For simple tools that are just a function, use `functiontool.New` to skip the struct boilerplate:

```go
import "github.com/joakimcarlsson/ai/tool/functiontool"

type WeatherParams struct {
    Location string `json:"location" desc:"City name"`
    Units    string `json:"units" desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

weatherTool := functiontool.New("get_weather", "Get current weather for a location",
    func(ctx context.Context, p WeatherParams) (string, error) {
        return fmt.Sprintf("Sunny, 22°C in %s", p.Location), nil
    },
)
```

The JSON schema is inferred from the parameter struct using the same struct tags as `tool.NewInfo`. The result is a standard `BaseTool` that works with the registry, toolsets, hooks, and agent system.

### Supported Signatures

The function's first parameter can optionally be `context.Context`, and the second can be a struct for input parameters. Both are optional:

```go
// With context and params
functiontool.New("name", "desc", func(ctx context.Context, p Params) (string, error) { ... })

// Params only (no context)
functiontool.New("name", "desc", func(p Params) (string, error) { ... })

// Context only (no input schema)
functiontool.New("name", "desc", func(ctx context.Context) (string, error) { ... })

// No inputs at all
functiontool.New("name", "desc", func() (string, error) { ... })
```

### Return Types

The first return value determines the response type:

```go
// String → tool.NewTextResponse
func(p Params) (string, error)

// tool.Response → passed through directly
func(p Params) (tool.Response, error)

// Any other type → tool.NewJSONResponse (auto-marshaled)
func(p Params) (MyStruct, error)
```

### Options

```go
// Require human confirmation before execution
functiontool.New("delete", "Delete records", deleteFn, functiontool.WithConfirmation())
```

## Using Tools with LLM

```go
weatherTool := &WeatherTool{}
tools := []tool.BaseTool{weatherTool}

response, err := client.SendMessages(ctx, messages, tools)
```

## Provider Built-in Tools

Beyond client-side function tools (the `BaseTool` interface above), most
vendors expose **server-side built-in tools** that run inside the provider's
infrastructure: web search, code execution, file search, and so on. The agent
loop never sees these as `ToolCall` entries — the provider executes the tool
internally and inlines the result into the assistant message. Citations and
other structured output land in `Response.ProviderMetadata` under a
provider-namespaced key.

Built-ins are opt-in per-client via `With*` options. Off by default — some are
billed per call.

```go
import (
    llmanthropic "github.com/joakimcarlsson/ai/llm/anthropic"
    llmgemini "github.com/joakimcarlsson/ai/llm/gemini"
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    llmgroq "github.com/joakimcarlsson/ai/llm/groq"
    llmxai "github.com/joakimcarlsson/ai/llm/xai"
)

// Anthropic web_search
anthropic := llmanthropic.NewLLM(
    llmanthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llmanthropic.WithModel(model.AnthropicModels[model.Claude47Opus]),
    llmanthropic.WithWebSearch(llmanthropic.WebSearchConfig{
        MaxUses:        5,
        AllowedDomains: []string{"go.dev", "pkg.go.dev"},
    }),
)

// Gemini google_search + code_execution
gemini := llmgemini.NewLLM(
    llmgemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    llmgemini.WithModel(model.GeminiModels[model.Gemini25Flash]),
    llmgemini.WithGoogleSearch(),
    llmgemini.WithCodeExecution(),
)

// OpenAI Responses API: web_search, file_search, code_interpreter
openaiR := llmopenai.NewResponsesLLM(
    llmopenai.WithResponsesAPIKey(os.Getenv("OPENAI_API_KEY")),
    llmopenai.WithResponsesModel(model.OpenAIModels[model.GPT5]),
    llmopenai.WithWebSearch(llmopenai.WebSearchOpts{
        SearchContextSize: llmopenai.SearchContextMedium,
    }),
)

// Groq compound models: browser_search, code_execution, visit_website
groq := llmgroq.NewCompoundLLM(
    llmgroq.WithCompoundAPIKey(os.Getenv("GROQ_API_KEY")),
    llmgroq.WithCompoundModel(model.Model{APIModel: "groq/compound"}),
    llmgroq.WithBrowserSearch(),
)

// xAI Responses API: web_search, x_search, code_execution
xai := llmxai.NewResponsesLLM(
    llmxai.WithResponsesAPIKey(os.Getenv("XAI_API_KEY")),
    llmxai.WithResponsesModel(model.XAIModels[model.XAIGrok4]),
    llmxai.WithWebSearch(),
    llmxai.WithXSearch(),
)
```

Built-in tool results — citations, search chunks, executed-tool summaries —
arrive on `Response.ProviderMetadata` under namespaced keys:

```go
resp, _ := client.SendMessages(ctx, messages, nil)
fmt.Println(resp.Content)

if results, ok := resp.ProviderMetadata["anthropic.web_search_results"].([]map[string]any); ok {
    for _, r := range results {
        fmt.Printf("  ↳ %s (%s)\n", r["title"], r["url"])
    }
}
```

The keys vary per provider:

| Provider | Key | Shape |
|---|---|---|
| Anthropic | `anthropic.web_search_results` | `[]map[string]any` — URL, title, page age, encrypted content |
| Gemini | `gemini.grounding` | `map[string]any` — `web_search_queries`, `chunks` (URI/title/domain) |
| Gemini | `gemini.url_context` | `map[string]any` — retrieved URLs and status |
| OpenAI | `openai.url_citations` | `[]map[string]any` — URL, title, start/end indices |
| Groq | `groq.executed_tools` | `[]map[string]any` — Groq's raw executed-tool entries |
| xAI | `xai.citations` | `[]map[string]any` — URL, title, start/end indices |

Built-in tools are **not dispatched through the agent loop** — they don't
appear as `message.ToolCall` entries and don't go through `registry.Execute`.
This means you can use them alongside ordinary function tools without
collision. See the [`builtin-tools`](https://github.com/joakimcarlsson/ai/tree/main/examples/llm/builtin-tools)
example for a runnable provider-switch demo.

## Struct Tag Schema Generation

Generate JSON schemas automatically from Go structs:

```go
type SearchParams struct {
    Query   string   `json:"query" desc:"Search query"`
    Limit   int      `json:"limit" desc:"Max results" required:"false"`
    Filters []string `json:"filters" desc:"Filter tags" required:"false"`
}

info := tool.NewInfo("search", "Search documents", SearchParams{})
```

Supported tags:

| Tag | Description |
|-----|-------------|
| `json` | Parameter name |
| `desc` | Parameter description |
| `required` | `"true"` or `"false"` (non-pointer fields default to required) |
| `enum` | Comma-separated allowed values |

## Rich Tool Responses

```go
// Text response
tool.NewTextResponse("Result text")

// JSON response (auto-marshals any value)
tool.NewJSONResponse(map[string]any{"status": "ok", "count": 42})

// File/binary response
tool.NewFileResponse(pdfBytes, "application/pdf")

// Image response (base64)
tool.NewImageResponse(base64ImageData)

// Error response
tool.NewTextErrorResponse("Something went wrong")
```

## Parsing Tool Input

The agent package provides a generic helper:

```go
input, err := agent.ParseToolInput[WeatherParams](params.Input)
```

## Requiring Confirmation

Set `RequireConfirmation` on a tool's `Info` to require human approval before execution:

```go
func (t *DeleteTool) Info() tool.Info {
    info := tool.NewInfo("delete_records", "Delete database records", DeleteParams{})
    info.RequireConfirmation = true
    return info
}
```

Tools can also request confirmation dynamically from within `Run()`:

```go
func (t *TransferTool) Run(ctx context.Context, params tool.Call) (tool.Response, error) {
    if amount > 10000 {
        if err := tool.RequestConfirmation(ctx, "Large transfer", params); err != nil {
            return tool.Response{}, err
        }
    }
    // ...
}
```

Both require a `ConfirmationProvider` on the agent. See [Tool Confirmation](../agent/confirmation.md) for the full protocol.

## Toolsets

For grouping, filtering, and dynamically controlling which tools are available at runtime, see [Toolsets](../agent/toolsets.md).
