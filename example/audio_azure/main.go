// Example audio_azure demonstrates text-to-speech with Azure Speech Services.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	client, err := tts.NewAudioGeneration(
		model.ProviderAzureSpeech,
		tts.WithAPIKey(
			os.Getenv("AZURE_SPEECH_KEY"),
		),
		tts.WithModel(
			model.AzureSpeechAudioModels[model.AzureSpeechNeural],
		),
		tts.WithAzureSpeechOptions(
			tts.WithAzureRegion("eastus"),
			tts.WithAzureVoiceName(
				"en-US-JennyNeural",
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateAudio(
		context.Background(),
		"Hello! This is a test of the Azure Speech Services API.",
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
}
