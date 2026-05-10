package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/tool"
)

const searchToolName = "search_knowledge_base"

const searchToolDescription = "Search the knowledge base for passages relevant to a question. Use this when the user asks about domain content the assistant might not know from training. Returns the top matches as a numbered list with source document IDs."

// SearchTool wraps a KnowledgeBase as a tool.BaseTool the LLM can call
// explicitly. Add it to an agent or voice agent via WithTools.
//
// Pair with WithKnowledgeBase: the option auto-injects retrieval
// context every turn, the tool lets the LLM dig deeper when the
// auto-injection is not enough.
func SearchTool(kb KnowledgeBase) tool.BaseTool {
	return &searchTool{kb: kb}
}

type searchTool struct {
	kb KnowledgeBase
}

type searchToolInput struct {
	Query string `json:"query"`
	K     int    `json:"k,omitempty"`
}

func (s *searchTool) Info() tool.Info {
	return tool.Info{
		Name:        searchToolName,
		Description: searchToolDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Question or search phrase to look up in the knowledge base.",
				},
				"k": map[string]any{
					"type":        "integer",
					"description": "Maximum number of passages to return (default 5).",
				},
			},
			"required": []string{"query"},
		},
		Required: []string{"query"},
	}
}

func (s *searchTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var in searchToolInput
	if params.Input != "" {
		if err := json.Unmarshal([]byte(params.Input), &in); err != nil {
			return tool.NewTextErrorResponse(
				fmt.Sprintf("invalid input: %v", err),
			), nil
		}
	}
	if strings.TrimSpace(in.Query) == "" {
		return tool.NewTextErrorResponse("query must not be empty"), nil
	}
	k := in.K
	if k <= 0 {
		k = 5
	}
	hits, err := s.kb.Retrieve(ctx, in.Query, k)
	if err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(formatHits(hits)), nil
}

// formatHits renders Hits as a numbered list suitable for an LLM tool
// response. Used by SearchTool and reused by agent/voice integrations
// to format auto-injected context with the same shape.
func formatHits(hits []Hit) string {
	if len(hits) == 0 {
		return "No relevant passages found."
	}
	var b strings.Builder
	for i, h := range hits {
		fmt.Fprintf(
			&b,
			"%d. [%s] %s\n",
			i+1,
			h.DocumentID,
			strings.TrimSpace(h.Content),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatHits renders Hits as a numbered list. Exported for use by
// callers that want to format retrieval results with the same shape
// SearchTool uses.
func FormatHits(hits []Hit) string { return formatHits(hits) }
