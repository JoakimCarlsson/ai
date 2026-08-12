package main

import (
	"context"
)

// bergetModel is the subset of https://api.berget.ai/v1/models this tool reads.
type bergetModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"model_type"`
	Pricing struct {
		Currency string  `json:"currency"`
		Input    float64 `json:"input"`
		Output   float64 `json:"output"`
		Unit     string  `json:"unit"`
	} `json:"pricing"`
	Capabilities struct {
		FormattedOutput bool `json:"formatted_output"`
		FunctionCalling bool `json:"function_calling"`
		JSONMode        bool `json:"json_mode"`
		Streaming       bool `json:"streaming"`
		Vision          bool `json:"vision"`
	} `json:"capabilities"`
}

type bergetList struct {
	Data []bergetModel `json:"data"`
}

const bergetSource = "https://api.berget.ai/v1/models"

// berget serves chat, embedding, rerank and speech-to-text models from a single
// endpoint, split by model_type.
func berget() provider {
	const (
		defaultWindow  = "131072"
		defaultMaxOut  = "8192"
		bergetProvider = `"berget"`
	)

	windowDoc := []string{
		"Berget bills in EUR, so the cost fields are EUR, not USD.",
		"",
		"The API publishes no context window and no display names, so Name,",
		"ContextWindow and CanReason are carried over from the previous catalog;",
		"new models default to 131072 and are named after their slug.",
	}

	return provider{
		name: "berget",
		fetch: func(ctx context.Context) ([]model, error) {
			var list bergetList
			if err := fetchJSON(ctx, bergetSource, &list); err != nil {
				return nil, err
			}

			var models []model
			for _, m := range list.Data {
				k, ok := bergetKind(m.Type)
				if !ok {
					continue
				}
				models = append(models, bergetEntry(k, m))
			}
			return models, nil
		},
		targets: []target{
			{
				kind:       kindChat,
				path:       "llm/berget/models.go",
				pkg:        "berget",
				importPath: "github.com/joakimcarlsson/ai/llm",
				typeExpr:   "llm.Model",
				source:     bergetSource,
				idVerbatim: true,
				order:      chatFields,
				doc:        windowDoc,
				defaults: map[string]string{
					"Provider":         bergetProvider,
					"ContextWindow":    defaultWindow,
					"DefaultMaxTokens": defaultMaxOut,
				},
			},
			{
				kind:       kindEmbedding,
				path:       "embeddings/berget/models.go",
				pkg:        "berget",
				importPath: "github.com/joakimcarlsson/ai/embeddings",
				typeExpr:   "embeddings.EmbeddingModel",
				source:     bergetSource,
				idVerbatim: true,
				order:      embeddingFields,
				doc: []string{
					"Berget bills in EUR, so CostPer1MTokens is EUR, not USD.",
					"",
					"Display names, input limits and embedding dimensions are not part of",
					"the models response and are carried over from the previous catalog.",
				},
				defaults: map[string]string{"Provider": bergetProvider},
			},
			{
				kind:       kindRerank,
				path:       "rerankers/berget/models.go",
				pkg:        "berget",
				importPath: "github.com/joakimcarlsson/ai/rerankers",
				typeExpr:   "rerankers.RerankerModel",
				source:     bergetSource,
				idVerbatim: true,
				order:      rerankFields,
				doc: []string{
					"Berget bills in EUR, so CostPer1MTokens is EUR, not USD.",
					"",
					"Display names and token limits are not part of the models response",
					"and are carried over from the previous catalog.",
				},
				defaults: map[string]string{"Provider": bergetProvider},
			},
			{
				kind:       kindTranscription,
				path:       "stt/berget/models.go",
				pkg:        "berget",
				importPath: "github.com/joakimcarlsson/ai/stt",
				typeExpr:   "stt.TranscriptionModel",
				source:     bergetSource,
				idVerbatim: true,
				order:      transcriptionFields,
				doc: []string{
					"Berget bills transcription in EUR per audio second; CostPer1MIn holds",
					"the per-minute equivalent, matching the AssemblyAI convention in this",
					"repository.",
					"",
					"Display names, audio formats and timestamp support are not part of",
					"the models response and are carried over from the previous catalog.",
				},
				defaults: map[string]string{"Provider": bergetProvider},
			},
		},
	}
}

func bergetKind(modelType string) (kind, bool) {
	switch modelType {
	case "text":
		return kindChat, true
	case "embedding":
		return kindEmbedding, true
	case "rerank":
		return kindRerank, true
	case "speech-to-text":
		return kindTranscription, true
	default:
		return "", false
	}
}

func bergetEntry(k kind, m bergetModel) model {
	currency := m.Pricing.Currency
	if currency == "" {
		currency = "EUR"
	}

	fields := map[string]string{
		"Provider": `"berget"`,
		"APIModel": quote(m.ID),
		"Currency": quote(currency),
	}
	seed := map[string]string{"Name": quote(m.Name)}

	switch k {
	case kindChat:
		fields["CostPer1MIn"] = amount(m.Pricing.Input)
		fields["CostPer1MOut"] = amount(m.Pricing.Output)
		fields["SupportsAttachments"] = boolean(m.Capabilities.Vision)
		fields["SupportsStructuredOut"] = boolean(
			m.Capabilities.FormattedOutput || m.Capabilities.JSONMode,
		)
	case kindEmbedding, kindRerank:
		fields["CostPer1MTokens"] = amount(m.Pricing.Input)
	case kindTranscription:
		fields["CostPer1MIn"] = amount(m.Pricing.Input * 60)
		fields["CostPer1MOut"] = amount(m.Pricing.Output * 60)
	}

	return model{kind: k, apiModel: m.ID, fields: fields, seed: seed}
}
