// Package prompt provides template processing for AI prompts with caching, validation, and extensible functions.
//
// This package enables dynamic prompt generation using Go's text/template syntax.
// It supports variable substitution, conditionals, loops, and custom template functions.
// Templates can be cached for performance and validated for required variables.
//
// # Basic Usage
//
//	result, err := prompt.Process("Hello, {{.name}}!", map[string]any{
//	    "name": "World",
//	})
//
// # With Caching
//
// For frequently used templates, caching avoids repeated parsing:
//
//	cache := prompt.NewCache()
//	tmpl, err := prompt.New("You are {{.role}}.",
//	    prompt.WithCache(cache),
//	    prompt.WithName("system"),
//	)
//	result, err := tmpl.Process(map[string]any{"role": "a helpful assistant"})
//
// # With Validation
//
// Ensure required variables are present before execution:
//
//	result, err := prompt.Process(template, data,
//	    prompt.WithRequired("name", "role"),
//	)
//
// # With Custom Functions
//
// Extend the template with custom functions:
//
//	result, err := prompt.Process("{{formatDate .timestamp}}", data,
//	    prompt.WithFuncs(template.FuncMap{
//	        "formatDate": myDateFormatter,
//	    }),
//	)
//
// # Built-in Functions
//
// The package provides many useful functions beyond Go's defaults:
//
//   - String: upper, lower, title, trim, trimPrefix, trimSuffix, replace, contains, hasPrefix, hasSuffix
//   - Collection: join, split, first, last, list
//   - Default: default, coalesce, empty, ternary
//   - Comparison: eq, ne, neq, lt, le, gt, ge
//   - Formatting: indent, nindent, quote, squote
package prompt
