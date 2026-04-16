# Embeddings

## Text Embeddings

```go
import (
    "github.com/joakimcarlsson/ai/embeddings"
    "github.com/joakimcarlsson/ai/model"
)

embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
)
if err != nil {
    log.Fatal(err)
}

texts := []string{
    "Hello, world!",
    "This is a test document.",
}

response, err := embedder.GenerateEmbeddings(context.Background(), texts)
if err != nil {
    log.Fatal(err)
}

for i, embedding := range response.Embeddings {
    fmt.Printf("Text: %s\n", texts[i])
    fmt.Printf("Dimensions: %d\n", len(embedding))
    fmt.Printf("First 5 values: %v\n", embedding[:5])
}
```

## Multimodal Embeddings

```go
embedder, err := embeddings.NewEmbedding(model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.VoyageMulti3]),
)

multimodalInputs := []embeddings.MultimodalInput{
    {
        Content: []embeddings.MultimodalContent{
            {Type: "text", Text: "This is a banana."},
            {Type: "image_url", ImageURL: "https://example.com/banana.jpg"},
        },
    },
}

response, err := embedder.GenerateMultimodalEmbeddings(context.Background(), multimodalInputs)
```

## Contextualized Embeddings

Embed document chunks with awareness of their surrounding context. Each chunk embedding incorporates information from the full document, improving retrieval for chunks that lack standalone meaning.

```go
documentChunks := [][]string{
    { // Document 1
        "Introduction to quantum computing...",
        "Qubits differ from classical bits...",
        "Quantum entanglement enables...",
    },
    { // Document 2
        "Machine learning overview...",
        "Neural networks consist of...",
    },
}

response, err := embedder.GenerateContextualizedEmbeddings(context.Background(), documentChunks)

// response.DocumentEmbeddings[0][1] = embedding for "Qubits differ..." with context from Document 1
```

## Client Options

```go
embedder, err := embeddings.NewEmbedding(
    model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embeddings.WithBatchSize(100),
    embeddings.WithDimensions(1024),
    embeddings.WithTimeout(30*time.Second),
    embeddings.WithVoyageOptions(
        embeddings.WithInputType("document"),
        embeddings.WithOutputDimension(1024),
        embeddings.WithOutputDtype("float"),
    ),
)
```

## Cohere

```go
embedder, err := embeddings.NewEmbedding(model.ProviderCohere,
    embeddings.WithAPIKey(os.Getenv("COHERE_API_KEY")),
    embeddings.WithModel(model.CohereEmbeddingModels[model.CohereEmbedEnV3]),
    embeddings.WithCohereOptions(
        embeddings.WithCohereInputType("search_document"),
    ),
)

response, err := embedder.GenerateEmbeddings(ctx, texts)
```

### Cohere Options

| Option | Description |
|--------|-------------|
| `WithCohereInputType(string)` | Input type: `"search_document"`, `"search_query"` |
| `WithCohereTruncation(string)` | Truncation strategy: `"NONE"`, `"START"`, `"END"` |
| `WithCohereEmbeddingTypes([]string)` | Types: `"float"`, `"int8"`, `"uint8"`, `"binary"`, `"ubinary"` |

**Models:** `CohereEmbedV4` (1024 dims, 128K tokens), `CohereEmbedMultiV3` (1024 dims), `CohereEmbedEnV3` (1024 dims)

## Google Gemini

```go
embedder, err := embeddings.NewEmbedding(model.ProviderGemini,
    embeddings.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    embeddings.WithModel(model.GeminiEmbeddingModels[model.GeminiTextEmbedding004]),
    embeddings.WithGeminiOptions(
        embeddings.WithGeminiTaskType("RETRIEVAL_DOCUMENT"),
    ),
)

response, err := embedder.GenerateEmbeddings(ctx, texts)
```

### Gemini Options

| Option | Description |
|--------|-------------|
| `WithGeminiTaskType(string)` | Task type: `"RETRIEVAL_DOCUMENT"`, `"RETRIEVAL_QUERY"` |

**Models:** `GeminiTextEmbedding004` (768 dims, supports 768/512/256)

## Mistral

```go
embedder, err := embeddings.NewEmbedding(model.ProviderMistral,
    embeddings.WithAPIKey(os.Getenv("MISTRAL_API_KEY")),
    embeddings.WithModel(model.MistralEmbeddingModels[model.MistralEmbed]),
)

response, err := embedder.GenerateEmbeddings(ctx, texts)
```

### Mistral Options

| Option | Description |
|--------|-------------|
| `WithMistralOutputDimension(int)` | Output dimensionality (for Codestral Embed) |
| `WithMistralOutputDtype(string)` | Data type: `"float"`, `"int8"`, `"uint8"`, `"binary"`, `"ubinary"` |

**Models:** `MistralEmbed` (1024 dims, 8K tokens), `CodestralEmbed` (1536 dims, supports 1536/1024/768/512/256, 32K tokens)

## AWS Bedrock

```go
embedder, err := embeddings.NewEmbedding(model.ProviderBedrock,
    embeddings.WithModel(model.BedrockEmbeddingModels[model.BedrockTitanEmbedV2]),
    embeddings.WithBedrockOptions(
        embeddings.WithBedrockRegion("us-east-1"),
    ),
)

response, err := embedder.GenerateEmbeddings(ctx, texts)
```

### Bedrock Options

| Option | Description |
|--------|-------------|
| `WithBedrockRegion(string)` | AWS region (default: `"us-east-1"`) |
| `WithBedrockProfile(string)` | AWS shared config profile for credentials |

**Models:** `BedrockTitanEmbedV2` (1024 dims, supports 256/384/512/1024), `BedrockCohereEmbedEn` (1024 dims), `BedrockCohereEmbedMulti` (1024 dims)

Bedrock uses AWS credentials from the environment or shared config — no API key required.

## Embedding Interface

```go
type Embedding interface {
    GenerateEmbeddings(ctx, texts, inputType...) (*EmbeddingResponse, error)
    GenerateMultimodalEmbeddings(ctx, inputs, inputType...) (*EmbeddingResponse, error)
    GenerateContextualizedEmbeddings(ctx, documentChunks, inputType...) (*ContextualizedEmbeddingResponse, error)
    Model() model.EmbeddingModel
}
```
