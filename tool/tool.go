// Package tool provides interfaces and utilities for implementing and managing AI model tools.
//
// This package defines the core interfaces for creating tools that can be called by AI models,
// along with utilities for tool registration, execution, and response handling. It supports
// both custom tools and MCP (Model Context Protocol) integration for advanced tooling scenarios.
//
// Tools are functions that AI models can call to perform specific tasks like web searches,
// calculations, file operations, or API calls. The package provides a standardized interface
// for tool definition and execution across all AI providers.
//
// Key components include:
//   - BaseTool interface for implementing custom tools
//   - ToolInfo for describing tool capabilities and parameters
//   - ToolResponse for structured tool execution results
//   - Registry for managing collections of tools
//   - MCP integration for external tool providers
//
// Example usage:
//
//	type WeatherTool struct{}
//
//	func (w *WeatherTool) Info() tool.ToolInfo {
//		return tool.ToolInfo{
//			Name:        "get_weather",
//			Description: "Get current weather for a location",
//			Parameters: map[string]any{
//				"location": map[string]any{
//					"type":        "string",
//					"description": "City name",
//				},
//			},
//			Required: []string{"location"},
//		}
//	}
//
//	func (w *WeatherTool) Run(ctx context.Context, params tool.ToolCall) (tool.ToolResponse, error) {
//		return tool.NewTextResponse("Sunny, 22°C"), nil
//	}
package tool

import (
	"context"
	"encoding/json"
)

// BaseTool defines the interface that all tools must implement.
// Tools provide functionality that AI models can invoke during conversations.
type BaseTool interface {
	// Info returns metadata about the tool including its name, description, and parameters.
	Info() ToolInfo
	// Run executes the tool with the provided parameters and returns a response.
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

// ToolInfo describes a tool's capabilities and parameters using JSON Schema format.
type ToolInfo struct {
	// Name is the unique identifier for the tool.
	Name string `json:"name"`
	// Description explains what the tool does and when to use it.
	Description string `json:"description"`
	// Parameters defines the tool's input schema using JSON Schema format.
	Parameters map[string]any `json:"parameters"`
	// Required lists the parameter names that must be provided.
	Required []string `json:"required"`
}

// ToolCall represents a request to execute a specific tool with parameters.
type ToolCall struct {
	// ID is a unique identifier for this tool call instance.
	ID string `json:"id"`
	// Name is the name of the tool to execute.
	Name string `json:"name"`
	// Input contains the JSON-encoded parameters for the tool.
	Input string `json:"input"`
}

// ToolResponseType indicates the format of a tool's response content.
type ToolResponseType string

const (
	// ToolResponseTypeText indicates the response contains plain text content.
	ToolResponseTypeText ToolResponseType = "text"
	// ToolResponseTypeImage indicates the response contains image content.
	ToolResponseTypeImage ToolResponseType = "image"
)

// ToolResponse represents the result of a tool execution.
type ToolResponse struct {
	// Type indicates the format of the response content.
	Type ToolResponseType `json:"type"`
	// Content contains the actual response data (text, base64 image, etc.).
	Content string `json:"content"`
	// Metadata contains optional JSON-encoded additional information.
	Metadata string `json:"metadata,omitempty"`
	// IsError indicates whether the tool execution resulted in an error.
	IsError bool `json:"is_error"`
}

// NewTextResponse creates a successful text response.
func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: false,
	}
}

// NewTextErrorResponse creates an error text response.
func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}

// NewImageResponse creates a successful image response.
func NewImageResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeImage,
		Content: content,
		IsError: false,
	}
}

// WithResponseMetadata adds JSON metadata to a tool response.
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

// Registry manages a collection of tools and provides methods for registration and execution.
type Registry struct {
	tools map[string]BaseTool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]BaseTool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool BaseTool) {
	r.tools[tool.Info().Name] = tool
}

// Get retrieves a tool by name from the registry.
func (r *Registry) Get(name string) (BaseTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools.
func (r *Registry) List() []BaseTool {
	tools := make([]BaseTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Names returns all tool names in the registry.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool by name with the provided parameters.
func (r *Registry) Execute(
	ctx context.Context,
	call ToolCall,
) (ToolResponse, error) {
	tool, exists := r.tools[call.Name]
	if !exists {
		return NewTextErrorResponse("tool not found: " + call.Name), nil
	}

	return tool.Run(ctx, call)
}
