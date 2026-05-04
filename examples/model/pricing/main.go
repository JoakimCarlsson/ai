package main

import (
	"fmt"

	"github.com/joakimcarlsson/ai/model"
)

func main() {
	chatModel := model.OpenAIModels[model.GPT54Nano]
	inputTokens := int64(8_000)
	outputTokens := int64(1_200)
	cachedInputTokens := int64(2_000)

	fmt.Printf(
		"%s chat estimate: $%.6f\n",
		chatModel.Name,
		estimateChatCost(
			chatModel,
			inputTokens,
			outputTokens,
			cachedInputTokens,
		),
	)

	embeddingModel := model.VoyageEmbeddingModels[model.Voyage35Lite]
	embeddingTokens := int64(25_000)
	fmt.Printf("%s embedding estimate: $%.6f\n",
		embeddingModel.Name,
		estimatePerMillion(embeddingTokens, embeddingModel.CostPer1MTokens),
	)

	imageModel := model.GeminiImageGenerationModels[model.Imagen4Fast]
	fmt.Printf("%s image estimate: $%.6f\n",
		imageModel.Name,
		imageModel.Pricing[imageModel.DefaultSize][imageModel.DefaultQuality],
	)

	audioModel := model.ElevenLabsAudioModels[model.ElevenMultilingualV2]
	characters := int64(3_500)
	fmt.Printf("%s TTS estimate: $%.6f\n",
		audioModel.Name,
		estimatePerMillion(characters, audioModel.CostPer1MChars),
	)
}

func estimateChatCost(
	m model.Model,
	inputTokens int64,
	outputTokens int64,
	cachedInputTokens int64,
) float64 {
	uncachedInputTokens := inputTokens - cachedInputTokens
	if uncachedInputTokens < 0 {
		uncachedInputTokens = 0
	}

	return estimatePerMillion(uncachedInputTokens, m.CostPer1MIn) +
		estimatePerMillion(cachedInputTokens, m.CostPer1MInCached) +
		estimatePerMillion(outputTokens, m.CostPer1MOut)
}

func estimatePerMillion(units int64, costPerMillion float64) float64 {
	return float64(units) / 1_000_000 * costPerMillion
}
