// Example audio_alignment demonstrates ElevenLabs speech synthesis with character-level alignment data.
package main

import (
	"context"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}

	ctx := context.Background()

	client, err := tts.NewAudioGeneration(
		model.ProviderElevenLabs,
		tts.WithAPIKey(apiKey),
		tts.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
		tts.WithElevenLabsOptions(
			tts.WithElevenLabsVoiceID("EXAVITQu4vr4xnSDxMaL"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	text := "Hello, world! This is a test of alignment data."
	response, err := client.GenerateAudio(ctx, text,
		tts.WithAlignmentEnabled(true),
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
