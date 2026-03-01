# MCP (Model Context Protocol) Integration

This library integrates with the official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk) to provide seamless access to MCP servers and their tools.

## Stdio Connection (subprocess)

```go
import "github.com/joakimcarlsson/ai/tool"

mcpServers := map[string]tool.MCPServer{
    "filesystem": {
        Type:    tool.MCPStdio,
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"},
        Env:     []string{"NODE_ENV=production"},
    },
}

mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
if err != nil {
    log.Fatal(err)
}

response, err := client.SendMessages(ctx, messages, mcpTools)

defer tool.CloseMCPPool()
```

## SSE Connection (HTTP)

```go
mcpServers := map[string]tool.MCPServer{
    "remote": {
        Type: tool.MCPSse,
        URL:  "https://your-mcp-server.com/mcp",
        Headers: map[string]string{
            "Authorization": "Bearer your-token",
        },
    },
}

mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
if err != nil {
    log.Fatal(err)
}

defer tool.CloseMCPPool()
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/joakimcarlsson/ai/message"
    "github.com/joakimcarlsson/ai/model"
    llm "github.com/joakimcarlsson/ai/providers"
    "github.com/joakimcarlsson/ai/tool"
)

func main() {
    ctx := context.Background()

    mcpServers := map[string]tool.MCPServer{
        "context7": {
            Type:    tool.MCPStdio,
            Command: "npx",
            Args: []string{
                "-y",
                "@upstash/context7-mcp",
                "--api-key",
                os.Getenv("CONTEXT7_API_KEY"),
            },
        },
    }

    mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
    if err != nil {
        log.Fatal(err)
    }
    defer tool.CloseMCPPool()

    client, err := llm.NewLLM(
        model.ProviderOpenAI,
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
    )
    if err != nil {
        log.Fatal(err)
    }

    messages := []message.Message{
        message.NewUserMessage("Explain React hooks using Context7 to fetch the latest documentation"),
    }

    response, err := client.SendMessages(ctx, messages, mcpTools)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

## StreamableHTTP Connection

The newer MCP transport for HTTP-based servers:

```go
mcpServers := map[string]tool.MCPServer{
    "remote": {
        Type: tool.MCPStreamableHTTP,
        URL:  "https://your-mcp-server.com/mcp",
        Headers: map[string]string{
            "Authorization": "Bearer your-token",
        },
    },
}

mcpTools, err := tool.GetMcpTools(ctx, mcpServers)
defer tool.CloseMCPPool()
```

## Transport Types

| Type | Constant | Use Case |
|------|----------|----------|
| Stdio | `tool.MCPStdio` | Local subprocess (e.g., `npx` commands) |
| SSE | `tool.MCPSse` | HTTP server with Server-Sent Events |
| StreamableHTTP | `tool.MCPStreamableHTTP` | HTTP server with streamable responses |

## MCPServer Config

```go
type MCPServer struct {
    Command string            // Stdio: command to run
    Args    []string          // Stdio: command arguments
    Env     []string          // Stdio: environment variables
    Type    MCPType           // Transport type
    URL     string            // SSE/StreamableHTTP: server URL
    Headers map[string]string // SSE/StreamableHTTP: custom HTTP headers
}
```

## Features

- Supports stdio, SSE, and StreamableHTTP transports
- Connection pooling for efficient reuse of MCP server connections
- Custom HTTP headers for authentication on remote servers
- Automatic tool discovery and registration
- Compatible with all official MCP servers
- Tools are namespaced with server name (e.g., `context7_search`)
- Graceful cleanup with `CloseMCPPool()`
