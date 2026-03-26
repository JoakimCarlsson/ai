// Example function_tool demonstrates wrapping plain Go functions as tools using functiontool.New.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool/functiontool"
)

type weatherParams struct {
	Location string `json:"location" desc:"The city name"`
	Units    string `json:"units"    desc:"Temperature units" enum:"celsius,fahrenheit" required:"false"`
}

type userLookupParams struct {
	Username string `json:"username" desc:"The username to look up"`
}

type userInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func main() {
	ctx := context.Background()

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	weatherTool := functiontool.New(
		"get_weather",
		"Get the current weather for a location",
		func(_ context.Context, p weatherParams) (string, error) {
			units := p.Units
			if units == "" {
				units = "celsius"
			}
			return fmt.Sprintf(
				"The weather in %s is sunny, 22°%s",
				p.Location,
				units,
			), nil
		},
	)

	userTool := functiontool.New(
		"lookup_user",
		"Look up a user by username and return their profile",
		func(p userLookupParams) (userInfo, error) {
			return userInfo{
				ID:       42,
				Username: p.Username,
				Email:    p.Username + "@example.com",
			}, nil
		},
	)

	myAgent := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a helpful assistant with access to weather and user lookup tools.",
		),
		agent.WithTools(weatherTool, userTool),
	)

	response, err := myAgent.Chat(
		ctx,
		"What's the weather in Paris? Also look up user 'alice'.",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
