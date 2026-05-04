package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
	sttdeepgram "github.com/joakimcarlsson/ai/stt/deepgram"
)

func main() {
	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	if apiKey == "" {
		log.Fatal("DEEPGRAM_API_KEY is required")
	}
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <audio-file>", os.Args[0])
	}

	audioPath := os.Args[1]
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		log.Fatal(err)
	}

	client := sttdeepgram.NewSpeechToText(
		sttdeepgram.WithAPIKey(apiKey),
		sttdeepgram.WithModel(model.DeepgramTranscriptionModels[model.DeepgramNova3]),
		sttdeepgram.WithPunctuate(true),
		sttdeepgram.WithSmartFormat(true),
	)

	resp, err := client.Transcribe(
		context.Background(),
		audio,
		stt.WithLanguage("en"),
		stt.WithFilename(audioPath),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Text)
}
