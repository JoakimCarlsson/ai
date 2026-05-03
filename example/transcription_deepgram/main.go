// Example transcription_deepgram demonstrates speech-to-text with Deepgram.
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
		model.ProviderDeepgram,
		stt.WithAPIKey(
			os.Getenv("DEEPGRAM_API_KEY"),
		),
		stt.WithModel(
			model.DeepgramTranscriptionModels[model.DeepgramNova3],
		),
		stt.WithDeepgramOptions(
			stt.WithDeepgramPunctuate(true),
			stt.WithDeepgramSmartFormat(true),
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
		stt.WithLanguage("en"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Duration: %.2fs\n", response.Duration)
	fmt.Printf("Words: %d\n", len(response.Words))
}
