// Example transcription_assemblyai demonstrates speech-to-text with AssemblyAI.
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
		model.ProviderAssemblyAI,
		stt.WithAPIKey(
			os.Getenv("ASSEMBLYAI_API_KEY"),
		),
		stt.WithModel(
			model.AssemblyAITranscriptionModels[model.AssemblyAIBest],
		),
		stt.WithAssemblyAIOptions(
			stt.WithAssemblyAISpeakerLabels(true),
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
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Duration: %.2fs\n", response.Duration)
	fmt.Printf("Words: %d\n", len(response.Words))
}
