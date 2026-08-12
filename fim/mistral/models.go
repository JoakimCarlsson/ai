package mistral

import (
	"github.com/joakimcarlsson/ai/fim"
)

// Mistral fill-in-the-middle model IDs.
const (
	Codestral string = "codestral"
)

// Models maps Mistral FIM model IDs to their configurations.
//
// This is the subset of Mistral's catalogue served on the fill-in-the-middle
// endpoint, which is why it is much smaller than the chat catalogue in
// llm/mistral. Codestral is the code model Mistral trains for infilling; the
// general chat models do not accept a suffix and are deliberately absent.
//
// Source: https://docs.mistral.ai/capabilities/code_generation/.
// Fetched: 2026-07-26.
var Models = map[string]fim.Model{
	Codestral: {
		ID:       Codestral,
		Name:     "Codestral",
		Provider: "mistral",
		APIModel: "codestral-2508",
	},
}
