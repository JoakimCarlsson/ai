package agent

// ChatOption is a functional option for per-call overrides on Chat() and ChatStream().
type ChatOption func(*chatConfig)

type chatConfig struct {
	maxIterations int // 0 = use agent default
}

func applyChatOptions(opts []ChatOption) chatConfig {
	var cfg chatConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithMaxTurns sets the maximum number of tool-execution iterations for this call.
// Overrides the agent's WithMaxIterations setting. 0 means use the agent default.
func WithMaxTurns(n int) ChatOption {
	return func(c *chatConfig) {
		c.maxIterations = n
	}
}
