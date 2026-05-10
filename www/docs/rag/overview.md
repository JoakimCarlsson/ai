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

## Stores

### In-memory (`rag/store/memory`)

```go
import ragmem "github.com/joakimcarlsson/ai/rag/store/memory"

store := ragmem.New()
```

Slice + RWMutex with cosine-similarity scoring. Brute-force linear scan over every chunk on every query, no index. Suitable for examples, tests, and small-scale prototypes; data is lost when the process exits, so every restart re-embeds the corpus from scratch. Fine up to a few thousand chunks; falls over above ~100k.

### PostgreSQL + pgvector (`rag/store/pgvector`)

Persistent store backed by a Postgres database with the [pgvector](https://github.com/pgvector/pgvector) extension. Schema is owned by versioned migrations embedded in the package; the runtime path doesn't issue DDL.

```go
import pgstore "github.com/joakimcarlsson/ai/rag/store/pgvector"

// Once at deploy time, as a privileged Postgres role:
if err := pgstore.Migrate(ctx, adminDSN, 1536); err != nil {
    log.Fatal(err)
}

// At application start, as a low-privilege role:
store, err := pgstore.New(ctx, runtimeDSN, 1536,
    pgstore.WithTable("rag_chunks"), // optional, default is "rag_chunks"
)
```

`Migrate` is idempotent and concurrent-safe (an advisory lock serialises racing migrators so they don't collide on `CREATE EXTENSION`). `New` does not run DDL; it reads the `rag_pgvector_migrations` ledger and refuses to start if the schema version is older than the build expects, or if the embedding column type does not match the configured `dims`.

#### Privileges

| Step | Required Postgres privileges |
|---|---|
| `Migrate` | `CREATE EXTENSION` (first run only — superuser or trusted-extension owner), plus `CREATE` on the schema for the chunks table, indexes, and ledger |
| `New` + runtime | `SELECT, INSERT, UPDATE, DELETE` on the chunks table; `SELECT` on the `rag_pgvector_migrations` ledger. No `CREATE`, no superuser |

The two-role pattern (admin owns DDL, app owns DML) follows the AWS Bedrock-on-Aurora reference and is the standard production posture.

#### Schema

Migration `001_initial.sql` creates:

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE rag_chunks (
    id           TEXT PRIMARY KEY,
    kb_id        TEXT NOT NULL,
    document_id  TEXT NOT NULL,
    content      TEXT NOT NULL,
    chunk_index  INT  NOT NULL,
    metadata     JSONB,
    model        TEXT,
    embedding    vector(<dims>),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX rag_chunks_kb_idx   ON rag_chunks (kb_id);
CREATE INDEX rag_chunks_hnsw_idx ON rag_chunks USING hnsw (embedding vector_cosine_ops);
```

Plus a bookkeeping table created by the migrator:

```sql
CREATE TABLE rag_pgvector_migrations (
    version    INT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The `model` column on `rag_chunks` carries `EmbeddedChunk.Model` so you can detect drift if you switch embedders.

#### Plugging into your own migrator

If your application already runs `golang-migrate`, `goose`, `atlas`, or another migration tool, skip `Migrate` and feed the embedded SQL files into your existing pipeline:

```go
fsys := pgstore.MigrationsFS()  // fs.FS containing 001_initial.sql, ...
```

Files contain Go-template placeholders (`{{.Table}}`, `{{.Dims}}`) that need substituting before they execute; render them with `text/template` before handing them to your migrator.

#### Dimension changes

pgvector stores dimensionality in the column type as `vector(N)`. Switching embedders to a different `N` is a destructive migration: `New` refuses to start when the configured `dims` does not match the stored column type, with a message saying a re-embed is required. The migration system can ship the mechanics (add sibling column, swap, drop), but the user has to provide the embedder for the backfill — only the application knows which model to switch to.

#### Local dev

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg18
    environment: { POSTGRES_USER: rag, POSTGRES_PASSWORD: rag, POSTGRES_DB: rag }
    ports: ["5433:5432"]
    volumes: [rag_pgdata:/var/lib/postgresql]
volumes: { rag_pgdata: }
```

### Combining with `memory/pgvector`

Memory and the knowledge base are independent stores. If both are configured, they each query their own table on every `Chat` and embed the user message separately, so you'll see two embedding API calls per turn. Schemas don't collide (`memories` table vs `rag_chunks` table), and you can share a single Postgres database.

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

- `examples/agent/rag/` — CLI eval harness: golden retrieval set, LLM-as-judge for faithfulness, retrieval recall + MRR over an in-memory store
- `examples/agent/rag-pgvector/` — same pipeline backed by Postgres + pgvector via docker-compose; demonstrates persistence and content-hash skip on re-runs
- `examples/voice/rag/` — web/voice version of the same pattern with an in-browser UI
