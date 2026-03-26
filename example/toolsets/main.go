// Example toolsets demonstrates dynamic tool filtering based on runtime context.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

type phaseKey struct{}

type scanParams struct {
	Target string `json:"target" desc:"Target hostname or IP"`
}

type scanTool struct{}

func (s *scanTool) Info() tool.Info {
	return tool.NewInfo(
		"scan_ports",
		"Scan open ports on a target",
		scanParams{},
	)
}

func (s *scanTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input scanParams
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Scan results for %s: ports 22, 80, 443 open",
			input.Target,
		),
	), nil
}

type exploitParams struct {
	Target string `json:"target" desc:"Target hostname or IP"`
	Port   int    `json:"port"   desc:"Port to exploit"`
}

type exploitTool struct{}

func (e *exploitTool) Info() tool.Info {
	return tool.NewInfo(
		"run_exploit",
		"Attempt exploitation on target port",
		exploitParams{},
	)
}

func (e *exploitTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input exploitParams
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Exploit succeeded on %s:%d — shell obtained",
			input.Target,
			input.Port,
		),
	), nil
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

	allTools := tool.NewToolset("pentest",
		&scanTool{},
		&exploitTool{},
	)

	filtered := tool.NewFilterToolset("phase-aware", allTools,
		func(ctx context.Context, t tool.BaseTool) bool {
			phase, _ := ctx.Value(phaseKey{}).(string)
			if t.Info().Name == "run_exploit" {
				return phase == "exploitation"
			}
			return true
		},
	)

	myAgent := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a penetration testing assistant. Use available tools to help with the engagement.",
		),
		agent.WithToolsets(filtered),
		agent.WithMaxIterations(1),
	)

	reconCtx := context.WithValue(ctx, phaseKey{}, "recon")
	resp, err := myAgent.Chat(reconCtx, "Scan 10.0.0.1 for open ports")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[recon] %s\n", resp.Content)

	exploitCtx := context.WithValue(ctx, phaseKey{}, "exploitation")
	resp, err = myAgent.Chat(exploitCtx, "Exploit port 80 on 10.0.0.1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[exploit] %s\n", resp.Content)
}
