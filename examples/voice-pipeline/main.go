// voice-pipeline wires AssemblyAI STT → OpenAI GPT-5.4 → ElevenLabs TTS into
// a slim end-to-end voice loop. It takes a PCM audio file (16kHz mono, 16-bit),
// streams it through AssemblyAI's Universal-Streaming WebSocket for live
// transcription, sends the final transcript to GPT-5.4 as a user message,
// and streams the LLM reply through ElevenLabs' WS TTS endpoint, writing
// the resulting audio to disk.
//
//	go run . input.pcm
//
// Required env: ASSEMBLYAI_API_KEY, OPENAI_API_KEY, ELEVENLABS_API_KEY.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
	sttassemblyai "github.com/joakimcarlsson/ai/stt/assemblyai"
	ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
)

const (
	pcmSampleRate = 16000
	pcmFrameMs    = 100
	pcmFrameBytes = pcmSampleRate * 2 * pcmFrameMs / 1000
	outputPath    = "pipeline-output.mp3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <input.pcm>", os.Args[0])
	}
	inputPath := os.Args[1]

	assemblyKey := os.Getenv("ASSEMBLYAI_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	elevenKey := os.Getenv("ELEVENLABS_API_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	transcript, err := transcribe(ctx, assemblyKey, inputPath)
	if err != nil {
		log.Fatalf("stt: %v", err)
	}
	if transcript == "" {
		log.Fatal("stt: empty transcript")
	}
	fmt.Printf("[stt] heard: %q\n", transcript)

	reply, err := chat(ctx, openaiKey, transcript)
	if err != nil {
		log.Fatalf("llm: %v", err)
	}
	if reply == "" {
		log.Fatal("llm: empty reply")
	}
	fmt.Printf("[llm] reply: %q\n", reply)

	if err := speak(ctx, elevenKey, reply, outputPath); err != nil {
		log.Fatalf("tts: %v", err)
	}
	fmt.Printf("[tts] saved %s\n", outputPath)
}

func transcribe(ctx context.Context, apiKey, pcmPath string) (string, error) {
	f, err := os.Open(pcmPath)
	if err != nil {
		return "", fmt.Errorf("open pcm: %w", err)
	}
	defer f.Close()

	client := sttassemblyai.NewSpeechToText(
		sttassemblyai.WithAPIKey(apiKey),
		sttassemblyai.WithModel(
			model.AssemblyAITranscriptionModels[model.AssemblyAIUniversalStreamingEnglish],
		),
	)

	audio := make(chan []byte, 8)
	results, err := client.StreamTranscribe(
		ctx,
		audio,
		stt.WithSampleRate(pcmSampleRate),
	)
	if err != nil {
		return "", fmt.Errorf("stream transcribe: %w", err)
	}

	go feedPCM(ctx, f, audio)

	var committed strings.Builder
	var latestPartial string
	for r := range results {
		if r.Error != nil {
			return "", r.Error
		}
		if r.Text == "" {
			continue
		}
		if r.IsFinal {
			if committed.Len() > 0 {
				committed.WriteByte(' ')
			}
			committed.WriteString(r.Text)
			latestPartial = ""
			continue
		}
		latestPartial = r.Text
	}
	if latestPartial != "" {
		if committed.Len() > 0 {
			committed.WriteByte(' ')
		}
		committed.WriteString(latestPartial)
	}
	return committed.String(), nil
}

func feedPCM(ctx context.Context, r io.Reader, audio chan<- []byte) {
	defer close(audio)
	frame := make([]byte, pcmFrameBytes)
	tick := time.NewTicker(pcmFrameMs * time.Millisecond)
	defer tick.Stop()
	for {
		n, err := io.ReadFull(r, frame)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, frame[:n])
			select {
			case audio <- chunk:
			case <-ctx.Done():
				return
			}
		}
		if err != nil {
			return
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			return
		}
	}
}

func chat(ctx context.Context, apiKey, prompt string) (string, error) {
	client := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54]),
		llmopenai.WithMaxTokens(256),
	)
	resp, err := client.SendMessages(ctx, []message.Message{
		message.NewUserMessage(prompt),
	}, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func speak(ctx context.Context, apiKey, text, outputPath string) error {
	client := ttselevenlabs.NewGeneration(
		ttselevenlabs.WithAPIKey(apiKey),
		ttselevenlabs.WithModel(
			model.ElevenLabsAudioModels[model.ElevenTurboV2_5],
		),
		ttselevenlabs.WithOutputFormat("mp3_44100_128"),
	)

	chunks, err := client.StreamAudio(ctx, text)
	if err != nil {
		return fmt.Errorf("stream audio: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	for c := range chunks {
		if c.Error != nil {
			return c.Error
		}
		if len(c.Data) > 0 {
			if _, err := out.Write(c.Data); err != nil {
				return err
			}
		}
		if c.Done {
			break
		}
	}
	return nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}
