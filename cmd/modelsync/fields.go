package main

import (
	"math"
	"regexp"
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

func quote(s string) string { return strconv.Quote(s) }

func boolean(b bool) string { return strconv.FormatBool(b) }

func integer(v int64) string { return strconv.FormatInt(v, 10) }

// number reads a whole number the source publishes as a string, such as an
// embedding dimension. A value that is not one reads as zero, which every
// caller treats as unpublished.
func number(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

var fileSize = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*(MB|GB)\s*$`)

// megabytes reads a file size the source writes for people, such as "25MB" or
// "2 GB", as the megabyte count the catalogs hold.
func megabytes(s string) int64 {
	m := fileSize.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	if strings.EqualFold(m[2], "GB") {
		v *= 1024
	}
	return int64(math.Round(v))
}

// amount renders a price without the float noise a unit conversion leaves
// behind, matching how the rates are written by hand.
func amount(v float64) string {
	rounded := math.Round(v*1e6) / 1e6
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}
