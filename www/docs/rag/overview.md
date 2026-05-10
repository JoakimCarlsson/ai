# Retrieval-Augmented Generation (RAG)

The `rag` package provides primitives for grounding agent and voice responses in a domain knowledge base: chunk documents, embed the chunks, store them in a vector store, retrieve relevant passages for a query, and inject them into the LLM call.

The shape mirrors `agent.WithMemory`: build a `KnowledgeBase`, attach it via `WithKnowledgeBase`, and the framework handles per-turn retrieval and context injection.

## Architecture

A `KnowledgeBase` composes three pieces:

- **Embedder** — any `embeddings.Embedding` implementation (OpenAI, Voyage, Cohere, Gemini, Bedrock, Mistral)
- **Chunker** — splits documents into retrievable units; `rag/chunkers/fixed` ships a token-aware fixed-size chunker
- **Store** — persists pre-embedded chunks and serves vector similarity queries; `rag/store/memory` is the default in-process implementation

Retrieval happens in two layers. The `Store` returns the top-k by cosine similarity (and ignores forward-compatibility `SearchOption`s in v1). The `KnowledgeBase` then applies orchestrator-level filters (`WithMinScore`, `WithMaxDistance`).

## Quick start

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    embedopenai "github.com/joakimcarlsson/ai/embeddings/openai"
    "github.com/joakimcarlsson/ai/model"
    "github.com/joakimcarlsson/ai/rag"
    "github.com/joakimcarlsson/ai/rag/chunkers/fixed"
    ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
)

embedder := embedopenai.NewEmbedding(
    embedopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    embedopenai.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
)

kb := rag.New("docs", embedder, ragmem.New(),
    rag.WithChunker(fixed.Default),
)

_ = kb.Ingest(ctx, []rag.Document{
    {ID: "policy-returns", Content: "Items can be returned within 30 days..."},
    {ID: "policy-shipping", Content: "Standard shipping takes 3 to 5 business days..."},
})

a := agent.New(llmClient,
    agent.WithSystemPrompt("Answer using the knowledge base. Cite the document ID."),
    agent.WithKnowledgeBase(kb),
    agent.WithTools(rag.SearchTool(kb)),
)

response, _ := a.Chat(ctx, "How do returns work?")
```

`WithKnowledgeBase` is the auto-injection path: every `Chat` retrieves the top-5 chunks for the user's message and prepends them as a "Relevant context from the knowledge base:" segment in the system prompt. `rag.SearchTool(kb)` is the on-demand path: the LLM can issue follow-up retrievals via the `search_knowledge_base` tool when the auto-injection is not enough.

## Public API

```go
type Document struct {
    ID       string
    Content  string
    Metadata map[string]any
}

type Chunk struct {
    ID, DocumentID, Content string
    Index                   int
    Metadata                map[string]any
}

type EmbeddedChunk struct {
    Chunk
    Embedding []float32
    Model     string  // APIModel id used at ingest time; for drift detection
}

type Hit struct {
    Chunk
    Score float64
}

type Chunker interface {
    Chunk(Document) []Chunk
}

type Store interface {
    Upsert(ctx, kbID, []EmbeddedChunk) error
    Search(ctx, kbID, []float32, k, ...SearchOption) ([]Hit, error)
    Delete(ctx, chunkID) error
}

type KnowledgeBase interface {
    ID() string
    Ingest(ctx, []Document) error
    Retrieve(ctx, query, k) ([]Hit, error)
}

func New(id string, e embeddings.Embedding, s Store, opts ...Option) KnowledgeBase
func SearchTool(kb KnowledgeBase) tool.BaseTool
func FormatHits(hits []Hit) string
```

## Constructor options

| Option | Description | Default |
|--------|-------------|---------|
| `WithChunker(c)` | Chunker implementation | rune-window 2048/256 |
| `WithMinScore(f)` | Drop hits below score `f` | disabled |
| `WithMaxDistance(d)` | Drop hits whose `(1-score) > d` | disabled |
| `WithIDGenerator(g)` | Override the chunk ID generator | UUIDv4 |

For token-precise chunking, import `rag/chunkers/fixed`:

```go
rag.WithChunker(fixed.Default)              // size=512, overlap=64
rag.WithChunker(fixed.New(1024, 128))       // custom
```

## SearchOption (forward compatibility)

`Store.Search` accepts variadic `SearchOption`s. v1 ships only the type; helpers (filters, score thresholds, namespaces) land alongside the stores that support them. Existing stores ignore unrecognised options, so adding new options is non-breaking.

```go
type SearchOption interface {
    applySearchOption(*searchConfig)  // private; only the rag package can implement
}
```

Store implementations call `rag.ApplySearchOptions(opts...)` once at the top of Search to drain options into a `SearchConfig`.

## Embedding model tracking

Each `EmbeddedChunk` carries the `APIModel` id of the embedder used at ingest time. Mismatched query/index encoders are a known silent failure mode in production RAG; the field lets a future re-embed pass detect when the active embedder differs from what the index was built with.

```go
chunks := /* fetched from store */
for _, c := range chunks {
    if c.Model != currentEmbedder.Model().APIModel {
        // re-embed needed
    }
}
```

## Built-in store

`rag/store/memory.New()` returns an in-process slice + RWMutex store with cosine-similarity scoring. Suitable for examples, tests, and small-scale prototypes; data is lost when the process exits. Use a dedicated `rag/store/*` implementation (e.g., pgvector when it lands) for persistence and scale.

## Voice

Voice agents wire the same way:

```go
import "github.com/joakimcarlsson/ai/voice"

agent := voice.New(llmClient, sttClient, ttsClient,
    voice.WithKnowledgeBase(kb),
    voice.WithTools(rag.SearchTool(kb)),
)
```

Recall fires on the first LLM iteration of each user turn and is cached on the per-turn `turnState` so subsequent tool-call iterations of the same turn do not re-search.

## Examples

- `examples/agent/rag/` — CLI: load markdown into a KB, ask a question, print a grounded answer
- `examples/voice/rag/` — web/voice version of the same pattern with an in-browser UI
