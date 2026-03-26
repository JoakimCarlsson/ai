// Example confirmation_stream demonstrates tool confirmation with streaming, where confirmation
// events appear on the event channel and the provider is unblocked via a channel.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type transferParams struct {
	To     string  `json:"to"     desc:"Recipient name"`
	Amount float64 `json:"amount" desc:"Amount to transfer"`
}

type transferTool struct{}

func (t *transferTool) Info() tool.Info {
	info := tool.NewInfo(
		"transfer_funds",
		"Transfer money to a recipient",
		transferParams{},
	)
	info.RequireConfirmation = true
	return info
}

func (t *transferTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input transferParams
	if err := json.Unmarshal(
		[]byte(params.Input),
		&input,
	); err != nil {
		return tool.NewTextErrorResponse(err.Error()), nil
	}
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Transferred $%.2f to %s",
			input.Amount,
			input.To,
		),
	), nil
}

type balanceTool struct{}

func (b *balanceTool) Info() tool.Info {
	return tool.NewInfo(
		"check_balance",
		"Check the current account balance",
		struct{}{},
	)
}

func (b *balanceTool) Run(
	_ context.Context,
	_ tool.Call,
) (tool.Response, error) {
	return tool.NewTextResponse("Current balance: $5,420.00"), nil
}

type pendingApproval struct {
	approved bool
	done     chan struct{}
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

	var mu sync.Mutex
	pending := make(map[string]*pendingApproval)

	provider := func(
		_ context.Context,
		req tool.ConfirmationRequest,
	) (bool, error) {
		p := &pendingApproval{done: make(chan struct{})}
		mu.Lock()
		pending[req.ToolCallID] = p
		mu.Unlock()
		<-p.done
		return p.approved, nil
	}

	myAgent := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a banking assistant. Check balance with check_balance, transfer with transfer_funds.",
		),
		agent.WithTools(&balanceTool{}, &transferTool{}),
		agent.WithConfirmationProvider(provider),
	)

	fmt.Println("Streaming with confirmation:")
	fmt.Println()

	for event := range myAgent.ChatStream(
		ctx,
		"Check my balance, then transfer $200 to Alice",
	) {
		switch event.Type {
		case types.EventContentDelta:
			fmt.Print(event.Content)

		case types.EventConfirmationRequired:
			req := event.ConfirmationRequest
			fmt.Printf(
				"\n[Confirmation] Tool %q with: %s\n",
				req.ToolName,
				req.Input,
			)
			fmt.Print("Approve? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			approved := strings.TrimSpace(
				strings.ToLower(line),
			) == "y"

			mu.Lock()
			p := pending[req.ToolCallID]
			mu.Unlock()
			p.approved = approved
			close(p.done)

		case types.EventToolUseStop:
			if event.ToolResult != nil {
				fmt.Printf(
					"\n[%s] %s\n",
					event.ToolResult.ToolName,
					event.ToolResult.Output,
				)
			}

		case types.EventComplete:
			fmt.Println()

		case types.EventError:
			log.Fatal(event.Error)
		}
	}
}
