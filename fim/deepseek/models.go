package deepseek

import (
	"github.com/joakimcarlsson/ai/fim"
)

// DeepSeek fill-in-the-middle model IDs.
const (
	V32 string = "deepseek-v3.2"
)

// Models maps DeepSeek FIM model IDs to their configurations.
//
// DeepSeek serves fill-in-the-middle from its beta completions endpoint rather
// than the chat endpoint, and only the base V3.2 model is documented as
// supporting it. The reasoning variants are absent because infilling and
// chain-of-thought decoding are not offered together.
//
// Source: https://api-docs.deepseek.com/guides/fim_completion.
// Fetched: 2026-07-26.
var Models = map[string]fim.Model{
	V32: {
		ID:       V32,
		Name:     "DeepSeek V3.2",
		Provider: "deepseek",
		APIModel: "deepseek-v3.2",
	},
}
