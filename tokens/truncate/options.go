package truncate

// Config holds configuration for the truncate strategy.
type Config struct {
	// PreservePairs keeps user/assistant message pairs together when truncating.
	PreservePairs bool
	// MinMessages is the minimum number of messages to keep.
	MinMessages int
}

// Option configures the truncate strategy.
type Option func(*Config)

// PreservePairs keeps user/assistant message pairs together when truncating.
func PreservePairs() Option {
	return func(c *Config) {
		c.PreservePairs = true
	}
}

// MinMessages sets the minimum number of messages to keep.
func MinMessages(n int) Option {
	return func(c *Config) {
		c.MinMessages = n
	}
}

func Apply(opts ...Option) *Config {
	cfg := &Config{
		PreservePairs: false,
		MinMessages:   1,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
