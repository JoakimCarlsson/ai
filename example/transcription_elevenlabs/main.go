// Example transcription_elevenlabs demonstrates speech-to-text with ElevenLabs Scribe.
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
		model.ProviderElevenLabs,
		transcription.WithAPIKey(
			os.Getenv("ELEVENLABS_API_KEY"),
		),
		transcription.WithModel(
			model.ElevenLabsTranscriptionModels[model.ElevenLabsScribeV2],
		),
		transcription.WithElevenLabsSTTOptions(
			transcription.WithElevenLabsDiarize(true),
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
		transcription.WithLanguage("eng"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Language: %s\n", response.Language)
	fmt.Printf("Words: %d\n", len(response.Words))
}
