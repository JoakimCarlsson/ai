package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
	sttazure "github.com/joakimcarlsson/ai/stt/azure"
	sttdeepgram "github.com/joakimcarlsson/ai/stt/deepgram"
	sttopenai "github.com/joakimcarlsson/ai/stt/openai"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <audio-file>", os.Args[0])
	}

	audioPath := os.Args[1]
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		log.Fatal(err)
	}

	client, provider := newSTT()
	resp, err := client.Transcribe(
		context.Background(),
		audio,
		stt.WithLanguage("en"),
		stt.WithFilename(audioPath),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("[%s] %s\n", provider, resp.Text)
}

func newSTT() (stt.SpeechToText, string) {
	switch provider := providerName(); provider {
	case "deepgram":
		return sttdeepgram.NewSpeechToText(
			sttdeepgram.WithAPIKey(requiredEnv("DEEPGRAM_API_KEY")),
			sttdeepgram.WithModel(
				model.DeepgramTranscriptionModels[model.DeepgramNova3],
			),
			sttdeepgram.WithPunctuate(true),
			sttdeepgram.WithSmartFormat(true),
		), provider
	case "openai":
		return sttopenai.NewSpeechToText(
			sttopenai.WithAPIKey(requiredEnv("OPENAI_API_KEY")),
			sttopenai.WithModel(
				model.OpenAITranscriptionModels[model.GPT4oTranscribe],
			),
			sttopenai.WithLanguage("en"),
		), provider
	case "azure":
		return sttazure.NewSpeechToText(
			sttazure.WithAPIKey(requiredEnv("AZURE_SPEECH_KEY")),
			sttazure.WithRegion(envOrDefault("AZURE_SPEECH_REGION", "eastus")),
			sttazure.WithModel(
				model.AzureSpeechTranscriptionModels[model.AzureSpeechFastTranscription],
			),
			sttazure.WithLocales("en-US"),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use openai, deepgram, or azure)",
			provider,
		)
		return nil, ""
	}
}

func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func providerName() string {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		return "openai"
	}
	return provider
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
