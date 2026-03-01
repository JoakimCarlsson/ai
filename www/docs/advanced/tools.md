# Tool Calling

## Defining a Tool

```go
import "github.com/joakimcarlsson/ai/tool"

type WeatherParams struct {
    Location string `json:"location" desc:"City name"`
    Units    string `json:"units" desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

type WeatherTool struct{}

func (w *WeatherTool) Info() tool.ToolInfo {
    return tool.NewToolInfo("get_weather", "Get current weather for a location", WeatherParams{})
}

func (w *WeatherTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
    var input WeatherParams
    json.Unmarshal([]byte(params.Input), &input)
    return tool.NewTextResponse("Sunny, 22°C"), nil
}
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

info := tool.NewToolInfo("search", "Search documents", SearchParams{})
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
