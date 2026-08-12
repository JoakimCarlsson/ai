package main

import (
	"context"
	"slices"
)

// modelsDevModel is the subset of https://models.dev/api.json this tool reads.
type modelsDevModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Attachment bool   `json:"attachment"`
	Reasoning  bool   `json:"reasoning"`
	Structured bool   `json:"structured_output"`
	Modalities struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"modalities"`
	Limit struct {
		Context int64 `json:"context"`
		Output  int64 `json:"output"`
	} `json:"limit"`
	Cost struct {
		Input      float64 `json:"input"`
		Output     float64 `json:"output"`
		CacheRead  float64 `json:"cache_read"`
		CacheWrite float64 `json:"cache_write"`
	} `json:"cost"`
}

type modelsDevProvider struct {
	Models map[string]modelsDevModel `json:"models"`
}

const modelsDevSource = "https://models.dev/api.json"

// modelsDevCatalog binds one models.dev provider to one of our catalogs.
type modelsDevCatalog struct {
	provider string
	pkg      string
	path     string
	idPrefix string
	idFull   bool
}

// modelsDevCatalogs lists the catalogs sourced from models.dev. Providers that
// serve their own model API are deliberately absent: OpenRouter and Berget are
// synced from their own endpoints, which are authoritative for their rates and,
// in Berget's case, for the currency models.dev converts away.
var modelsDevCatalogs = []modelsDevCatalog{
	{provider: "anthropic", pkg: "anthropic", path: "llm/anthropic/models.go"},
	{provider: "openai", pkg: "openai", path: "llm/openai/models.go"},
	{provider: "google", pkg: "gemini", path: "llm/gemini/models.go"},
	{
		provider: "google-vertex",
		pkg:      "vertexai",
		path:     "llm/vertexai/models.go",
		idPrefix: "vertexai.",
	},
	{
		provider: "azure",
		pkg:      "azure",
		path:     "llm/azure/models.go",
		idPrefix: "azure.",
	},
	{
		provider: "groq",
		pkg:      "groq",
		path:     "llm/groq/models.go",
		idFull:   true,
	},
	{provider: "mistral", pkg: "mistral", path: "llm/mistral/models.go"},
	{provider: "deepseek", pkg: "deepseek", path: "llm/deepseek/models.go"},
	{provider: "xai", pkg: "xai", path: "llm/xai/models.go"},
	{
		provider: "cerebras",
		pkg:      "cerebras",
		path:     "llm/cerebras/models.go",
		idPrefix: "cerebras.",
	},
	{
		provider: "fireworks-ai",
		pkg:      "fireworks",
		path:     "llm/fireworks/models.go",
		idPrefix: "fireworks.",
	},
	{
		provider: "togetherai",
		pkg:      "together",
		path:     "llm/together/models.go",
		idPrefix: "together.",
		idFull:   true,
	},
	{
		provider: "perplexity",
		pkg:      "perplexity",
		path:     "llm/perplexity/models.go",
	},
}

// modelsDev sources the catalogs of providers that publish no pricing of their
// own from the models.dev index.
//
// models.dev indexes providers rather than serving them, so a model missing
// from it means the index has not caught up, not that the provider withdrew the
// model. Its catalogs therefore keep stale entries and report them instead of
// deleting constants other code may import.
func modelsDev() provider {
	targets := make([]target, 0, len(modelsDevCatalogs))
	for _, c := range modelsDevCatalogs {
		targets = append(targets, target{
			kind:       kind("modelsdev:" + c.provider),
			path:       c.path,
			pkg:        c.pkg,
			importPath: "github.com/joakimcarlsson/ai/llm",
			typeExpr:   "llm.Model",
			source:     modelsDevSource,
			idPrefix:   c.idPrefix,
			idFull:     c.idFull,
			keepStale:  true,
			order:      chatFields,
			doc: []string{
				"Rates are USD per 1M tokens, as models.dev indexes them for the",
				c.provider + " provider.",
				"",
				"models.dev indexes providers rather than serving them, so models it",
				"stops listing are kept here and reported by the sync instead of",
				"being deleted. Display names are set when a model is first added and",
				"carried over from then on.",
			},
			defaults: map[string]string{"Provider": quote(c.pkg)},
		})
	}

	return provider{
		name:    "models.dev",
		source:  modelsDevSource,
		targets: targets,
		fetch: func(ctx context.Context) ([]model, error) {
			var index map[string]modelsDevProvider
			if err := fetchJSON(ctx, modelsDevSource, &index); err != nil {
				return nil, err
			}

			var models []model
			for _, c := range modelsDevCatalogs {
				p, ok := index[c.provider]
				if !ok {
					continue
				}
				for _, m := range p.Models {
					models = append(models, modelsDevEntry(c, m))
				}
			}
			return models, nil
		},
	}
}

func modelsDevEntry(c modelsDevCatalog, m modelsDevModel) model {
	fields := map[string]string{
		"Provider":              quote(c.pkg),
		"APIModel":              quote(m.ID),
		"Currency":              `"USD"`,
		"CostPer1MIn":           amount(m.Cost.Input),
		"CostPer1MOut":          amount(m.Cost.Output),
		"CostPer1MInCached":     amount(m.Cost.CacheRead),
		"CostPer1MOutCached":    amount(m.Cost.CacheWrite),
		"ContextWindow":         integer(m.Limit.Context),
		"DefaultMaxTokens":      integer(m.Limit.Output),
		"CanReason":             boolean(m.Reasoning),
		"SupportsAttachments":   boolean(m.Attachment),
		"SupportsStructuredOut": boolean(m.Structured),
		"SupportsImageGeneration": boolean(
			slices.Contains(m.Modalities.Output, "image"),
		),
	}

	return model{
		kind:     kind("modelsdev:" + c.provider),
		apiModel: m.ID,
		fields:   fields,
		seed:     map[string]string{"Name": quote(m.Name)},
	}
}
