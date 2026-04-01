// Example audio_azure demonstrates text-to-speech with Azure Speech Services.
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
		model.ProviderAzureSpeech,
		audio.WithAPIKey(
			os.Getenv("AZURE_SPEECH_KEY"),
		),
		audio.WithModel(
			model.AzureSpeechAudioModels[model.AzureSpeechNeural],
		),
		audio.WithAzureSpeechOptions(
			audio.WithAzureRegion("eastus"),
			audio.WithAzureVoiceName(
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
