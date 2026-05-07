package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
	ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
)

func main() {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		log.Fatal("ELEVENLABS_API_KEY is required")
	}

	client := ttselevenlabs.NewGeneration(
		ttselevenlabs.WithAPIKey(apiKey),
		ttselevenlabs.WithModel(
			model.ElevenLabsAudioModels[model.ElevenMultilingualV2],
		),
		ttselevenlabs.WithOutputFormat("mp3_44100_128"),
	)

	resp, err := client.GenerateAudio(
		context.Background(),
		"Hello from the ElevenLabs text to speech example.",
		tts.WithStability(0.5),
		tts.WithSimilarityBoost(0.75),
	)
	if err != nil {
		log.Fatal(err)
	}

	const output = "elevenlabs-speech.mp3"
	if err := os.WriteFile(output, resp.AudioData, 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"saved %s (%s, %d characters)\n",
		output,
		resp.ContentType,
		resp.Usage.Characters,
	)
}
