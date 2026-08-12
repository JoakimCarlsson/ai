package rerankers

// RerankerModel represents a document reranking model with its configuration and capabilities.
//
// Each rerankers provider package publishes its own catalog of these under the
// name Models, so a configuration is selected as, for example,
// voyage.Models[voyage.Rerank25Lite].
type RerankerModel struct {
	// ID is the unique identifier for this reranker model.
	ID string `json:"id"`
	// Name is the human-readable name of the reranker model.
	Name string `json:"name"`
	// Provider identifies which AI service provides this model.
	Provider string `json:"provider"`
	// APIModel is the model identifier used in API requests.
	APIModel string `json:"api_model"`
	// CostPer1MTokens is the cost per 1 million tokens in USD.
	CostPer1MTokens float64 `json:"cost_per_1m_tokens"`
	// MaxQueryTokens is the maximum number of tokens allowed in the query.
	MaxQueryTokens int64 `json:"max_query_tokens"`
	// MaxTotalTokens is the maximum total tokens allowed across query and documents.
	MaxTotalTokens int64 `json:"max_total_tokens"`
}
