package main

import "strings"

// Import paths and catalog types, one per module a catalog can live in.
const (
	llmImport        = "github.com/joakimcarlsson/ai/llm"
	imageImport      = "github.com/joakimcarlsson/ai/image"
	ttsImport        = "github.com/joakimcarlsson/ai/tts"
	sttImport        = "github.com/joakimcarlsson/ai/stt"
	embeddingsImport = "github.com/joakimcarlsson/ai/embeddings"
	rerankersImport  = "github.com/joakimcarlsson/ai/rerankers"
)

// targets lists every generated models.go, bound to the provider and kind in
// api.json it mirrors. Each catalog is written by exactly one entry.
var targets = []target{
	chat("anthropic", "llm/anthropic", "anthropic"),
	chat("openai", "llm/openai", "openai"),
	chat("google", "llm/gemini", "gemini"),
	prefixed(chat("vertexai", "llm/vertexai", "vertexai"), "vertexai."),
	prefixed(chat("azure", "llm/azure", "azure"), "azure."),
	full(chat("groq", "llm/groq", "groq")),
	chat("mistral", "llm/mistral", "mistral"),
	chat("deepseek", "llm/deepseek", "deepseek"),
	chat("xai", "llm/xai", "xai"),
	prefixed(chat("cerebras", "llm/cerebras", "cerebras"), "cerebras."),
	prefixed(chat("fireworks", "llm/fireworks", "fireworks"), "fireworks."),
	full(prefixed(chat("together", "llm/together", "together"), "together.")),
	chat("perplexity", "llm/perplexity", "perplexity"),
	openRouter(prefixed(
		chat("openrouter", "llm/openrouter", "openrouter"),
		"openrouter.",
	)),
	openRouter(prefixed(
		image("openrouter", "image/openrouter", "openrouter"),
		"openrouter.",
	)),
	openRouter(prefixed(
		speech("openrouter", "tts/openrouter", "openrouter"),
		"openrouter.",
	)),
	openRouter(prefixed(
		transcription("openrouter", "stt/openrouter", "openrouter"),
		"openrouter.",
	)),
	full(bergetChat()),
	full(embedding("berget", "embeddings/berget", "berget")),
	full(rerank("berget", "rerankers/berget", "berget")),
	full(transcription("berget", "stt/berget", "berget")),
}

// target is one generated models.go file.
type target struct {
	// provider and kind name the slice of api.json this catalog mirrors.
	provider   string
	kind       kind
	path       string
	pkg        string
	importPath string
	typeExpr   string
	// idFull keeps the provider's whole model ID in the catalog ID instead of
	// dropping the vendor prefix.
	idFull   bool
	idPrefix string
	doc      []string
	order    []string
	defaults map[string]string
	// name adjusts a display name before it seeds a new entry.
	name func(string) string
}

func (t target) displayName(name string) string {
	if t.name == nil {
		return name
	}
	return t.name(name)
}

func chat(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindChat,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: llmImport,
		typeExpr:   "llm.Model",
		order:      chatFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"Rates are per 1M tokens, in the currency the provider bills in.",
			"Context windows and output limits are taken from the same entry.",
		),
	}
}

func image(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindImage,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: imageImport,
		typeExpr:   "image.GenerationModel",
		order:      imageFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"Pricing is per image, in the currency the provider bills in. The",
			"source publishes a single rate per model, so an entry's size and",
			"quality table is written only when the model is new to the catalog",
			"and is carried over from then on.",
		),
	}
}

func speech(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindSpeech,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: ttsImport,
		typeExpr:   "tts.AudioModel",
		order:      speechFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"CostPer1MChars is per 1M characters, in the currency the provider",
			"bills in. Default format and latency are not published and are",
			"carried over from the previous catalog.",
		),
	}
}

func transcription(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindTranscription,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: sttImport,
		typeExpr:   "stt.TranscriptionModel",
		order:      transcriptionFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"CostPer1MIn and CostPer1MOut are per 1M tokens where the model is",
			"priced per token, and per audio minute where it is priced by",
			"duration, matching the convention the hand-written catalogs use.",
		),
	}
}

func embedding(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindEmbedding,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: embeddingsImport,
		typeExpr:   "embeddings.EmbeddingModel",
		order:      embeddingFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"CostPer1MTokens is per 1M input tokens, in the currency the",
			"provider bills in.",
		),
	}
}

func rerank(provider, dir, pkg string) target {
	return target{
		provider:   provider,
		kind:       kindRerank,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: rerankersImport,
		typeExpr:   "rerankers.RerankerModel",
		order:      rerankFields,
		defaults:   map[string]string{"Provider": quote(pkg)},
		doc: doc(
			"CostPer1MTokens is per 1M input tokens, in the currency the",
			"provider bills in.",
		),
	}
}

// bergetChat keeps the window defaults the Berget catalog was built with: the
// source publishes no context window for Berget's models, and a catalog entry
// with a zero window is unusable.
func bergetChat() target {
	t := chat("berget", "llm/berget", "berget")
	t.defaults["ContextWindow"] = "131072"
	t.defaults["DefaultMaxTokens"] = "8192"
	t.doc = append(t.doc, "",
		"Berget publishes no context window, so a model new to this catalog",
		"defaults to 131072 tokens.",
	)
	return t
}

func prefixed(t target, prefix string) target {
	t.idPrefix = prefix
	return t
}

func full(t target) target {
	t.idFull = true
	return t
}

// openRouter marks a catalog OpenRouter routes to upstream providers for: it
// passes their rates through, and prefixes display names so a routed model
// reads apart from the same model served directly.
func openRouter(t target) target {
	t.name = func(name string) string {
		if _, rest, ok := strings.Cut(name, ": "); ok {
			name = rest
		}
		return "OpenRouter – " + name
	}
	t.doc = append(t.doc, "",
		"OpenRouter routes to upstream providers and passes their rates",
		"through; the figures here are the ones it quotes.",
	)
	return t
}

// doc completes a catalog's header with the two rules every generated catalog
// follows, so each one states them rather than assuming the reader knows.
func doc(lines ...string) []string {
	return append(lines, "",
		"A model the source stops listing is removed from this catalog.",
		"Display names and anything the source does not publish are set when a",
		"model is first added and carried over from then on.",
	)
}
