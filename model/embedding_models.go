package model

type EmbeddingModel struct {
	ID              ModelID       `json:"id"`
	Name            string        `json:"name"`
	Provider        ModelProvider `json:"provider"`
	APIModel        string        `json:"api_model"`
	CostPer1MTokens float64       `json:"cost_per_1m_tokens"`
	MaxInputTokens  int64         `json:"max_input_tokens"`
	EmbeddingDims   int           `json:"embedding_dimensions"`
}
