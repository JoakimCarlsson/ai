package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// HandoffConfig configures a handoff target that the current agent can transfer
// control to. When a handoff triggers, the target agent takes over the conversation
// with its own system prompt and tools, inheriting the full message history.
// A "transfer_to_<Name>" tool is automatically registered on the source agent.
type HandoffConfig struct {
	// Name identifies the target agent. The generated tool name is "transfer_to_<Name>".
	Name string
	// Description explains when this handoff should be used, shown to the LLM.
	Description string
	// Agent is the target agent that receives control after the handoff.
	Agent *Agent
}

type handoffInput struct {
	Reason string `json:"reason" desc:"Brief explanation of why control is being transferred" required:"false"`
}

type handoffTool struct {
	config HandoffConfig
}

func newHandoffTool(config HandoffConfig) *handoffTool {
	return &handoffTool{config: config}
}

func (t *handoffTool) Info() tool.ToolInfo {
	toolName := "transfer_to_" + t.config.Name
	description := fmt.Sprintf(
		"Transfer control to %s. %s",
		t.config.Name,
		t.config.Description,
	)
	return tool.NewToolInfo(toolName, description, handoffInput{})
}

func (t *handoffTool) Run(
	ctx context.Context,
	params tool.ToolCall,
) (tool.ToolResponse, error) {
	var input handoffInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		input.Reason = ""
	}

	msg := fmt.Sprintf("Transferring to %s.", t.config.Name)
	if input.Reason != "" {
		msg = fmt.Sprintf("Transferring to %s: %s", t.config.Name, input.Reason)
	}

	return tool.NewTextResponse(msg), nil
}

func detectHandoff(
	toolCalls []message.ToolCall,
	handoffs []HandoffConfig,
) *HandoffConfig {
	for _, tc := range toolCalls {
		if h := isHandoffTool(tc.Name, handoffs); h != nil {
			return h
		}
	}
	return nil
}

func isHandoffTool(name string, handoffs []HandoffConfig) *HandoffConfig {
	for i := range handoffs {
		if "transfer_to_"+handoffs[i].Name == name {
			return &handoffs[i]
		}
	}
	return nil
}

func rebuildMessagesForHandoff(
	ctx context.Context,
	newAgent *Agent,
	messages []message.Message,
) ([]message.Message, error) {
	systemPrompt, err := newAgent.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

	var rebuilt []message.Message

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = newAgent.llm.Model().ID
		rebuilt = append(rebuilt, sysMsg)
	}

	for _, msg := range messages {
		if msg.Role != message.System {
			rebuilt = append(rebuilt, msg)
		}
	}

	return rebuilt, nil
}

func findAgentName(root *Agent, active *Agent) string {
	for _, h := range root.handoffs {
		if h.Agent == active {
			return h.Name
		}
		if name := findAgentName(h.Agent, active); name != "" {
			return name
		}
	}
	return ""
}
