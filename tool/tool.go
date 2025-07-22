package tool

import (
	"context"
	"encoding/json"
)

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Required    []string       `json:"required"`
}

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

type ToolResponseType string

const (
	ToolResponseTypeText  ToolResponseType = "text"
	ToolResponseTypeImage ToolResponseType = "image"
)

type ToolResponse struct {
	Type     ToolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
}

// NewTextResponse creates a successful text response
func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: false,
	}
}

// NewTextErrorResponse creates an error text response
func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}

// NewImageResponse creates a successful image response
func NewImageResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeImage,
		Content: content,
		IsError: false,
	}
}

// WithResponseMetadata adds JSON metadata to a tool response
func WithResponseMetadata(response ToolResponse, metadata any) ToolResponse {
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return response
		}
		response.Metadata = string(metadataBytes)
	}
	return response
}

type Registry struct {
	tools map[string]BaseTool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]BaseTool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool BaseTool) {
	r.tools[tool.Info().Name] = tool
}

// Get retrieves a tool by name from the registry
func (r *Registry) Get(name string) (BaseTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *Registry) List() []BaseTool {
	tools := make([]BaseTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Names returns all tool names in the registry
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool by name with the provided parameters
func (r *Registry) Execute(ctx context.Context, call ToolCall) (ToolResponse, error) {
	tool, exists := r.tools[call.Name]
	if !exists {
		return NewTextErrorResponse("tool not found: " + call.Name), nil
	}

	return tool.Run(ctx, call)
}
