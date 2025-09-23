package model

// EmbeddingModel represents an embedding model with its configuration and capabilities.
type EmbeddingModel struct {
	// ID is the unique identifier for this embedding model.
	ID ModelID `json:"id"`
	// Name is the human-readable name of the embedding model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider ModelProvider `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// CostPer1MTokens is the cost per 1 million tokens in USD.
	CostPer1MTokens float64 `json:"cost_per_1m_tokens"`
	// MaxInputTokens is the maximum number of input tokens per request.
	MaxInputTokens int64 `json:"max_input_tokens"`
	// EmbeddingDims is the default dimensionality of the embedding vectors.
	EmbeddingDims int `json:"embedding_dimensions"`
	// SupportedDimensions lists alternative dimensions if the model supports them.
	SupportedDimensions []int `json:"supported_dimensions,omitempty"`
	// MaxBatchSize is the maximum number of inputs per batch request.
	MaxBatchSize int `json:"max_batch_size,omitempty"`
	// SupportsOutputDtype indicates if the model supports different output data types.
	SupportsOutputDtype bool `json:"supports_output_dtype,omitempty"`
	// MaxTokensPerBatch is the maximum total tokens allowed in a single batch.
	MaxTokensPerBatch int64 `json:"max_tokens_per_batch,omitempty"`
}

type RerankerModel struct {
	ID              ModelID       `json:"id"`
	Name            string        `json:"name"`
	Provider        ModelProvider `json:"provider"`
	APIModel        string        `json:"api_model"`
	CostPer1MTokens float64       `json:"cost_per_1m_tokens"`
	MaxQueryTokens  int64         `json:"max_query_tokens"`
	MaxTotalTokens  int64         `json:"max_total_tokens"`
}
