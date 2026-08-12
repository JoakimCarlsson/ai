package fim

// Model represents a fill-in-the-middle model.
//
// This is deliberately not the chat [github.com/joakimcarlsson/ai/llm].Model.
// Fill-in-the-middle needs a model trained for infilling and served on a
// dedicated endpoint, so only a small subset of a provider's chat models work
// here. Taking a distinct type means WithModel cannot be handed a chat-only
// model that would fail at request time.
//
// The fields are the ones the FIM clients actually use to build a request.
// Pricing and context-window metadata live on the chat model configuration,
// which is a different concern from selecting an infilling endpoint.
//
// Each FIM provider package publishes its FIM-capable models under the name
// Models, so a configuration is selected as, for example,
// mistral.Models[mistral.Codestral].
type Model struct {
	// ID is the unique identifier for this model within the library.
	ID string `json:"id"`
	// Name is the human-readable name of the model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider string `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
}
