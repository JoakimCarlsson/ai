package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPType string

const (
	MCPStdio          MCPType = "stdio"
	MCPSse            MCPType = "sse"
	MCPStreamableHTTP MCPType = "streamable_http"
)

type mcpTool struct {
	mcpName   string
	tool      *mcp.Tool
	mcpConfig MCPServer
}

type MCPClient interface {
	ListTools(
		ctx context.Context,
		params *mcp.ListToolsParams,
	) (*mcp.ListToolsResult, error)
	CallTool(
		ctx context.Context,
		params *mcp.CallToolParams,
	) (*mcp.CallToolResult, error)
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
	params := make(map[string]any)
	required := []string{}

	if b.tool.InputSchema != nil {
		if schemaMap, ok := b.tool.InputSchema.(map[string]any); ok {
			if props, ok := schemaMap["properties"].(map[string]any); ok {
				params = props
			}
			if req, ok := schemaMap["required"].([]any); ok {
				for _, r := range req {
					if str, ok := r.(string); ok {
						required = append(required, str)
					}
				}
			}
		}
	}

	return ToolInfo{
		Name:        fmt.Sprintf("%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  params,
		Required:    required,
	}
}

func runTool(
	ctx context.Context,
	c MCPClient,
	toolName string,
	input string,
) (ToolResponse, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return NewTextErrorResponse(
			fmt.Sprintf("error parsing parameters: %s", err),
		), nil
	}

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := c.CallTool(ctx, params)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	output := ""
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output += textContent.Text
		} else {
			output += fmt.Sprintf("%v", content)
		}
	}

	return NewTextResponse(output), nil
}

func (b *mcpTool) Run(
	ctx context.Context,
	params ToolCall,
) (ToolResponse, error) {
	c, err := pool.getClient(ctx, b.mcpName, b.mcpConfig)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}
	return runTool(ctx, c, b.tool.Name, params.Input)
}

func newMcpTool(
	name string,
	tool *mcp.Tool,
	mcpConfig MCPServer,
) BaseTool {
	return &mcpTool{
		mcpName:   name,
		tool:      tool,
		mcpConfig: mcpConfig,
	}
}

func getTools(ctx context.Context, name string, m MCPServer) ([]BaseTool, error) {
	var stdioTools []BaseTool
	c, err := pool.getClient(ctx, name, m)
	if err != nil {
		return nil, fmt.Errorf("error getting mcp client for %s: %w", name, err)
	}

	params := &mcp.ListToolsParams{}
	tools, err := c.ListTools(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error listing tools for %s: %w", name, err)
	}
	for _, t := range tools.Tools {
		stdioTools = append(stdioTools, newMcpTool(name, t, m))
	}
	return stdioTools, nil
}

// GetMcpTools connects to MCP servers and returns available tools
func GetMcpTools(
	ctx context.Context,
	servers map[string]MCPServer,
) ([]BaseTool, error) {
	var tools []BaseTool
	for name, m := range servers {
		serverTools, err := getTools(ctx, name, m)
		if err != nil {
			return nil, err
		}
		tools = append(tools, serverTools...)
	}

	return tools, nil
}
