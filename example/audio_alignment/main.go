package main

import (
	"context"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/audio"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := "sk_d3db1d738742edda91935c8a8ce919935e7efd26063d3915"
	if apiKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}

	ctx := context.Background()

	client, err := audio.NewAudioGeneration(
		model.ProviderElevenLabs,
		audio.WithAPIKey(apiKey),
		audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
	)
	if err != nil {
		log.Fatal(err)
	}

	text := "Hello, world! This is a test of alignment data."
	response, err := client.GenerateAudio(ctx, text,
		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
		audio.WithAlignmentEnabled(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("output_with_alignment.mp3", response.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	if response.Alignment != nil {
		for i, char := range response.Alignment.Characters {
			start := response.Alignment.CharacterStartTimesSeconds[i]
			end := response.Alignment.CharacterEndTimesSeconds[i]
			_ = char
			_ = start
			_ = end
		}
	}
}
