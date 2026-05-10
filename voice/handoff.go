package voice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// HandoffConfig configures a handoff target. The configured agent's name
// becomes a "transfer_to_<Name>" tool registered on the source agent. When
// the LLM calls that tool, the runner switches the active agent for the
// rest of the conversation: the target's system prompt, tools, LLM, hooks,
// context strategy, and (chained) handoffs all take over. The target's
// STT/TTS clients are ignored — the conversation's audio path stays bound
// to the original Agent so no audio glitch happens at the transfer
// boundary.
//
// Mirrors agent.HandoffConfig exactly.
type HandoffConfig struct {
	// Name identifies the target agent. The generated tool name is
	// "transfer_to_<Name>".
	Name string
	// Description tells the LLM when this handoff should be used.
	Description string
	// Agent is the target Agent that takes over after the handoff.
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

func (t *handoffTool) Info() tool.Info {
	toolName := "transfer_to_" + t.config.Name
	description := fmt.Sprintf(
		"Transfer control to %s. %s",
		t.config.Name,
		t.config.Description,
	)
	return tool.NewInfo(toolName, description, handoffInput{})
}

func (t *handoffTool) Run(
	_ context.Context,
	params tool.Call,
) (tool.Response, error) {
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

// detectHandoff scans the LLM's tool calls for any registered handoff and
// returns it if found.
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

// rebuildMessagesForHandoff strips system messages from the existing
// history and prepends the new agent's system prompt. The non-system
// messages (user turns, assistant turns, tool calls, tool results) are
// preserved so the new agent inherits the conversation context.
func rebuildMessagesForHandoff(
	newAgent *Agent,
	history []message.Message,
) []message.Message {
	var rebuilt []message.Message

	if newAgent.systemPrompt != "" {
		rebuilt = append(
			rebuilt,
			message.NewSystemMessage(newAgent.systemPrompt),
		)
	}

	for _, msg := range history {
		if msg.Role != message.System {
			rebuilt = append(rebuilt, msg)
		}
	}

	return rebuilt
}
