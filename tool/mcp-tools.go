package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

type mcpTool struct {
	mcpName   string
	tool      mcp.Tool
	mcpConfig MCPServer
}

type MCPClient interface {
	Initialize(
		ctx context.Context,
		request mcp.InitializeRequest,
	) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func (b *mcpTool) Info() ToolInfo {
	return ToolInfo{
		Name:        fmt.Sprintf("%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  b.tool.InputSchema.Properties,
		Required:    b.tool.InputSchema.Required,
	}
}

func runTool(ctx context.Context, c MCPClient, toolName string, input string) (ToolResponse, error) {
	defer c.Close()
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "llm",
		Version: "0.0.1",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	toolRequest := mcp.CallToolRequest{}
	toolRequest.Params.Name = toolName
	var args map[string]any
	if err = json.Unmarshal([]byte(input), &args); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	toolRequest.Params.Arguments = args
	result, err := c.CallTool(ctx, toolRequest)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	output := ""
	for _, v := range result.Content {
		if v, ok := v.(mcp.TextContent); ok {
			output = v.Text
		} else {
			output = fmt.Sprintf("%v", v)
		}
	}

	return NewTextResponse(output), nil
}

func (b *mcpTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	switch b.mcpConfig.Type {
	case MCPStdio:
		c, err := client.NewStdioMCPClient(
			b.mcpConfig.Command,
			b.mcpConfig.Env,
			b.mcpConfig.Args...,
		)
		if err != nil {
			return NewTextErrorResponse(err.Error()), nil
		}
		return runTool(ctx, c, b.tool.Name, params.Input)
	case MCPSse:
		c, err := client.NewSSEMCPClient(
			b.mcpConfig.URL,
			client.WithHeaders(b.mcpConfig.Headers),
		)
		if err != nil {
			return NewTextErrorResponse(err.Error()), nil
		}
		return runTool(ctx, c, b.tool.Name, params.Input)
	}

	return NewTextErrorResponse("invalid mcp type"), nil
}

func newMcpTool(
	name string,
	tool mcp.Tool,
	mcpConfig MCPServer,
) BaseTool {
	return &mcpTool{
		mcpName:   name,
		tool:      tool,
		mcpConfig: mcpConfig,
	}
}

var mcpTools []BaseTool

func getTools(ctx context.Context, name string, m MCPServer, c MCPClient) []BaseTool {
	var stdioTools []BaseTool
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "llm",
		Version: "0.0.1",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		slog.Error("error initializing mcp client", "error", err)
		return stdioTools
	}
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		slog.Error("error listing tools", "error", err)
		return stdioTools
	}
	for _, t := range tools.Tools {
		stdioTools = append(stdioTools, newMcpTool(name, t, m))
	}
	defer c.Close()
	return stdioTools
}

// GetMcpTools connects to MCP servers and returns available tools
func GetMcpTools(
	ctx context.Context,
	servers map[string]MCPServer,
) []BaseTool {
	if len(mcpTools) > 0 {
		return mcpTools
	}
	for name, m := range servers {
		switch m.Type {
		case MCPStdio:
			c, err := client.NewStdioMCPClient(
				m.Command,
				m.Env,
				m.Args...,
			)
			if err != nil {
				slog.Error("error creating mcp client", "error", err)
				continue
			}

			mcpTools = append(mcpTools, getTools(ctx, name, m, c)...)
		case MCPSse:
			c, err := client.NewSSEMCPClient(
				m.URL,
				client.WithHeaders(m.Headers),
			)
			if err != nil {
				slog.Error("error creating mcp client", "error", err)
				continue
			}
			mcpTools = append(mcpTools, getTools(ctx, name, m, c)...)
		}
	}

	return mcpTools
}
