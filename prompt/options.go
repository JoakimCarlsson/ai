package prompt

import "text/template"

// Config holds template processing configuration.
type Config struct {
	Cache      *Cache
	FuncMap    template.FuncMap
	Required   []string
	StrictMode bool
	Name       string
}

// Option configures template processing.
type Option func(*Config)

// WithCache enables template caching using the provided cache.
func WithCache(c *Cache) Option {
	return func(cfg *Config) {
		cfg.Cache = c
	}
}

// WithFuncs adds custom template functions that merge with the defaults.
func WithFuncs(funcs template.FuncMap) Option {
	return func(cfg *Config) {
		cfg.FuncMap = funcs
	}
}

// WithRequired specifies variables that must be present in the data map.
func WithRequired(vars ...string) Option {
	return func(cfg *Config) {
		cfg.Required = vars
	}
}

// WithStrictMode causes execution to error on missing variables instead of using zero values.
func WithStrictMode() Option {
	return func(cfg *Config) {
		cfg.StrictMode = true
	}
}

// WithName sets the template name used for cache keys and error messages.
func WithName(name string) Option {
	return func(cfg *Config) {
		cfg.Name = name
	}
}

func applyOptions(opts []Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func buildFuncMap(custom template.FuncMap) template.FuncMap {
	merged := make(template.FuncMap, len(DefaultFuncMap)+len(custom))
	for k, v := range DefaultFuncMap {
		merged[k] = v
	}
	for k, v := range custom {
		merged[k] = v
	}
	return merged
}
