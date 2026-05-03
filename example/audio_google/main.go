// Example audio_google demonstrates text-to-speech with Google Cloud TTS.
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
		model.ProviderGoogleCloud,
		tts.WithAPIKey(
			os.Getenv("GOOGLE_CLOUD_API_KEY"),
		),
		tts.WithModel(
			model.GoogleCloudAudioModels[model.GoogleCloudTTSWavenet],
		),
		tts.WithGoogleCloudTTSOptions(
			tts.WithGoogleCloudLanguageCode("en-US"),
			tts.WithGoogleCloudVoiceName(
				"en-US-Wavenet-D",
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.GenerateAudio(
		context.Background(),
		"Hello! This is a test of the Google Cloud text-to-speech API.",
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
