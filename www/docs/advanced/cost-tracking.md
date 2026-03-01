# Cost Tracking

All models include built-in pricing information for cost calculation.

## LLM Models

```go
model := model.OpenAIModels[model.GPT4o]
fmt.Printf("Input cost: $%.2f per 1M tokens\n", model.CostPer1MIn)
fmt.Printf("Output cost: $%.2f per 1M tokens\n", model.CostPer1MOut)

response, err := client.SendMessages(ctx, messages, nil)
inputCost := float64(response.Usage.InputTokens) * model.CostPer1MIn / 1_000_000
outputCost := float64(response.Usage.OutputTokens) * model.CostPer1MOut / 1_000_000
```

## Image Generation Models

```go
model := model.OpenAIImageGenerationModels[model.DALLE3]

// Pricing structure: size -> quality -> cost
standardCost := model.Pricing["1024x1024"]["standard"]  // $0.04
hdCost := model.Pricing["1024x1024"]["hd"]              // $0.08

// GPT Image 1 with multiple quality tiers
gptImageModel := model.OpenAIImageGenerationModels[model.GPTImage1]
lowCost := gptImageModel.Pricing["1024x1024"]["low"]       // $0.011
mediumCost := gptImageModel.Pricing["1024x1024"]["medium"] // $0.042
highCost := gptImageModel.Pricing["1024x1024"]["high"]     // $0.167
```

## Audio Generation Models

```go
model := model.ElevenLabsAudioModels[model.ElevenTurboV2_5]

fmt.Printf("Cost per 1M chars: $%.2f\n", model.CostPer1MChars)
fmt.Printf("Max characters per request: %d\n", model.MaxCharacters)
fmt.Printf("Supports streaming: %v\n", model.SupportsStreaming)

response, err := client.GenerateAudio(ctx, text, audio.WithVoiceID("voice-id"))
cost := float64(response.Usage.Characters) * model.CostPer1MChars / 1_000_000
fmt.Printf("Cost: $%.4f\n", cost)
```
