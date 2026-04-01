// Example transcription_google demonstrates speech-to-text with Google Cloud.
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
		model.ProviderGoogleCloud,
		transcription.WithAPIKey(
			os.Getenv("GOOGLE_CLOUD_API_KEY"),
		),
		transcription.WithModel(
			model.GoogleCloudTranscriptionModels[model.GoogleCloudSTTDefault],
		),
		transcription.WithGoogleCloudSTTOptions(
			transcription.WithGoogleCloudEncoding("LINEAR16"),
			transcription.WithGoogleCloudSampleRate(16000),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	audioData, err := os.ReadFile("audio.wav")
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Transcribe(
		context.Background(),
		audioData,
		transcription.WithLanguage("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Words: %d\n", len(response.Words))
}
