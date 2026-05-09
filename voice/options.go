package voice

import "github.com/joakimcarlsson/ai/tool"

// Option configures a VoiceAgent. Pass options to New.
type Option func(*VoiceAgent)

// WithSystemPrompt sets the system prompt prepended to every LLM call.
func WithSystemPrompt(prompt string) Option {
	return func(v *VoiceAgent) {
		v.systemPrompt = prompt
	}
}

// WithTools registers tools that the LLM may call during a conversation.
// Multiple WithTools options append.
func WithTools(tools ...tool.BaseTool) Option {
	return func(v *VoiceAgent) {
		v.tools = append(v.tools, tools...)
	}
}

// WithMaxToolIterations caps how many tool-call rounds may run inside a single
// assistant turn. Default is 4. Values <= 0 are ignored.
func WithMaxToolIterations(n int) Option {
	return func(v *VoiceAgent) {
		if n > 0 {
			v.maxToolIterations = n
		}
	}
}
