# Document Reranking

## Basic Usage

```go
import (
    "github.com/joakimcarlsson/ai/rerankers"
    "github.com/joakimcarlsson/ai/model"
)

reranker, err := rerankers.NewReranker(model.ProviderVoyage,
    rerankers.WithAPIKey(""),
    rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankers.WithReturnDocuments(true),
)
if err != nil {
    log.Fatal(err)
}

query := "What is machine learning?"
documents := []string{
    "Machine learning is a subset of artificial intelligence.",
    "The weather today is sunny.",
    "Deep learning uses neural networks.",
}

response, err := reranker.Rerank(context.Background(), query, documents)
if err != nil {
    log.Fatal(err)
}

for i, result := range response.Results {
    fmt.Printf("Rank %d (Score: %.4f): %s\n",
        i+1, result.RelevanceScore, result.Document)
}
```

## Client Options

```go
reranker, err := rerankers.NewReranker(
    model.ProviderVoyage,
    rerankers.WithAPIKey(""),
    rerankers.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rerankers.WithTopK(10),
    rerankers.WithReturnDocuments(true),
    rerankers.WithTruncation(true),
    rerankers.WithTimeout(30*time.Second),
)
```
