# Handoffs

Handoffs transfer full control from one agent to another. Unlike sub-agents (which return results to the orchestrator), handoffs permanently switch the active agent.

## Setup

```go
billing := agent.New(llmClient,
    agent.WithSystemPrompt("You handle billing inquiries."),
)

support := agent.New(llmClient,
    agent.WithSystemPrompt("You handle technical support."),
)

triage := agent.New(llmClient,
    agent.WithSystemPrompt("Route the user to the right specialist."),
    agent.WithHandoffs(
        agent.HandoffConfig{Name: "billing", Description: "Billing questions", Agent: billing},
        agent.HandoffConfig{Name: "support", Description: "Technical issues", Agent: support},
    ),
)

response, _ := triage.Chat(ctx, "I was charged twice on my last invoice")
fmt.Println(response.AgentName) // "billing"
```

## How It Works

1. Each `HandoffConfig` auto-generates a `transfer_to_<name>` tool
2. When the triage agent calls `transfer_to_billing`, control transfers permanently
3. The billing agent's system prompt replaces the triage agent's
4. The conversation history carries over
5. `ChatResponse.AgentName` indicates which agent produced the final response

## HandoffConfig

```go
type HandoffConfig struct {
    Name        string  // Used to generate transfer_to_<name> tool
    Description string  // Tells the LLM when to transfer
    Agent       *Agent  // The target agent
}
```

## Handoffs vs Sub-Agents

| | Sub-Agents | Handoffs |
|---|---|---|
| Control flow | Returns to orchestrator | Permanent transfer |
| System prompt | Sub-agent uses its own | Replaces current |
| Use case | Task delegation | Routing/triage |
