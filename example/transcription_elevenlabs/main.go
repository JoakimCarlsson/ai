// Example transcription_elevenlabs demonstrates speech-to-text with ElevenLabs Scribe.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
)

func main() {
	client, err := stt.NewSpeechToText(
		model.ProviderElevenLabs,
		stt.WithAPIKey(
			os.Getenv("ELEVENLABS_API_KEY"),
		),
		stt.WithModel(
			model.ElevenLabsTranscriptionModels[model.ElevenLabsScribeV2],
		),
		stt.WithElevenLabsSTTOptions(
			stt.WithElevenLabsDiarize(true),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	audioData, err := os.ReadFile("tts.mp3")
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Transcribe(
		context.Background(),
		audioData,
		stt.WithLanguage("eng"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Language: %s\n", response.Language)
	fmt.Printf("Words: %d\n", len(response.Words))
}
