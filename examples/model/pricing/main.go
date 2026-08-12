package main

import (
	"fmt"

	"github.com/joakimcarlsson/ai/llm"

	"github.com/joakimcarlsson/ai/embeddings/voyage"
	"github.com/joakimcarlsson/ai/image/gemini"
	"github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/tts/elevenlabs"
)

func main() {
	chatModel := openai.Models[openai.GPT54Nano]
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

	embeddingModel := voyage.Models[voyage.Voyage35Lite]
	embeddingTokens := int64(25_000)
	fmt.Printf("%s embedding estimate: $%.6f\n",
		embeddingModel.Name,
		estimatePerMillion(embeddingTokens, embeddingModel.CostPer1MTokens),
	)

	imageModel := gemini.Models[gemini.Imagen4Fast]
	fmt.Printf("%s image estimate: $%.6f\n",
		imageModel.Name,
		imageModel.Pricing[imageModel.DefaultSize][imageModel.DefaultQuality],
	)

	audioModel := elevenlabs.Models[elevenlabs.MultilingualV2]
	characters := int64(3_500)
	fmt.Printf("%s TTS estimate: $%.6f\n",
		audioModel.Name,
		estimatePerMillion(characters, audioModel.CostPer1MChars),
	)
}

func estimateChatCost(
	m llm.Model,
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
