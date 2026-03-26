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
