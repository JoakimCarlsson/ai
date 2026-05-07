# Document Reranking

The `rerankers` modality. Vendors under `rerankers/`.

## Voyage

```go
import (
    "github.com/joakimcarlsson/ai/model"
    rrvoyage "github.com/joakimcarlsson/ai/rerankers/voyage"
)

reranker := rrvoyage.NewReranker(
    rrvoyage.WithAPIKey(os.Getenv("VOYAGE_API_KEY")),
    rrvoyage.WithModel(model.VoyageRerankerModels[model.Rerank25Lite]),
    rrvoyage.WithTopK(5),
    rrvoyage.WithReturnDocuments(true),
)

query := "What is machine learning?"
documents := []string{
    "Machine learning is a subset of artificial intelligence.",
    "The weather today is sunny.",
    "Deep learning uses neural networks.",
}

resp, err := reranker.Rerank(ctx, query, documents)
for i, r := range resp.Results {
    fmt.Printf("Rank %d (score=%.4f): %s\n", i+1, r.RelevanceScore, r.Document)
}
```

## Cohere

```go
import rrcohere "github.com/joakimcarlsson/ai/rerankers/cohere"

reranker := rrcohere.NewReranker(
    rrcohere.WithAPIKey(os.Getenv("COHERE_API_KEY")),
    rrcohere.WithModel(model.CohereRerankerModels[model.RerankV35]),
    rrcohere.WithTopK(5),
    rrcohere.WithReturnDocuments(true),
)

resp, err := reranker.Rerank(ctx, query, documents)
```

## Vendor-specific options

Voyage:

```go
rrvoyage.WithTruncation(true)
```

Cohere:

```go
rrcohere.WithMaxChunksPerDoc(8)
```
