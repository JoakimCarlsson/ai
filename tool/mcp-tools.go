package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
	toolRequest := mcp.CallToolRequest{}
	toolRequest.Params.Name = toolName
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
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
	c, err := pool.getClient(ctx, b.mcpName, b.mcpConfig)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}
	return runTool(ctx, c, b.tool.Name, params.Input)
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

func getTools(ctx context.Context, name string, m MCPServer) []BaseTool {
	var stdioTools []BaseTool
	c, err := pool.getClient(ctx, name, m)
	if err != nil {
		slog.Error("error getting mcp client", "error", err)
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
		mcpTools = append(mcpTools, getTools(ctx, name, m)...)
	}

	return mcpTools
}
