package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/prompt"
)

//go:embed prompts/*.md
var promptFiles embed.FS

func main() {
	cache := prompt.NewCache()

	systemPrompt, err := renderPrompt(
		cache,
		"prompts/system.md",
		map[string]any{
			"tone":       "Friendly",
			"audience":   "Go developers using the prompt package",
			"style":      "Keep the output short and concrete.",
			"priorities": []string{"clarity", "examples", "maintainability"},
			"forbidden": []string{
				"provider-specific advice",
				"long disclaimers",
			},
		},
		"tone",
		"audience",
		"priorities",
	)
	if err != nil {
		log.Fatal(err)
	}

	userPrompt, err := renderPrompt(cache, "prompts/task.md", map[string]any{
		"format": "short implementation note",
		"topic":  "embedding markdown prompt templates in Go",
		"context": "The templates live next to the example as .md files.\n" +
			"go:embed ships them with the binary.\n" +
			"The prompt package validates inputs and renders them.",
		"requirements": []string{
			"mention go:embed",
			"mention required variables",
			"mention cached parsed templates",
		},
	}, "format", "topic", "context", "requirements")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Rendered system.md ===")
	fmt.Println(systemPrompt)
	fmt.Println("=== Rendered task.md ===")
	fmt.Println(userPrompt)
}

func renderPrompt(
	cache *prompt.Cache,
	path string,
	data map[string]any,
	required ...string,
) (string, error) {
	source, err := promptFiles.ReadFile(path)
	if err != nil {
		return "", err
	}

	tmpl, err := prompt.New(
		string(source),
		prompt.WithName(path),
		prompt.WithCache(cache),
		prompt.WithRequired(required...),
		prompt.WithStrictMode(),
	)
	if err != nil {
		return "", err
	}

	return tmpl.Process(data)
}
