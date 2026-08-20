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
// api.json it mirrors. Every catalog in the repository is here, and each one is
// written by exactly one entry.
var targets = []target{
	chat("anthropic", "llm/anthropic", "anthropic"),
	chat("openai", "llm/openai", "openai"),
	chat("google", "llm/gemini", "gemini"),
	prefixed(chat("vertexai", "llm/vertexai", "vertexai"), "vertexai."),
	prefixed(chat("azure", "llm/azure", "azure"), "azure."),
	full(chat("bedrock", "llm/bedrock", "bedrock")),
	full(chat("groq", "llm/groq", "groq")),
	chat("mistral", "llm/mistral", "mistral"),
	chat("deepseek", "llm/deepseek", "deepseek"),
	chat("xai", "llm/xai", "xai"),
	prefixed(chat("cerebras", "llm/cerebras", "cerebras"), "cerebras."),
	prefixed(chat("fireworks", "llm/fireworks", "fireworks"), "fireworks."),
	full(prefixed(chat("together", "llm/together", "together"), "together.")),
	chat("perplexity", "llm/perplexity", "perplexity"),
	full(prefixed(chat("ollama", "llm/ollama", "ollama"), "ollama.")),
	full(bergetChat()),
	openRouter(prefixed(
		chat("openrouter", "llm/openrouter", "openrouter"),
		"openrouter.",
	)),

	image("openai", "image/openai", "openai"),
	image("google", "image/gemini", "gemini"),
	image("xai", "image/xai", "xai"),
	prefixed(image("azure", "image/azure", "azure"), "azure."),
	openRouter(prefixed(
		image("openrouter", "image/openrouter", "openrouter"),
		"openrouter.",
	)),

	embedding("openai", "embeddings/openai", "openai"),
	embedding("google", "embeddings/gemini", "gemini"),
	embedding("cohere", "embeddings/cohere", "cohere"),
	embedding("mistral", "embeddings/mistral", "mistral"),
	embedding("voyage", "embeddings/voyage", "voyage"),
	full(embedding("bedrock", "embeddings/bedrock", "bedrock")),
	full(embedding("berget", "embeddings/berget", "berget")),

	rerank("cohere", "rerankers/cohere", "cohere"),
	rerank("voyage", "rerankers/voyage", "voyage"),
	full(rerank("berget", "rerankers/berget", "berget")),

	transcription("openai", "stt/openai", "openai"),
	billed(
		transcription("googlecloud", "stt/google", "google"),
		"google-cloud",
	),
	transcription("assemblyai", "stt/assemblyai", "assemblyai"),
	transcription("deepgram", "stt/deepgram", "deepgram"),
	transcription("elevenlabs", "stt/elevenlabs", "elevenlabs"),
	billed(transcription("azure", "stt/azure", "azure"), "azure-speech"),
	full(transcription("berget", "stt/berget", "berget")),
	openRouter(prefixed(
		transcription("openrouter", "stt/openrouter", "openrouter"),
		"openrouter.",
	)),

	speech("openai", "tts/openai", "openai"),
	speech("elevenlabs", "tts/elevenlabs", "elevenlabs"),
	billed(speech("google", "tts/google", "google"), "google-cloud"),
	billed(speech("azure", "tts/azure", "azure"), "azure-speech"),
	speech("deepgram", "tts/deepgram", "deepgram"),
	openRouter(prefixed(
		speech("openrouter", "tts/openrouter", "openrouter"),
		"openrouter.",
	)),
}

// target is one generated models.go file.
type target struct {
	// source and kind name the slice of api.json this catalog mirrors.
	source     string
	kind       kind
	path       string
	pkg        string
	importPath string
	typeExpr   string
	// provider is the value written to every entry's Provider field, which is
	// not always the package name: one package can front a service the rest of
	// the library knows under another name.
	provider string
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

func chat(source, dir, pkg string) target {
	return newTarget(source, kindChat, dir, pkg, llmImport, "llm.Model",
		chatFields, doc(
			"Rates are per 1M tokens, in the currency the provider bills in.",
			"Context windows and output limits are taken from the same entry.",
		))
}

func image(source, dir, pkg string) target {
	return newTarget(source, kindImage, dir, pkg, imageImport,
		"image.GenerationModel", imageFields, doc(
			"Pricing is per image, in the currency the provider bills in. Where",
			"the source publishes a single rate per model, an entry's size and",
			"quality table is written only when the model is new to the catalog",
			"and is carried over from then on.",
		))
}

func speech(source, dir, pkg string) target {
	return newTarget(source, kindSpeech, dir, pkg, ttsImport, "tts.AudioModel",
		speechFields, doc(
			"CostPer1MChars is per 1M characters, in the currency the provider",
			"bills in. Default format and latency are not published and are",
			"carried over from the previous catalog.",
		))
}

func transcription(source, dir, pkg string) target {
	return newTarget(source, kindTranscription, dir, pkg, sttImport,
		"stt.TranscriptionModel", transcriptionFields, doc(
			"CostPer1MIn and CostPer1MOut are per 1M tokens where the model is",
			"priced per token, and per audio minute where it is priced by",
			"duration, matching the convention the hand-written catalogs use.",
		))
}

func embedding(source, dir, pkg string) target {
	return newTarget(source, kindEmbedding, dir, pkg, embeddingsImport,
		"embeddings.EmbeddingModel", embeddingFields, doc(
			"CostPer1MTokens is per 1M input tokens, in the currency the",
			"provider bills in.",
		))
}

func rerank(source, dir, pkg string) target {
	return newTarget(source, kindRerank, dir, pkg, rerankersImport,
		"rerankers.RerankerModel", rerankFields, doc(
			"CostPer1MTokens is per 1M input tokens, in the currency the",
			"provider bills in.",
		))
}

func newTarget(
	source string,
	k kind,
	dir, pkg, importPath, typeExpr string,
	order, doc []string,
) target {
	return target{
		source:     source,
		kind:       k,
		path:       dir + "/models.go",
		pkg:        pkg,
		importPath: importPath,
		typeExpr:   typeExpr,
		provider:   pkg,
		order:      order,
		doc:        doc,
		defaults:   map[string]string{"Provider": quote(pkg)},
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

// billed names the service an entry is attributed to when the package name is
// not it, so a catalog keeps the Provider string the rest of the library and
// its users already know it by.
func billed(t target, provider string) target {
	t.provider = provider
	t.defaults["Provider"] = quote(provider)
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
