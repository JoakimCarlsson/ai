// Example transcription_assemblyai demonstrates speech-to-text with AssemblyAI.
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
		model.ProviderAssemblyAI,
		transcription.WithAPIKey(
			os.Getenv("ASSEMBLYAI_API_KEY"),
		),
		transcription.WithModel(
			model.AssemblyAITranscriptionModels[model.AssemblyAIBest],
		),
		transcription.WithAssemblyAIOptions(
			transcription.WithAssemblyAISpeakerLabels(true),
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
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Duration: %.2fs\n", response.Duration)
	fmt.Printf("Words: %d\n", len(response.Words))
}
