package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/joakimcarlsson/ai/audio"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}

	ctx := context.Background()

	client, err := audio.NewAudioGeneration(
		model.ProviderElevenLabs,
		audio.WithAPIKey(apiKey),
		audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
	)
	if err != nil {
		log.Fatal(err)
	}

	text := "Hello, world! This is a test of streaming with alignment data."

	chunkChan, err := client.StreamAudio(ctx, text,
		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
		audio.WithAlignmentEnabled(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	for chunk := range chunkChan {
		if chunk.Error != nil {
			log.Fatalf("Stream error: %v", chunk.Error)
		}

		if chunk.Done {
			break
		}

		if len(chunk.Data) > 0 {
			if _, err := stdin.Write(chunk.Data); err != nil {
				log.Fatal(err)
			}

			if chunk.Alignment != nil {
				for _, char := range chunk.Alignment.Characters {
					fmt.Print(char)
				}
			}
		}
	}

	fmt.Println()

	stdin.Close()
	cmd.Wait()
}
