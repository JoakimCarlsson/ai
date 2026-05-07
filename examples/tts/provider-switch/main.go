package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
	ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
	ttsopenai "github.com/joakimcarlsson/ai/tts/openai"
)

func main() {
	client, provider := newTTS()

	resp, err := client.GenerateAudio(
		context.Background(),
		"Switching text to speech providers only changes construction code.",
	)
	if err != nil {
		log.Fatal(err)
	}

	output := provider + "-speech.mp3"
	if err := os.WriteFile(output, resp.AudioData, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[%s] saved %s (%s)\n", provider, output, resp.ContentType)
}

func newTTS() (tts.Generation, string) {
	switch provider := providerName(); provider {
	case "elevenlabs":
		return ttselevenlabs.NewGeneration(
			ttselevenlabs.WithAPIKey(requiredEnv("ELEVENLABS_API_KEY")),
			ttselevenlabs.WithModel(
				model.ElevenLabsAudioModels[model.ElevenMultilingualV2],
			),
			ttselevenlabs.WithOutputFormat("mp3_44100_128"),
		), provider
	case "openai":
		return ttsopenai.NewGeneration(
			ttsopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			ttsopenai.WithModel(model.OpenAIAudioModels[model.OpenAIMiniTTS]),
			ttsopenai.WithVoice("alloy"),
			ttsopenai.WithOutputFormat("mp3"),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai or elevenlabs)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
	}
	return provider
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
