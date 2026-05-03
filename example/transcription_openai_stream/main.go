// Example transcription_openai_stream demonstrates streaming speech-to-text
// against OpenAI's Realtime API in transcription mode. It reads a raw
// PCM16-LE mono 24 kHz file (audio.pcm — Realtime requires 24 kHz), splits
// it into 20 ms frames, and prints accumulated delta and completed events.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/transcription"
)

const (
	sampleRate = 24000
	channels   = 1
	frameMs    = 20
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	client, err := transcription.NewSpeechToText(
		model.ProviderOpenAI,
		transcription.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		transcription.WithModel(model.OpenAITranscriptionModels[model.GPT4oTranscribe]),
	)
	if err != nil {
		return err
	}
	if !client.SupportsStreaming() {
		return errors.New("openai client does not support streaming")
	}

	pcm, err := os.ReadFile("audio.pcm")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	audio := make(chan []byte, 64)
	results, err := client.StreamTranscribe(ctx, audio,
		transcription.WithStreamSampleRate(sampleRate),
		transcription.WithStreamChannels(channels),
		transcription.WithOpenAIRealtimeVADSilenceMs(500),
	)
	if err != nil {
		return err
	}

	go feedPCM(audio, pcm)

	for r := range results {
		if r.Error != nil {
			return r.Error
		}
		marker := "delta  "
		if r.IsFinal {
			marker = "DONE   "
		}
		fmt.Printf("[%s] %s\n", marker, r.Text)
	}
	return nil
}

func feedPCM(audio chan<- []byte, pcm []byte) {
	defer close(audio)
	frameBytes := sampleRate * channels * 2 * frameMs / 1000
	tick := time.NewTicker(frameMs * time.Millisecond)
	defer tick.Stop()
	for off := 0; off < len(pcm); off += frameBytes {
		end := min(off+frameBytes, len(pcm))
		frame := make([]byte, end-off)
		copy(frame, pcm[off:end])
		audio <- frame
		<-tick.C
	}
}
