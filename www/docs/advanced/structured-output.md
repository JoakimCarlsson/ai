# Structured Output

Constrained generation that forces the LLM to return valid JSON matching a schema.

## Usage

```go
type CodeAnalysis struct {
    Language   string   `json:"language"`
    Functions  []string `json:"functions"`
    Complexity string   `json:"complexity"`
}

schema := &schema.StructuredOutputInfo{
    Name:        "code_analysis",
    Description: "Analyze code structure",
    Parameters: map[string]any{
        "language": map[string]any{
            "type":        "string",
            "description": "Programming language",
        },
        "functions": map[string]any{
            "type": "array",
            "items": map[string]any{"type": "string"},
            "description": "List of function names",
        },
        "complexity": map[string]any{
            "type": "string",
            "enum": []string{"low", "medium", "high"},
        },
    },
    Required: []string{"language", "functions", "complexity"},
}

response, err := client.SendMessagesWithStructuredOutput(ctx, messages, nil, schema)
if err != nil {
    log.Fatal(err)
}

var analysis CodeAnalysis
json.Unmarshal([]byte(*response.StructuredOutput), &analysis)
```

!!! note
    Structured output is supported by OpenAI, Gemini, Azure OpenAI, Vertex AI, Groq, OpenRouter, and xAI. Anthropic and AWS Bedrock do not currently support it.
