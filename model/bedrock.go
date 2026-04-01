package model

// ProviderBedrock is the AWS Bedrock provider identifier.
const (
	ProviderBedrock Provider = "bedrock"

	BedrockTitanEmbedV2     ID = "amazon.titan-embed-text-v2:0"
	BedrockCohereEmbedEn    ID = "cohere.embed-english-v3"
	BedrockCohereEmbedMulti ID = "cohere.embed-multilingual-v3"
)

// BedrockEmbeddingModels maps Bedrock embedding model IDs to their configurations.
var BedrockEmbeddingModels = map[ID]EmbeddingModel{
	BedrockTitanEmbedV2: {
		ID:              BedrockTitanEmbedV2,
		Name:            "Amazon Titan Embed Text v2",
		Provider:        ProviderBedrock,
		APIModel:        "amazon.titan-embed-text-v2:0",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  8192,
		EmbeddingDims:   1024,
		SupportedDimensions: []int{
			256, 384, 512, 1024,
		},
		MaxBatchSize: 1,
	},
	BedrockCohereEmbedEn: {
		ID:              BedrockCohereEmbedEn,
		Name:            "Cohere Embed English v3 (Bedrock)",
		Provider:        ProviderBedrock,
		APIModel:        "cohere.embed-english-v3",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
	BedrockCohereEmbedMulti: {
		ID:              BedrockCohereEmbedMulti,
		Name:            "Cohere Embed Multilingual v3 (Bedrock)",
		Provider:        ProviderBedrock,
		APIModel:        "cohere.embed-multilingual-v3",
		CostPer1MTokens: 0.10,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
		MaxBatchSize:    96,
	},
}
