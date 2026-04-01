// Example audio_openai demonstrates text-to-speech generation with OpenAI.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/audio"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, err := audio.NewAudioGeneration(
		model.ProviderOpenAI,
		audio.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		audio.WithModel(
			model.OpenAIAudioModels[model.OpenAITTS1],
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateAudio(
		context.Background(),
		"Hello! This is a test of the OpenAI text-to-speech API.",
		audio.WithVoiceID("nova"),
		audio.WithOutputFormat("mp3"),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("output.mp3", response.AudioData, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Generated %d bytes of audio\n",
		len(response.AudioData),
	)

	voices, err := client.ListVoices(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Available voices: %d\n", len(voices))
	for _, v := range voices {
		fmt.Printf("  - %s (%s)\n", v.Name, v.VoiceID)
	}
}
