// Example transcription_deepgram_stream demonstrates streaming speech-to-text
// against Deepgram's WebSocket endpoint. It reads a raw PCM16-LE mono 16 kHz
// file (audio.pcm), splits it into 20 ms frames, and prints interim and final
// transcripts as they arrive.
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
	sampleRate = 16000
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
		model.ProviderDeepgram,
		transcription.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
		transcription.WithModel(model.DeepgramTranscriptionModels[model.DeepgramNova3]),
		transcription.WithDeepgramOptions(
			transcription.WithDeepgramStreamEndpointingMs(300),
		),
	)
	if err != nil {
		return err
	}
	if !client.SupportsStreaming() {
		return errors.New("deepgram client does not support streaming")
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
		transcription.WithLanguage("en-US"),
	)
	if err != nil {
		return err
	}

	go feedPCM(audio, pcm)

	for r := range results {
		if r.Error != nil {
			return r.Error
		}
		marker := "interim"
		if r.IsFinal {
			marker = "FINAL  "
		}
		fmt.Printf("[%s conf=%.2f] %s\n", marker, r.Confidence, r.Text)
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
