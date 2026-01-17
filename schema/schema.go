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
func NewStructuredOutputInfo(
	name, description string,
	parameters map[string]any,
	required []string,
) *StructuredOutputInfo {
	return &StructuredOutputInfo{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Required:    required,
	}
}

// NewStructuredOutputFromStruct creates a new structured output schema from a Go struct.
// It uses reflection to automatically generate the JSON schema from struct fields and tags.
//
// Supported struct tags:
//   - json: field name in JSON (e.g., `json:"field_name"`)
//   - desc: field description (e.g., `desc:"The field description"`)
//   - enum: comma-separated enum values (e.g., `enum:"value1,value2"`)
//   - required: explicitly mark as required or not (e.g., `required:"true"` or `required:"false"`)
//
// Example:
//
//	type Person struct {
//	    Name string `json:"name" desc:"Person's full name"`
//	    Age  int    `json:"age" desc:"Person's age in years"`
//	}
//	schema := NewStructuredOutputFromStruct("person", "A person object", Person{})
func NewStructuredOutputFromStruct(
	name, description string,
	structType any,
) *StructuredOutputInfo {
	parameters, required := GenerateSchema(structType)
	return &StructuredOutputInfo{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Required:    required,
	}
}
