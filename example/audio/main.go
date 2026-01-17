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

	basicExample(client)
	customVoiceExample(client)
	streamingExample(client)
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
		log.Fatal(err)
	}

	err = os.WriteFile("output_basic.mp3", response.AudioData, 0644)
	if err != nil {
		log.Fatal(err)
	}
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
		log.Fatal(err)
	}

	err = os.WriteFile("output_custom.mp3", response.AudioData, 0644)
	if err != nil {
		log.Fatal(err)
	}
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
		log.Fatal(err)
	}

	if _, err := exec.LookPath("ffplay"); err == nil {
		playStreamingAudio(chunkChan)
	} else {
		saveStreamingAudio(chunkChan)
	}
}

func playStreamingAudio(chunkChan <-chan audio.AudioChunk) {
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
			stdin.Close()
			cmd.Wait()
			log.Fatal(chunk.Error)
		}

		if chunk.Done {
			break
		}

		if len(chunk.Data) > 0 {
			if _, err := stdin.Write(chunk.Data); err != nil {
				stdin.Close()
				cmd.Wait()
				log.Fatal(err)
			}
		}
	}

	stdin.Close()
	cmd.Wait()
}

func saveStreamingAudio(chunkChan <-chan audio.AudioChunk) {
	file, err := os.Create("output_stream.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for chunk := range chunkChan {
		if chunk.Error != nil {
			log.Fatal(chunk.Error)
		}

		if chunk.Done {
			break
		}

		if len(chunk.Data) > 0 {
			if _, err := file.Write(chunk.Data); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func listVoicesExample(client audio.AudioGeneration) {
	voices, err := client.ListVoices(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for i, voice := range voices {
		if i >= 5 {
			break
		}
		fmt.Printf("%s (%s) - %s\n", voice.Name, voice.VoiceID, voice.Category)
	}
}
