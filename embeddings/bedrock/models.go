package bedrock

import (
	"github.com/joakimcarlsson/ai/embeddings"
)

// ProviderBedrock is the AWS Bedrock provider identifier.
const (
	TitanEmbedV2      string = "amazon.titan-embed-text-v2:0"
	TitanEmbedImageG1 string = "amazon.titan-embed-image-v1"
	CohereEmbedEn     string = "cohere.embed-english-v3"
	CohereEmbedMulti  string = "cohere.embed-multilingual-v3"
	CohereEmbedV4     string = "cohere.embed-v4:0"
)

// Models maps Bedrock embedding model IDs to their configurations.
//
// Pricing source: https://aws.amazon.com/bedrock/pricing/.
// Fetched: not re-verified in the 2026-07-26 sweep.
var Models = map[string]embeddings.EmbeddingModel{
	TitanEmbedV2: {
		ID:              TitanEmbedV2,
		Name:            "Amazon Titan Embed Text v2",
		Provider:        "bedrock",
		APIModel:        "amazon.titan-embed-text-v2:0",
		CostPer1MTokens: 0.02,
		MaxInputTokens:  8192,
		EmbeddingDims:   1024,
		SupportedDimensions: []int{
			256,
			384,
			512,
			1024,
		},
		MaxBatchSize: 1,
	},
	CohereEmbedEn: {
		ID:              CohereEmbedEn,
		Name:            "Cohere Embed English v3 (Bedrock)",
		Provider:        "bedrock",
		APIModel:        "cohere.embed-english-v3",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	CohereEmbedMulti: {
		ID:              CohereEmbedMulti,
		Name:            "Cohere Embed Multilingual v3 (Bedrock)",
		Provider:        "bedrock",
		APIModel:        "cohere.embed-multilingual-v3",
		CostPer1MTokens: 0.1,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	TitanEmbedImageG1: {
		ID:                  TitanEmbedImageG1,
		Name:                "Amazon Titan Multimodal Embeddings G1",
		Provider:            "bedrock",
		APIModel:            "amazon.titan-embed-image-v1",
		CostPer1MTokens:     0.8,
		MaxInputTokens:      256,
		EmbeddingDims:       1024,
		SupportedDimensions: []int{256, 384, 1024},
		MaxBatchSize:        1,
	},
	CohereEmbedV4: {
		ID:              CohereEmbedV4,
		Name:            "Cohere Embed v4 (Bedrock)",
		Provider:        "bedrock",
		APIModel:        "cohere.embed-v4:0",
		CostPer1MTokens: 0.12,
		MaxInputTokens:  128000,
		EmbeddingDims:   1536,
		SupportedDimensions: []int{
			256,
			512,
			1024,
			1536,
		},
		MaxBatchSize: 96,
	},
}
