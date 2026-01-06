package memory

import (
	llm "github.com/joakimcarlsson/ai/providers"
)

// Config holds memory-related configuration for an agent.
type Config struct {
	AutoExtract bool
	AutoDedup   bool
	LLM         llm.LLM
}

// Option is a functional option for configuring memory behavior.
type Option func(*Config)

// AutoExtract enables automatic fact extraction from conversations.
// When enabled, the agent uses an LLM to extract relevant facts from each conversation
// and stores them in the memory store.
func AutoExtract() Option {
	return func(c *Config) {
		c.AutoExtract = true
	}
}

// AutoDedup enables LLM-based memory deduplication on store.
// When enabled, before storing a new memory, the agent searches for similar existing
// memories and asks an LLM to decide whether to ADD, UPDATE, DELETE, or skip.
func AutoDedup() Option {
	return func(c *Config) {
		c.AutoDedup = true
	}
}

// LLM sets a separate LLM for memory operations (extraction and deduplication).
// Useful for using a cheaper or faster model for background memory tasks while keeping
// the main conversation on a more capable model.
func LLM(l llm.LLM) Option {
	return func(c *Config) {
		c.LLM = l
	}
}

// Apply applies all options to a Config and returns it.
func Apply(opts ...Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
