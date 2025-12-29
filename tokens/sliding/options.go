package sliding

// Config holds configuration for the sliding window strategy.
type Config struct {
	// KeepLast is the number of recent messages to retain.
	KeepLast int
}

// Option configures the sliding window strategy.
type Option func(*Config)

// KeepLast sets how many recent messages to retain.
func KeepLast(n int) Option {
	return func(c *Config) {
		c.KeepLast = n
	}
}

func Apply(opts ...Option) *Config {
	cfg := &Config{
		KeepLast: 10,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
