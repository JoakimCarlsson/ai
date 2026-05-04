package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	aitracing "github.com/joakimcarlsson/ai/tracing"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	providers, err := aitracing.New(ctx,
		aitracing.WithResource(resource.Default()),
		aitracing.WithSpanProcessors(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx, providers)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	client := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
		llmopenai.WithMaxTokens(128),
	)

	resp, err := client.SendMessages(ctx, []message.Message{
		message.NewUserMessage(
			"Say hello from an OpenTelemetry traced request.",
		),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}

func shutdown(ctx context.Context, providers *aitracing.Providers) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := providers.Shutdown(ctx); err != nil {
		log.Printf("failed to shutdown OpenTelemetry providers: %v", err)
	}
}
