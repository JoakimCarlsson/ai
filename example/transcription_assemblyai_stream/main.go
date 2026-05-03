// Example transcription_assemblyai_stream demonstrates streaming speech-to-text
// against AssemblyAI's v3 Universal Streaming WebSocket. It reads a raw
// PCM16-LE mono 16 kHz file (tts.pcm), splits it into 20 ms frames, and
// prints transcripts as they arrive (with end_of_turn marking finals).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
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
	client, err := stt.NewSpeechToText(
		model.ProviderAssemblyAI,
		stt.WithAPIKey(os.Getenv("ASSEMBLYAI_API_KEY")),
		stt.WithModel(model.AssemblyAITranscriptionModels[model.AssemblyAIBest]),
		stt.WithStreamSampleRate(sampleRate),
		stt.WithStreamChannels(channels),
		stt.WithAssemblyAIOptions(
			stt.WithAssemblyAIEndOfTurnSilenceMs(700),
		),
	)
	if err != nil {
		return err
	}
	if !client.SupportsStreaming() {
		return errors.New("assemblyAI client does not support streaming")
	}

	pcm, err := os.ReadFile("tts.pcm")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	audio := make(chan []byte, 64)
	results, err := client.StreamTranscribe(ctx, audio)
	if err != nil {
		return err
	}

	go feedPCM(audio, pcm)

	for r := range results {
		if r.Error != nil {
			return r.Error
		}
		marker := "turn   "
		if r.IsFinal {
			marker = "END    "
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
