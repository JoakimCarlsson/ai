// Example transcription_deepgram demonstrates speech-to-text with Deepgram.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/transcription"
)

func main() {
	client, err := transcription.NewSpeechToText(
		model.ProviderDeepgram,
		transcription.WithAPIKey(
			os.Getenv("DEEPGRAM_API_KEY"),
		),
		transcription.WithModel(
			model.DeepgramTranscriptionModels[model.DeepgramNova3],
		),
		transcription.WithDeepgramOptions(
			transcription.WithDeepgramPunctuate(true),
			transcription.WithDeepgramSmartFormat(true),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	audioData, err := os.ReadFile("audio.mp3")
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Transcribe(
		context.Background(),
		audioData,
		transcription.WithLanguage("en"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Duration: %.2fs\n", response.Duration)
	fmt.Printf("Words: %d\n", len(response.Words))
}
