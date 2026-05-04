# Embeddings

Each native embedding vendor is its own Go module under `embeddings/`.

## Text embeddings

OpenAI:

```go
import (
    "github.com/joakimcarlsson/ai/embeddings"
    embopenai "github.com/joakimcarlsson/ai/embeddings/openai"
    "github.com/joakimcarlsson/ai/model"
)

embedder := embopenai.NewEmbedding(
    embopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    embopenai.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
)

resp, err := embedder.GenerateEmbeddings(ctx, []string{
    "Hello, world",
    "How are you?",
})
fmt.Printf("Generated %d embeddings\n", len(resp.Embeddings))
```

Voyage:

```go
import embvoyage "github.com/joakimcarlsson/ai/embeddings/voyage"

embedder := embvoyage.NewEmbedding(
    embvoyage.WithAPIKey(os.Getenv("VOYAGE_API_KEY")),
    embvoyage.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embvoyage.WithInputType("document"),  // or "query"
)
```

Cohere:

```go
import embcohere "github.com/joakimcarlsson/ai/embeddings/cohere"

embedder := embcohere.NewEmbedding(
    embcohere.WithAPIKey(os.Getenv("COHERE_API_KEY")),
    embcohere.WithModel(model.CohereEmbeddingModels[model.EmbedV3]),
    embcohere.WithInputType("search_document"),
)
```

Gemini, Mistral, Bedrock follow the same shape — `embeddings/gemini`,
`embeddings/mistral`, `embeddings/bedrock`.

## Multimodal embeddings (Voyage)

```go
inputs := []embeddings.MultimodalInput{
    {
        Content: []embeddings.MultimodalContent{
            {Type: "text", Text: "A photo of a cat"},
            {Type: "image_url", ImageURL: "https://example.com/cat.jpg"},
        },
    },
}

resp, err := embedder.GenerateMultimodalEmbeddings(ctx, inputs)
```

## Contextualized embeddings (Voyage)

For document chunks where each chunk should be embedded with awareness of
its surrounding context:

```go
documents := [][]string{
    {"chunk 1 of doc 1", "chunk 2 of doc 1", "chunk 3 of doc 1"},
    {"chunk 1 of doc 2", "chunk 2 of doc 2"},
}

resp, err := embedder.GenerateContextualizedEmbeddings(ctx, documents)
// resp.DocumentEmbeddings[i][j] is the embedding of doc i, chunk j
```

## Per-call input type

The optional `inputType` variadic argument overrides the constructor default:

```go
resp, err := embedder.GenerateEmbeddings(ctx, texts, "query")
```

## Bedrock (Titan + Cohere on Bedrock)

```go
import embbedrock "github.com/joakimcarlsson/ai/embeddings/bedrock"

embedder := embbedrock.NewEmbedding(
    embbedrock.WithModel(model.BedrockEmbeddingModels[model.TitanEmbedV2]),
    embbedrock.WithRegion("us-east-1"),
    // or embbedrock.WithProfile("my-aws-profile"),
)
```

The vendor is detected from the model's API ID (`cohere.*` vs everything-else
defaults to Titan).
