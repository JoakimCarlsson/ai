package summarize

// Config holds configuration for the summarize strategy.
type Config struct {
	// KeepRecent is the number of recent messages to keep verbatim.
	KeepRecent int
}

// Option configures the summarize strategy.
type Option func(*Config)

// KeepRecent sets how many recent messages to keep verbatim (not summarized).
func KeepRecent(n int) Option {
	return func(c *Config) {
		c.KeepRecent = n
	}
}

func Apply(opts ...Option) *Config {
	cfg := &Config{
		KeepRecent: 5,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
