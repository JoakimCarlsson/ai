# Migration Guide — v0.18.x → v0.1.0 (multi-module split)

The library has been split from a single Go module into ~50 independent
per-vendor modules. Existing consumers on `v0.18.5` continue to work — the
old monolithic version stays published indefinitely. Migrating gives you
per-vendor dependency isolation: importing `llm/openai` no longer transitively
pulls Anthropic SDK / Google Genai / AWS SDK into your build.

This document covers everything a consumer needs to update.

## Why

Before the split, importing any sub-package of `github.com/joakimcarlsson/ai`
pulled every vendor SDK in the tree. Concrete example: `integrations/pgvector`
listed ~85 indirect dependencies because it transitively imported the root
`ai` module. After the split, the same backend ships ~13 transitive deps.
Each vendor implementation is its own module; you import only the SDKs you
actually use.

## Versioning

Every new module starts at **`v0.1.0`**. The leading zero signals the surface
may shift while the new layout settles. Modules will graduate to `v1.0.0` once
the API has been exercised by real consumers for a release cycle.

The integrations packages were at `integrations/{pgvector,postgres,sqlite}/v1.0.8`
in the old layout. Their new paths (`agent/memory/{pgvector,postgres,sqlite}`)
are different Go module identities — the proxy sees them as new modules — and
restart at `v0.1.0` for consistency with the rest of the split.

## Path renames

### Top-level package moves

| Old import path | New import path |
|---|---|
| `github.com/joakimcarlsson/ai/audio` | `github.com/joakimcarlsson/ai/tts` |
| `github.com/joakimcarlsson/ai/transcription` | `github.com/joakimcarlsson/ai/stt` |
| `github.com/joakimcarlsson/ai/image_generation` | `github.com/joakimcarlsson/ai/image` |
| `github.com/joakimcarlsson/ai/providers` | `github.com/joakimcarlsson/ai/llm` (interface only) |
| `github.com/joakimcarlsson/ai/integrations/pgvector` | `github.com/joakimcarlsson/ai/agent/memory/pgvector` |
| `github.com/joakimcarlsson/ai/integrations/postgres` | `github.com/joakimcarlsson/ai/agent/memory/postgres` |
| `github.com/joakimcarlsson/ai/integrations/sqlite` | `github.com/joakimcarlsson/ai/agent/memory/sqlite` |

`agent/`, `batch/`, `embeddings/`, `fim/`, `message/`, `model/`, `prompt/`,
`rerankers/`, `schema/`, `tokens/`, `tracing/`, `types/`, `tool/` keep their
top-level names.

### Vendor implementations: now in sub-modules

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

### New OpenAI-compatible LLM wrapper modules

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

## API changes

### Factory functions removed

The old layout exposed factory functions that switched on a `Provider`
constant. These are gone — each vendor sub-module exports its own `New*`
constructor.

```go
// Before
client := providers.NewLLM(model.ProviderOpenAI,
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

### Image module — substantial redesign

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

### Per-modality interface stays; vendor-construction is more verbose

Each modality interface (`llm.LLM`, `tts.Generation`, `stt.SpeechToText`,
`image.Generation`, `embeddings.Embedding`, `rerankers.Reranker`, `fim.FIM`)
keeps the same shape it had at v0.18.5, minus the factory functions. Code
written against those interfaces continues to work as long as you replace
the construction site with the vendor's own constructor.

## Mechanical migration

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

## Staying on the monolith

If you're not ready to migrate, keep `require github.com/joakimcarlsson/ai v0.18.5`
in your go.mod. That tag is supported by the Go module proxy indefinitely.
No new features land there; bug fixes that affect the monolith may be
back-ported on a best-effort basis only.

## Reporting issues

Open an issue at https://github.com/joakimcarlsson/ai/issues with the tag
`migration` if a path or API isn't covered above, or if migration breaks in
a way the steps don't anticipate.
