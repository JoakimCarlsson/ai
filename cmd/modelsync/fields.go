package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// Field orders mirror the struct declarations in llm, image, tts, stt,
// embeddings and rerankers, so a generated catalog reads like a hand-written
// one.
var (
	chatFields = []string{
		"ID",
		"Name",
		"Provider",
		"APIModel",
		"Currency",
		"CostPer1MIn",
		"CostPer1MOut",
		"CostPer1MInCached",
		"CostPer1MOutCached",
		"ContextWindow",
		"DefaultMaxTokens",
		"CanReason",
		"SupportsAttachments",
		"SupportsStructuredOut",
		"SupportsImageGeneration",
	}
	imageFields = []string{
		"ID", "Name", "Provider", "APIModel", "Currency",
		"Pricing", "MaxPromptTokens", "SupportedSizes", "DefaultSize",
		"SupportedQualities", "DefaultQuality", "SupportedAspectRatios",
		"DefaultAspectRatio", "SupportsStreaming",
	}
	speechFields = []string{
		"ID", "Name", "Provider", "APIModel", "Currency",
		"CostPer1MChars", "MaxCharacters", "SupportedFormats", "DefaultFormat",
		"SupportsStreaming", "LatencyMs",
	}
	transcriptionFields = []string{
		"ID", "Name", "Provider", "APIModel", "Currency",
		"CostPer1MIn", "CostPer1MOut", "MaxFileSizeMB", "SupportedFormats",
		"SupportsTimestamps", "SupportsWordTimestamps", "SupportsDiarization",
		"SupportsTranslation", "SupportsStreaming", "SupportedResponseFormats",
	}
	embeddingFields = []string{
		"ID", "Name", "Provider", "APIModel", "Currency",
		"CostPer1MTokens", "MaxInputTokens", "EmbeddingDims",
		"SupportedDimensions", "MaxBatchSize", "SupportsOutputDtype",
		"MaxTokensPerBatch",
	}
	rerankFields = []string{
		"ID", "Name", "Provider", "APIModel", "Currency",
		"CostPer1MTokens", "MaxQueryTokens", "MaxTotalTokens",
	}
)

func fetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: unexpected status %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", url, err)
	}
	return nil
}

func quote(s string) string { return strconv.Quote(s) }

func boolean(b bool) string { return strconv.FormatBool(b) }

func integer(v int64) string { return strconv.FormatInt(v, 10) }

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

// perMillion converts a per-token rate to a per-1M-token one.
func perMillion(perToken string) string {
	return amount(parseFloat(perToken) * 1e6)
}

// amount renders a price without the float noise a raw conversion leaves
// behind, matching how the rates are written by hand.
func amount(v float64) string {
	rounded := math.Round(v*1e6) / 1e6
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}
