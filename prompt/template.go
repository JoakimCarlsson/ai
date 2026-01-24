package prompt

import (
	"fmt"
	"strings"
	"text/template"
)

// Template represents a parsed prompt template with optional caching.
type Template struct {
	name     string
	source   string
	parsed   *template.Template
	required []string
}

// New creates a new Template from source with optional configuration.
func New(source string, opts ...Option) (*Template, error) {
	cfg := applyOptions(opts)

	name := cfg.Name
	if name == "" {
		name = "prompt"
	}

	cacheKey := name
	if cfg.Cache != nil {
		if cfg.Name == "" {
			cacheKey = hashSource(source)
		}
		if cached := cfg.Cache.Get(cacheKey); cached != nil {
			return &Template{
				name:     name,
				source:   source,
				parsed:   cached,
				required: cfg.Required,
			}, nil
		}
	}

	funcMap := buildFuncMap(cfg.FuncMap)

	parsed, err := template.New(name).Funcs(funcMap).Parse(source)
	if err != nil {
		return nil, fmt.Errorf("prompt: parse error: %w", err)
	}

	if cfg.StrictMode {
		parsed = parsed.Option("missingkey=error")
	}

	if cfg.Cache != nil {
		cfg.Cache.Set(cacheKey, parsed)
	}

	return &Template{
		name:     name,
		source:   source,
		parsed:   parsed,
		required: cfg.Required,
	}, nil
}

// Process executes the template with the provided data.
func (t *Template) Process(data map[string]any) (string, error) {
	if data == nil {
		data = make(map[string]any)
	}

	if err := validateRequired(data, t.required); err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.parsed.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("prompt: execute error: %w", err)
	}

	return buf.String(), nil
}

// Process is a convenience function for one-shot template processing.
func Process(source string, data map[string]any, opts ...Option) (string, error) {
	tmpl, err := New(source, opts...)
	if err != nil {
		return "", err
	}
	return tmpl.Process(data)
}
