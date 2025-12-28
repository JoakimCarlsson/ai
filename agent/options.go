package agent

import (
	"github.com/joakimcarlsson/ai/tool"
)

type AgentOption func(*Agent)

func WithSystemPrompt(prompt string) AgentOption {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

func WithTools(tools ...tool.BaseTool) AgentOption {
	return func(a *Agent) {
		a.tools = append(a.tools, tools...)
	}
}

func WithMaxIterations(max int) AgentOption {
	return func(a *Agent) {
		a.maxIterations = max
	}
}

func WithAutoExecute(auto bool) AgentOption {
	return func(a *Agent) {
		a.autoExecute = auto
	}
}

func WithMemory(memory Memory) AgentOption {
	return func(a *Agent) {
		a.memory = memory
	}
}

func WithUserIDKey(key string) AgentOption {
	return func(a *Agent) {
		a.userIDKey = key
	}
}

func WithAutoExtract(enabled bool) AgentOption {
	return func(a *Agent) {
		a.autoExtract = enabled
	}
}

