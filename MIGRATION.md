# Migration Guide

This document covers three migrations. Each section is self-contained.

- **[v0.18.x → v0.19.0](#v018x--v0190--model-module-dissolved)** — `model` module removed; every catalog moved into the provider module that calls its API.
- **[v0.1.x → v0.2.0](#v01x--v020--memory-and-session-lifted)** — `memory` and `session` lifted out of `agent/` to top-level modules.
- **[v0.18.x → v0.1.0](#v018x--v010--multi-module-split)** — single Go module split into ~50 per-vendor modules.

If you're on v0.18.x and want the latest, apply all three in order: the multi-module split first, then the lift, then the model dissolve. The path tables in the split section already show the final post-lift destinations, so those two can be done in a single pass; the model dissolve is independent of both and can be applied on its own.

---

## v0.18.x → v0.19.0 — `model` module dissolved

The `model` module is removed. Its catalogs now live in the module that calls
the API, and its shared shapes live in the package that consumes them.

The reason is release cadence. Catalog data changes weekly, but it lived in a
module that 114 `go.mod` files depended on, so adding a single model forced a
release of `model` and then of every consumer. No shared catalog module is left,
so a model update now touches one module and needs one tag.

No values changed. All 513 entries across 44 catalogs are byte-for-byte
identical to the ones in the deleted package.

### Catalogs

Each catalog is now called `Models` and lives in the provider module that calls
that API:

| Old | New module | New reference |
|---|---|---|
| `model.AnthropicModels` | `llm/anthropic` | `anthropic.Models` |
| `model.AssemblyAITranscriptionModels` | `stt/assemblyai` | `assemblyai.Models` |
| `model.AzureModels` | `llm/azure` | `azure.Models` |
| `model.AzureSpeechAudioModels` | `tts/azure` | `azure.Models` |
| `model.AzureSpeechTranscriptionModels` | `stt/azure` | `azure.Models` |
| `model.BedrockEmbeddingModels` | `embeddings/bedrock` | `bedrock.Models` |
| `model.BergetEmbeddingModels` | `embeddings/berget` | `berget.Models` |
| `model.BergetModels` | `llm/berget` | `berget.Models` |
| `model.BergetRerankerModels` | `rerankers/berget` | `berget.Models` |
| `model.BergetTranscriptionModels` | `stt/berget` | `berget.Models` |
| `model.CerebrasModels` | `llm/cerebras` | `cerebras.Models` |
| `model.CohereEmbeddingModels` | `embeddings/cohere` | `cohere.Models` |
| `model.CohereRerankerModels` | `rerankers/cohere` | `cohere.Models` |
| `model.DeepSeekModels` | `llm/deepseek` | `deepseek.Models` |
| `model.DeepgramAudioModels` | `tts/deepgram` | `deepgram.Models` |
| `model.DeepgramTranscriptionModels` | `stt/deepgram` | `deepgram.Models` |
| `model.ElevenLabsAudioModels` | `tts/elevenlabs` | `elevenlabs.Models` |
| `model.ElevenLabsTranscriptionModels` | `stt/elevenlabs` | `elevenlabs.Models` |
| `model.FireworksModels` | `llm/fireworks` | `fireworks.Models` |
| `model.GeminiEmbeddingModels` | `embeddings/gemini` | `gemini.Models` |
| `model.GeminiImageGenerationModels` | `image/gemini` | `gemini.Models` |
| `model.GeminiModels` | `llm/gemini` | `gemini.Models` |
| `model.GoogleCloudAudioModels` | `tts/google` | `google.Models` |
| `model.GoogleCloudTranscriptionModels` | `stt/google` | `google.Models` |
| `model.GroqModels` | `llm/groq` | `groq.Models` |
| `model.MistralEmbeddingModels` | `embeddings/mistral` | `mistral.Models` |
| `model.MistralModels` | `llm/mistral` | `mistral.Models` |
| `model.OllamaModels` | `llm/ollama` | `ollama.Models` |
| `model.OpenAIAudioModels` | `tts/openai` | `openai.Models` |
| `model.OpenAIEmbeddingModels` | `embeddings/openai` | `openai.Models` |
| `model.OpenAIImageGenerationModels` | `image/openai` | `openai.Models` |
| `model.OpenAIModels` | `llm/openai` | `openai.Models` |
| `model.OpenAITranscriptionModels` | `stt/openai` | `openai.Models` |
| `model.OpenRouterAudioModels` | `tts/openrouter` | `openrouter.Models` |
| `model.OpenRouterImageGenerationModels` | `image/openrouter` | `openrouter.Models` |
| `model.OpenRouterModels` | `llm/openrouter` | `openrouter.Models` |
| `model.OpenRouterTranscriptionModels` | `stt/openrouter` | `openrouter.Models` |
| `model.PerplexityModels` | `llm/perplexity` | `perplexity.Models` |
| `model.TogetherModels` | `llm/together` | `together.Models` |
| `model.VertexAIGeminiModels` | `llm/vertexai` | `vertexai.Models` |
| `model.VoyageEmbeddingModels` | `embeddings/voyage` | `voyage.Models` |
| `model.VoyageRerankerModels` | `rerankers/voyage` | `voyage.Models` |
| `model.XAIImageGenerationModels` | `image/xai` | `xai.Models` |
| `model.XAIModels` | `llm/xai` | `xai.Models` |

### Constants

Model ID constants drop the provider-brand prefix, since the package already
supplies it. The ID string values are unchanged, so anything you have persisted
still matches.

```go
model.OpenAIModels[model.GPT4o]                 ->  openai.Models[openai.GPT4o]
model.OpenRouterModels[model.OpenRouterGPT41]   ->  openrouter.Models[openrouter.GPT41]
model.GeminiEmbeddingModels[model.GeminiEmbedding2]
                                                ->  gemini.Models[gemini.Embedding2]
```

Where stripping the prefix would not leave a valid identifier the name is kept
as it was, so `model.Gemini36Flash` becomes `gemini.Gemini36Flash`: `36Flash`
cannot start an identifier.

### Shapes

| Old | New |
|---|---|
| `model.Model` | `llm.Model` |
| `model.EmbeddingModel` | `embeddings.EmbeddingModel` |
| `model.RerankerModel` | `rerankers.RerankerModel` |
| `model.AudioModel` | `tts.AudioModel` |
| `model.TranscriptionModel` | `stt.TranscriptionModel` |
| `model.ImageGenerationModel` | `image.GenerationModel` |
| `model.Option` | `llm.ModelOption` |
| `model.NewCustomModel`, `model.With*` | `llm.NewCustomModel`, `llm.With*` |

Two renames are worth noting. `Option` became `ModelOption` so it does not read
as a client option beside `openai.Option`. `ImageGenerationModel` became
`GenerationModel` because `image.ImageGenerationModel` stutters.

### `ID` and `Provider` are plain strings

`model.ID` and `model.Provider` were named string types. They are now plain
`string`, and the `model.Provider*` constants are gone. Use the literal:

```go
Provider: model.ProviderOpenAI   ->   Provider: "openai"
```

This leaves the `types` module untouched. `types` is required by 64 modules, so
putting the shared vocabulary there would have forced exactly the cascade this
change exists to remove. As a side effect, `message` no longer depends on
`types` at all.

### Fill-in-the-middle takes its own model type

`fim` previously accepted the chat `model.Model`, so it compiled with any chat
model even though only a few support infilling. It now takes `fim.Model`, and
each FIM provider publishes the models that actually work:

```go
fimmistral.WithModel(mistral.Models[mistral.Codestral])
    ->  fimmistral.WithModel(fimmistral.Models[fimmistral.Codestral])

fimdeepseek.WithModel(deepseek.Models[deepseek.V32])
    ->  fimdeepseek.WithModel(fimdeepseek.Models[fimdeepseek.V32])
```

`fim.Model` carries only `ID`, `Name`, `Provider` and `APIModel`, which is
everything the FIM clients read. Pricing and context-window metadata stay on the
chat model configuration, which is a separate concern from selecting an
infilling endpoint.

### Removed

`model.CohereModels`, `model.MetaModels` and `model.QwenModels` are gone, along
with their ID constants. Nothing referenced them except pricing aliases inside
`model.OpenRouterModels`, and those are now literal values. There is no
`llm/cohere`, `llm/meta` or `llm/qwen` module because those are not directly
callable here: Llama and Qwen are reached through OpenRouter, Together,
Fireworks, Groq or Bedrock, each of which carries its own catalog.

---

## v0.1.x → v0.2.0 — `memory` and `session` lifted

`memory` and `session` were lifted out of `agent/` to become top-level modules.
Both are pure conversation primitives that any LLM consumer (agents, voice
runtimes, evaluation pipelines) can use independently — keeping them under
`agent/` implied an ownership that did not exist.

`agent/memory` was already its own module; it now lives at the top level.
`agent/session` was a sub-package of the `agent` module; it has been extracted
into its own module.

| Old import path (v0.1.x) | New import path (v0.2.0) |
|---|---|
| `github.com/joakimcarlsson/ai/agent/memory` | `github.com/joakimcarlsson/ai/memory` |
| `github.com/joakimcarlsson/ai/agent/memory/pgvector` | `github.com/joakimcarlsson/ai/memory/pgvector` |
| `github.com/joakimcarlsson/ai/agent/memory/postgres` | `github.com/joakimcarlsson/ai/memory/postgres` |
| `github.com/joakimcarlsson/ai/agent/memory/sqlite` | `github.com/joakimcarlsson/ai/memory/sqlite` |
| `github.com/joakimcarlsson/ai/agent/session` | `github.com/joakimcarlsson/ai/session` |

Type identities (`memory.Store`, `session.Session`, `session.MemoryStore`,
`session.FileStore`, etc.) and method signatures are unchanged. Only the
import paths move.

Mechanical migration:

```
sed -i 's|github.com/joakimcarlsson/ai/agent/memory|github.com/joakimcarlsson/ai/memory|g' **/*.go
sed -i 's|github.com/joakimcarlsson/ai/agent/session|github.com/joakimcarlsson/ai/session|g' **/*.go
```

Then update your `go.mod`:

- Drop `require github.com/joakimcarlsson/ai/agent/memory` (and any sub-module entries).
- Add `require github.com/joakimcarlsson/ai/memory v0.1.0` (plus `memory/pgvector`, `memory/postgres`, `memory/sqlite` as needed).
- If you were using `session` transitively via the `agent` module, you now need an explicit `require github.com/joakimcarlsson/ai/session v0.1.0`.
- Run `go mod tidy`.

---

## v0.18.x → v0.1.0 — multi-module split

The library was split from a single Go module into ~50 independent per-vendor
modules. Existing consumers on `v0.18.5` continue to work — the old monolithic
version stays published indefinitely. Migrating gives you per-vendor dependency
isolation: importing `llm/openai` no longer transitively pulls Anthropic SDK /
Google Genai / AWS SDK into your build.

The path-rename tables below show v0.2.0 destinations (post-lift); if you're
landing on v0.1.x specifically, swap `memory` paths back to `agent/memory` and
`session` back to `agent/session`.

### Why

Before the split, importing any sub-package of `github.com/joakimcarlsson/ai`
pulled every vendor SDK in the tree. Concrete example: `integrations/pgvector`
listed ~85 indirect dependencies because it transitively imported the root
`ai` module. After the split, the same backend ships ~13 transitive deps.
Each vendor implementation is its own module; you import only the SDKs you
actually use.

### Versioning

Every new module starts at **`v0.1.0`**. The leading zero signals the surface
may shift while the new layout settles. Modules will graduate to `v1.0.0` once
the API has been exercised by real consumers for a release cycle.

The integrations packages were at `integrations/{pgvector,postgres,sqlite}/v1.0.8`
in the old layout. Their new paths (`memory/{pgvector,postgres,sqlite}`)
are different Go module identities — the proxy sees them as new modules — and
restart at `v0.1.0` for consistency with the rest of the split.

### Path renames

#### Top-level package moves

| Old import path | New import path |
|---|---|
| `github.com/joakimcarlsson/ai/audio` | `github.com/joakimcarlsson/ai/tts` |
| `github.com/joakimcarlsson/ai/transcription` | `github.com/joakimcarlsson/ai/stt` |
| `github.com/joakimcarlsson/ai/image_generation` | `github.com/joakimcarlsson/ai/image` |
| `github.com/joakimcarlsson/ai/providers` | `github.com/joakimcarlsson/ai/llm` (interface only) |
| `github.com/joakimcarlsson/ai/integrations/pgvector` | `github.com/joakimcarlsson/ai/memory/pgvector` |
| `github.com/joakimcarlsson/ai/integrations/postgres` | `github.com/joakimcarlsson/ai/memory/postgres` |
| `github.com/joakimcarlsson/ai/integrations/sqlite` | `github.com/joakimcarlsson/ai/memory/sqlite` |

`agent/`, `batch/`, `embeddings/`, `fim/`, `message/`, `model/`, `prompt/`,
`rerankers/`, `schema/`, `tokens/`, `tracing/`, `types/`, `tool/` keep their
top-level names.

#### Vendor implementations: now in sub-modules

Old layout: vendors were files inside the modality package
(e.g. `audio/elevenlabs.go`, `providers/openai.go`). Constructors used a
factory pattern (`audio.NewGeneration(provider, opts...)`).

New layout: each vendor is its own sub-module with its own `go.mod`.

| Old (file in modality package) | New (sub-module) |
|---|---|
| `providers/anthropic.go` | `github.com/joakimcarlsson/ai/llm/anthropic` |
| `providers/openai.go` | `github.com/joakimcarlsson/ai/llm/openai` |
| `providers/azure.go` | `github.com/joakimcarlsson/ai/llm/azure` |
| `providers/bedrock.go` | `github.com/joakimcarlsson/ai/llm/bedrock` |
| `providers/gemini.go` | `github.com/joakimcarlsson/ai/llm/gemini` |
| `providers/vertexai.go` | `github.com/joakimcarlsson/ai/llm/vertexai` |
| `audio/openai.go` | `github.com/joakimcarlsson/ai/tts/openai` |
| `audio/elevenlabs.go` | `github.com/joakimcarlsson/ai/tts/elevenlabs` |
| `audio/google.go` | `github.com/joakimcarlsson/ai/tts/google` |
| `audio/azure.go` | `github.com/joakimcarlsson/ai/tts/azure` |
| (new) | `github.com/joakimcarlsson/ai/tts/deepgram` |
| `transcription/openai.go` | `github.com/joakimcarlsson/ai/stt/openai` |
| `transcription/elevenlabs.go` | `github.com/joakimcarlsson/ai/stt/elevenlabs` |
| `transcription/deepgram.go` | `github.com/joakimcarlsson/ai/stt/deepgram` |
| `transcription/assemblyai.go` | `github.com/joakimcarlsson/ai/stt/assemblyai` |
| `transcription/google.go` | `github.com/joakimcarlsson/ai/stt/google` |
| `embeddings/openai.go` | `github.com/joakimcarlsson/ai/embeddings/openai` |
| `embeddings/cohere.go` | `github.com/joakimcarlsson/ai/embeddings/cohere` |
| `embeddings/gemini.go` | `github.com/joakimcarlsson/ai/embeddings/gemini` |
| `embeddings/mistral.go` | `github.com/joakimcarlsson/ai/embeddings/mistral` |
| `embeddings/voyage.go` | `github.com/joakimcarlsson/ai/embeddings/voyage` |
| `embeddings/bedrock.go` | `github.com/joakimcarlsson/ai/embeddings/bedrock` |
| `image_generation/openai.go` | `github.com/joakimcarlsson/ai/image/openai` |
| `image_generation/gemini.go` | `github.com/joakimcarlsson/ai/image/gemini` |
| (new) | `github.com/joakimcarlsson/ai/image/xai` |
| `rerankers/cohere.go` | `github.com/joakimcarlsson/ai/rerankers/cohere` |
| `rerankers/voyage.go` | `github.com/joakimcarlsson/ai/rerankers/voyage` |
| `fim/deepseek.go` | `github.com/joakimcarlsson/ai/fim/deepseek` |
| `fim/mistral.go` | `github.com/joakimcarlsson/ai/fim/mistral` |
| `batch/anthropic.go` | `github.com/joakimcarlsson/ai/batch/anthropic` |
| `batch/openai.go` | `github.com/joakimcarlsson/ai/batch/openai` |
| `batch/gemini.go` | `github.com/joakimcarlsson/ai/batch/gemini` |
| `batch/concurrent.go` | `github.com/joakimcarlsson/ai/batch/concurrent` |

#### New OpenAI-compatible LLM wrapper modules

Vendors that speak OpenAI's chat-completions wire format used to require
calling `llm/openai` with `WithBaseURL("https://...")`. Each now ships as
its own thin module — same `Option` type, hardcoded base URL, no other deps.

| Module | Endpoint |
|---|---|
| `github.com/joakimcarlsson/ai/llm/xai` | `https://api.x.ai/v1` |
| `github.com/joakimcarlsson/ai/llm/openrouter` | `https://openrouter.ai/api/v1` |
| `github.com/joakimcarlsson/ai/llm/groq` | `https://api.groq.com/openai/v1` |
| `github.com/joakimcarlsson/ai/llm/deepseek` | `https://api.deepseek.com/v1` |
| `github.com/joakimcarlsson/ai/llm/perplexity` | `https://api.perplexity.ai` |
| `github.com/joakimcarlsson/ai/llm/mistral` | `https://api.mistral.ai/v1` |
| `github.com/joakimcarlsson/ai/llm/cerebras` | `https://api.cerebras.ai/v1` |
| `github.com/joakimcarlsson/ai/llm/fireworks` | `https://api.fireworks.ai/inference/v1` |
| `github.com/joakimcarlsson/ai/llm/together` | `https://api.together.xyz/v1` |
| `github.com/joakimcarlsson/ai/llm/ollama` | `http://localhost:11434/v1` |

### API changes

#### Factory functions removed

The old layout exposed factory functions that switched on a `Provider`
constant. These are gone — each vendor sub-module exports its own `New*`
constructor.

```go
// Before
client := providers.NewLLM("openai",
    providers.WithAPIKey(key),
    providers.WithModel(m),
)

// After
import openaillm "github.com/joakimcarlsson/ai/llm/openai"

client := openaillm.NewLLM(
    openaillm.WithAPIKey(key),
    openaillm.WithModel(m),
)
```

Same shape applies to TTS, STT, embeddings, image, rerankers, FIM. The
`provider` argument disappears; the import path identifies the vendor.

#### Image module — substantial redesign

The `image` modality changed shape more than the others. Three things to
update:

**1. Per-call options removed at the modality level.** Previously
`image.WithSize(...)`, `image.WithQuality(...)`, `image.WithResponseFormat(...)`,
`image.WithN(...)`, `image.WithAspectRatio(...)` were passed to
`GenerateImage` as variadic options. They no longer exist. Configure on the
vendor's construction `Options` instead:

```go
// Before
client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(k),
    imageopenai.WithModel(m),
)
resp, _ := client.GenerateImage(ctx, prompt,
    image.WithSize("1024x1024"),
    image.WithQuality("hd"),
)

// After
client := imageopenai.NewGeneration(
    imageopenai.WithAPIKey(k),
    imageopenai.WithModel(m),
    imageopenai.WithSize(imageopenai.Size1024x1024),
    imageopenai.WithQuality(imageopenai.QualityHigh),
)
resp, _ := client.GenerateImage(ctx, prompt)
```

**2. Typed enums replace bare strings.** Each vendor exports typed string
enums for closed value sets. Soft-typed (the underlying type is `string`),
so passing a value outside the enum still compiles for forward-compat:

| Vendor | Enum types |
|---|---|
| `image/openai` | `Size`, `Quality`, `Background`, `Moderation`, `OutputFormat` |
| `image/gemini` | `AspectRatio`, `ImageSize`, `OutputMIMEType` (plus genai SDK enums for `PersonGeneration`, `SafetyFilterLevel`, `ImagePromptLanguage`) |
| `image/xai` | `AspectRatio`, `Resolution`, `ResponseFormat` |

**3. Legacy OpenAI image models dropped from the registry.** `DALLE2`,
`DALLE3`, `GPTImage1`, `GPTImage1Mini` are removed from
`model.OpenAIImageGenerationModels`. Only `GPTImage15` and `GPTImage2` ship.
The corresponding code paths in `image/openai` (DALL-E 3 `style`, gpt-image-1
quirks) are also removed — `image/openai` targets gpt-image-1.5 and
gpt-image-2 only.

If you were on DALL-E 3, switch to `model.GPTImage15`; the call surface is
similar but quality presets are `low`/`medium`/`high` rather than
`standard`/`hd`.

#### Per-modality interface stays; vendor-construction is more verbose

Each modality interface (`llm.LLM`, `tts.Generation`, `stt.SpeechToText`,
`image.Generation`, `embeddings.Embedding`, `rerankers.Reranker`, `fim.FIM`)
keeps the same shape it had at v0.18.5, minus the factory functions. Code
written against those interfaces continues to work as long as you replace
the construction site with the vendor's own constructor.

### Mechanical migration

1. **Pin to a working state first.** In your go.mod, your existing
   `require github.com/joakimcarlsson/ai v0.18.5` resolves indefinitely;
   the migration can take its time.

2. **Grep your imports** for `github.com/joakimcarlsson/ai/`. Tally up every
   distinct import path.

3. **Map each path** using the tables above.

4. **Update import statements** in your source files. `goimports` /
   `gofmt` after.

5. **Rewrite construction sites** that used factory functions. Each becomes
   a vendor-specific `New*` call.

6. **For `image/` callers,** move per-call options up to construction and
   replace string literals with the typed enums. If you were on DALL-E 3,
   pick `GPTImage15` or `GPTImage2`.

7. **Update go.mod:** remove `require github.com/joakimcarlsson/ai vX.Y.Z`;
   add `require github.com/joakimcarlsson/ai/<module> v0.1.0` for every new
   path you actually import. Run `go mod tidy`.

8. **Compile.** Fix any remaining type drift.

9. **Run tests.**

### Staying on the monolith

If you're not ready to migrate, keep `require github.com/joakimcarlsson/ai v0.18.5`
in your go.mod. That tag is supported by the Go module proxy indefinitely.
No new features land there; bug fixes that affect the monolith may be
back-ported on a best-effort basis only.

## Reporting issues

Open an issue at https://github.com/joakimcarlsson/ai/issues with the tag
`migration` if a path or API isn't covered above, or if migration breaks in
a way the steps don't anticipate.
