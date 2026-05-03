// Example transcription_google demonstrates speech-to-text with Google Cloud.
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
		model.ProviderGoogleCloud,
		stt.WithAPIKey(
			os.Getenv("GOOGLE_CLOUD_API_KEY"),
		),
		stt.WithModel(
			model.GoogleCloudTranscriptionModels[model.GoogleCloudSTTDefault],
		),
		stt.WithGoogleCloudSTTOptions(
			stt.WithGoogleCloudEncoding("LINEAR16"),
			stt.WithGoogleCloudSampleRate(16000),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	audioData, err := os.ReadFile("tts.wav")
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Transcribe(
		context.Background(),
		audioData,
		stt.WithLanguage("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcript: %s\n", response.Text)
	fmt.Printf("Words: %d\n", len(response.Words))
}
