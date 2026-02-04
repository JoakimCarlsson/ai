package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/transcription"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	client, err := transcription.NewSpeechToText(
		model.ProviderOpenAI,
		transcription.WithAPIKey(apiKey),
		transcription.WithModel(model.OpenAITranscriptionModels[model.Whisper1]),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	audioURL := "https://sys.jdaddy.net/preview/rob.mp3"
	resp, err := http.Get(audioURL)
	if err != nil {
		fmt.Printf("Error downloading audio: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading audio: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	response, err := client.Transcribe(ctx, audioData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(response.Text)
}
