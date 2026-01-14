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
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}

	client, err := audio.NewAudioGeneration(
		model.ProviderElevenLabs,
		audio.WithAPIKey(apiKey),
		audio.WithModel(
			model.ElevenLabsAudioModels[model.ElevenTurboV2_5],
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Example 1: Basic text-to-speech generation")
	basicExample(client)

	fmt.Println("\nExample 2: Custom voice settings")
	customVoiceExample(client)

	fmt.Println("\nExample 3: Streaming audio")
	streamingExample(client)

	fmt.Println("\nExample 4: List available voices")
	listVoicesExample(client)
}

func basicExample(client audio.AudioGeneration) {
	text := "Hello! This is a demonstration of the ElevenLabs text-to-speech API integration."

	response, err := client.GenerateAudio(
		context.Background(),
		text,
		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
	)
	if err != nil {
		log.Printf("Error generating audio: %v", err)
		return
	}

	err = os.WriteFile("output_basic.mp3", response.AudioData, 0644)
	if err != nil {
		log.Printf("Error writing file: %v", err)
		return
	}

	fmt.Printf("Generated audio saved to output_basic.mp3\n")
	fmt.Printf("Characters used: %d\n", response.Usage.Characters)
	fmt.Printf("Model: %s\n", response.Model)
}

func customVoiceExample(client audio.AudioGeneration) {
	text := "This audio uses custom voice settings for enhanced expressiveness and stability."

	response, err := client.GenerateAudio(
		context.Background(),
		text,
		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
		audio.WithStability(0.75),
		audio.WithSimilarityBoost(0.85),
		audio.WithStyle(0.5),
		audio.WithSpeakerBoost(true),
	)
	if err != nil {
		log.Printf("Error generating audio: %v", err)
		return
	}

	err = os.WriteFile("output_custom.mp3", response.AudioData, 0644)
	if err != nil {
		log.Printf("Error writing file: %v", err)
		return
	}

	fmt.Printf("Generated audio with custom settings saved to output_custom.mp3\n")
	fmt.Printf("Characters used: %d\n", response.Usage.Characters)
}

func streamingExample(client audio.AudioGeneration) {
	text := "This is a streaming audio example. The audio is generated and sent in chunks for real-time playback."

	chunkChan, err := client.StreamAudio(
		context.Background(),
		text,
		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
		audio.WithOptimizeStreamingLatency(3),
	)
	if err != nil {
		log.Printf("Error starting stream: %v", err)
		return
	}

	file, err := os.Create("output_stream.mp3")
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	chunkCount := 0
	totalBytes := 0

	for chunk := range chunkChan {
		if chunk.Error != nil {
			log.Printf("Stream error: %v", chunk.Error)
			return
		}

		if chunk.Done {
			fmt.Printf("Stream complete: received %d chunks, %d bytes total\n", chunkCount, totalBytes)
			break
		}

		if len(chunk.Data) > 0 {
			n, err := file.Write(chunk.Data)
			if err != nil {
				log.Printf("Error writing chunk: %v", err)
				return
			}
			chunkCount++
			totalBytes += n
		}
	}

	fmt.Printf("Streamed audio saved to output_stream.mp3\n")
}

func listVoicesExample(client audio.AudioGeneration) {
	voices, err := client.ListVoices(context.Background())
	if err != nil {
		log.Printf("Error listing voices: %v", err)
		return
	}

	fmt.Printf("Available voices: %d\n", len(voices))
	for i, voice := range voices {
		if i >= 5 {
			fmt.Printf("... and %d more voices\n", len(voices)-5)
			break
		}
		fmt.Printf("  - %s (%s) - %s\n", voice.Name, voice.VoiceID, voice.Category)
	}
}
