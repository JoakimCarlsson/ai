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

## Client Options

```go
embedder, err := embeddings.NewEmbedding(
    model.ProviderVoyage,
    embeddings.WithAPIKey(""),
    embeddings.WithModel(model.VoyageEmbeddingModels[model.Voyage35]),
    embeddings.WithBatchSize(100),
    embeddings.WithTimeout(30*time.Second),
    embeddings.WithVoyageOptions(
        embeddings.WithInputType("document"),
        embeddings.WithOutputDimension(1024),
        embeddings.WithOutputDtype("float"),
    ),
)
```
