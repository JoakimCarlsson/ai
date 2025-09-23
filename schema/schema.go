// Package schema provides types and utilities for defining structured output schemas.
//
// This package defines the structures used to constrain AI model outputs to specific
// JSON schemas, enabling structured data generation from language models that support
// this feature.
//
// The main type is StructuredOutputInfo, which uses JSON Schema format to define
// the expected structure, types, and constraints for model outputs.
package schema

// StructuredOutputInfo defines a JSON schema for constraining AI model outputs.
// It specifies the structure, types, and requirements for generated JSON data.
type StructuredOutputInfo struct {
	// Name is the identifier for this structured output schema.
	Name string `json:"name"`
	// Description explains what this structured output represents.
	Description string `json:"description"`
	// Parameters defines the JSON schema properties using JSON Schema format.
	Parameters map[string]any `json:"parameters"`
	// Required lists the property names that must be present in the output.
	Required []string `json:"required"`
}

// NewStructuredOutputInfo creates a new structured output schema with the provided parameters.
func NewStructuredOutputInfo(name, description string, parameters map[string]any, required []string) *StructuredOutputInfo {
	return &StructuredOutputInfo{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Required:    required,
	}
}
