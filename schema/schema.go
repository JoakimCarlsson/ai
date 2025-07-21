package schema

type StructuredOutputInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Required    []string       `json:"required"`
}

func NewStructuredOutputInfo(name, description string, parameters map[string]any, required []string) *StructuredOutputInfo {
	return &StructuredOutputInfo{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Required:    required,
	}
}