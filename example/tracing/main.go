// Example tracing demonstrates OpenTelemetry tracing with an agent that uses tools.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tracing"
)

type spanPrinter struct{}

func (s spanPrinter) ExportSpans(
	_ context.Context,
	spans []sdktrace.ReadOnlySpan,
) error {
	for _, span := range spans {
		fmt.Printf(
			"[SPAN] %s (duration: %s)\n",
			span.Name(),
			span.EndTime().Sub(span.StartTime()),
		)
		for _, attr := range span.Attributes() {
			fmt.Printf(
				"       %s = %v\n",
				attr.Key,
				attr.Value.Emit(),
			)
		}
		if span.Status().Code != 0 {
			fmt.Printf(
				"       status: %s\n",
				span.Status().Description,
			)
		}
		fmt.Println()
	}
	return nil
}

func (s spanPrinter) Shutdown(_ context.Context) error {
	return nil
}

func main() {
	ctx := context.Background()

	providers, err := tracing.New(ctx,
		tracing.WithSpanProcessors(
			sdktrace.NewSimpleSpanProcessor(spanPrinter{}),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	weatherTool := &weatherToolImpl{}

	myAgent := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a helpful weather assistant.",
		),
		agent.WithTools(weatherTool),
	)

	response, err := myAgent.Chat(
		ctx,
		"What's the weather in Tokyo?",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n--- Response ---")
	fmt.Println(response.Content)

	_ = providers.Shutdown(ctx)
}

type weatherParams struct {
	Location string `json:"location" desc:"The city name"`
}

type weatherToolImpl struct{}

func (w *weatherToolImpl) Info() tool.Info {
	return tool.NewInfo(
		"get_weather",
		"Get the current weather for a location",
		weatherParams{},
	)
}

func (w *weatherToolImpl) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input weatherParams
	if err := json.Unmarshal(
		[]byte(params.Input),
		&input,
	); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"The weather in %s is sunny, 22°C",
			input.Location,
		),
	), nil
}
