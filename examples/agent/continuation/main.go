package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

// countdownTool decreases a counter and recursively tells the LLM to call it again
type countdownTool struct{}

func newCountdownTool() *countdownTool {
	return &countdownTool{}
}

func (c *countdownTool) Info() tool.Info {
	return tool.NewInfo(
		"countdown",
		"Count down from a given number. Tells the model what the next number should be.",
		struct {
			Current int `json:"current" desc:"The current number to count down from"`
		}{},
	)
}

func (c *countdownTool) Run(
	ctx context.Context,
	tc tool.Call,
) (tool.Response, error) {
	var params struct {
		Current int `json:"current"`
	}
	if err := json.Unmarshal([]byte(tc.Input), &params); err != nil {
		return tool.NewTextResponse(""), err
	}
	if params.Current <= 0 {
		return tool.NewTextResponse("Done!"), nil
	}
	// Emulate a loop that forces multiple turns
	return tool.NewTextResponse(
		fmt.Sprintf(
			"Count is %d. Please call the countdown tool again with %d.",
			params.Current,
			params.Current-1,
		),
	), nil
}

// joinNames extracts the tool names from a slice of ToolCalls for display.
func joinNames(tcs []message.ToolCall) string {
	out := ""
	for i, tc := range tcs {
		if i > 0 {
			out += ", "
		}
		out += tc.Name
	}
	return out
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	llmClient := llmopenai.NewLLM(
		llmopenai.WithAPIKey(apiKey),
		llmopenai.WithModel(model.OpenAIModels[model.GPT4oMini]),
		llmopenai.WithMaxTokens(1024),
	)

	fmt.Println("=== Running Non-Streaming Continuation ===")
	runStatic(llmClient)

	fmt.Println("\n=== Running Streaming Continuation ===")
	runStream(llmClient)

	fmt.Println("\n=== Running Streaming Continuation (Timeout demo) ===")
	runStreamDemoAndTimeout(llmClient)
}

func runStatic(llmClient llm.LLM) {
	// Provider that approves the first continuation with a steering message, then declines the second
	staticProvider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationResponse, error) {
		fmt.Printf(
			"[Static Provider] max=%d total=%d pendingTools=[%s] (count=%d)\n",
			req.MaxIterations,
			req.TotalIterations,
			joinNames(req.ToolCalls),
			len(req.ToolCalls),
		)

		if req.TotalIterations == 2 {
			fmt.Println(
				"[Static Provider] Approving with a steering message and discarding current tool calls...",
			)
			return agent.ContinuationResponse{
				Decision:         agent.ContinuationApprove,
				Message:          "You are taking too long. Just tell me 'The countdown is canceled!' and stop calling tools.",
				DiscardToolCalls: true,
				// ToolMessage becomes the error text attached to each discarded
				// tool call's ToolResult so the model knows the call was canceled.
				ToolMessage: "Tool rejected.",
			}, nil
		}

		fmt.Println("[Static Provider] Declining...")
		return agent.ContinuationResponse{
			Decision: agent.ContinuationDecline,
			Message:  "We reached the hard limit. Wrap it up.",
		}, nil
	}

	assistant := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a helpful assistant. If asked to countdown, use the countdown tool continuously.",
		),
		agent.WithTools(newCountdownTool()),
		agent.WithMaxIterations(2), // Halt every 2 iterations
		agent.WithContinuationProvider(staticProvider),
	)

	resp, err := assistant.Chat(
		context.Background(),
		"Please count down from 5 using the tool.",
	)
	if err != nil {
		log.Fatalf("Error in chat: %v", err)
	}

	fmt.Printf("\nFinal Response:\n%s\n", resp.Content)
}

// runStreamDemoAndTimeout runs an extra scenario that exercises ContinuationTimeout:
// the provider never sends an approval/decline, so the consumer must surface a
// timeout decision (mirroring what a UI would do if the user walked away).
func runStreamDemoAndTimeout(llmClient llm.LLM) {
	fmt.Println("\n=== Running Streaming Continuation (Timeout demo) ===")

	// Provider blocks on a channel that is never written to, then surfaces a
	// timeout decision once the context is canceled.
	timeoutProvider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationResponse, error) {
		select {
		case <-time.After(2 * time.Second):
			return agent.ContinuationResponse{
				Decision: agent.ContinuationTimeout,
				Message:  "Tell me I did not respond in time.",
			}, nil
		case <-ctx.Done():
			return agent.ContinuationResponse{
				Decision: agent.ContinuationTimeout,
			}, ctx.Err()
		}
	}

	assistant := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a helpful assistant. If asked to countdown, use the countdown tool continuously.",
		),
		agent.WithTools(newCountdownTool()),
		agent.WithMaxIterations(1),
		agent.WithContinuationProvider(timeoutProvider),
	)

	eventStream := assistant.ChatStream(
		context.Background(),
		"Please count down from 3 using the tool.",
	)
	for event := range eventStream {
		switch event.Type {
		case types.EventContinuationRequired:
			req := event.ContinuationRequest
			fmt.Printf(
				"\n[Timeout Stream] Continuation required (total=%d) — no UI response expected\n",
				req.TotalIterations,
			)
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventComplete:
			fmt.Println("\n[Timeout Stream] Stream finished!")
		case types.EventError:
			fmt.Printf("\n[Timeout Stream] Error: %v\n", event.Error)
		}
	}
}

func runStream(llmClient llm.LLM) {
	// A channel to simulate an async UI approval process
	approvalChan := make(chan agent.ContinuationResponse)

	streamProvider := func(ctx context.Context, req agent.ContinuationRequest) (agent.ContinuationResponse, error) {
		// Wait for the UI/consumer to provide a decision
		select {
		case resp := <-approvalChan:
			return resp, nil
		case <-ctx.Done():
			return agent.ContinuationResponse{}, ctx.Err()
		}
	}

	assistant := agent.New(
		llmClient,
		agent.WithSystemPrompt(
			"You are a helpful assistant. If asked to countdown, use the countdown tool continuously.",
		),
		agent.WithTools(newCountdownTool()),
		agent.WithMaxIterations(2), // Halt every 2 iterations
		agent.WithContinuationProvider(streamProvider),
	)

	ctx := context.Background()
	eventStream := assistant.ChatStream(
		ctx,
		"Please count down from 3 using the tool.",
	)

	for event := range eventStream {
		switch event.Type {
		case types.EventContinuationRequired:
			req := event.ContinuationRequest
			fmt.Printf(
				"\n[Stream UI] Continuation required! (max=%d total=%d pendingTools=[%s])\n",
				req.MaxIterations,
				req.TotalIterations,
				joinNames(req.ToolCalls),
			)

			if req.TotalIterations == 2 {
				fmt.Println(
					"[Stream UI] Supplying approval with a steering message...",
				)
				approvalChan <- agent.ContinuationResponse{
					Decision:         agent.ContinuationApprove,
					Message:          "Skip the tools, just tell me what number you are currently at.",
					DiscardToolCalls: true,
				}
			} else {
				fmt.Println("[Stream UI] Supplying decline...")
				approvalChan <- agent.ContinuationResponse{
					Decision: agent.ContinuationDecline,
				}
			}
		case types.EventContentDelta:
			fmt.Print(event.Content)
		case types.EventToolUseStart:
			fmt.Printf(
				"\n[Stream UI] Tool Call Started: %s\n",
				event.ToolCall.Name,
			)
		case types.EventError:
			fmt.Printf("\n[Stream UI] Error: %v\n", event.Error)
		case types.EventComplete:
			fmt.Println("\n[Stream UI] Stream finished!")
		}
	}
}
