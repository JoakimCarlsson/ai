package model

// Berget AI (https://berget.ai) is a Swedish, EU-hosted inference provider with
// an OpenAI-compatible API at https://api.berget.ai/v1.
//
// All prices below are in EUR, not USD: Berget bills in EUR and the Cost*
// fields hold the raw EUR figures from the /v1/models API (fetched 2026-06-30).
// The API does not return context windows, so ContextWindow values come from
// the upstream model cards (131_072 where a model's window is unpublished).
const (
	ProviderBerget Provider = "berget"

	BergetGPTOSS120B      ID = "openai/gpt-oss-120b"
	BergetMistralMedium35 ID = "mistralai/Mistral-Medium-3.5-128B"
	BergetMistralSmall32  ID = "mistralai/Mistral-Small-3.2-24B-Instruct-2506"
	BergetGLM47           ID = "zai-org/GLM-4.7-FP8"
	BergetGLM52           ID = "zai-org/GLM-5.2"
	BergetKimiK26         ID = "moonshotai/Kimi-K2.6"
	BergetGemma431B       ID = "google/gemma-4-31B-it"
	BergetLlama3370B      ID = "meta-llama/Llama-3.3-70B-Instruct"

	BergetE5LargeInstruct ID = "intfloat/multilingual-e5-large-instruct"
	BergetE5Large         ID = "intfloat/multilingual-e5-large"

	BergetBGERerankerV2M3 ID = "BAAI/bge-reranker-v2-m3"

	BergetKBWhisperLarge       ID = "KBLab/kb-whisper-large"
	BergetNBWhisperLarge       ID = "NbAiLab/nb-whisper-large"
	BergetFasterWhisperLargeV3 ID = "Systran/faster-whisper-large-v3"
)

// BergetModels maps Berget chat model IDs to their configurations.
// Prices are EUR per 1M tokens.
var BergetModels = map[ID]Model{
	BergetGPTOSS120B: {
		ID:                    BergetGPTOSS120B,
		Name:                  "GPT-OSS 120B",
		Provider:              ProviderBerget,
		APIModel:              "openai/gpt-oss-120b",
		CostPer1MIn:           0.20,
		CostPer1MOut:          0.75,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	BergetMistralMedium35: {
		ID:                    BergetMistralMedium35,
		Name:                  "Mistral Medium 3.5",
		Provider:              ProviderBerget,
		APIModel:              "mistralai/Mistral-Medium-3.5-128B",
		CostPer1MIn:           1.50,
		CostPer1MOut:          5.00,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	BergetMistralSmall32: {
		ID:                    BergetMistralSmall32,
		Name:                  "Mistral Small 3.2 24B",
		Provider:              ProviderBerget,
		APIModel:              "mistralai/Mistral-Small-3.2-24B-Instruct-2506",
		CostPer1MIn:           0.30,
		CostPer1MOut:          0.30,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	BergetGLM47: {
		ID:                    BergetGLM47,
		Name:                  "GLM-4.7 FP8",
		Provider:              ProviderBerget,
		APIModel:              "zai-org/GLM-4.7-FP8",
		CostPer1MIn:           0.70,
		CostPer1MOut:          2.50,
		ContextWindow:         200_000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	BergetGLM52: {
		ID:                    BergetGLM52,
		Name:                  "GLM-5.2",
		Provider:              ProviderBerget,
		APIModel:              "zai-org/GLM-5.2",
		CostPer1MIn:           1.40,
		CostPer1MOut:          4.40,
		ContextWindow:         200_000,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
	BergetKimiK26: {
		ID:                    BergetKimiK26,
		Name:                  "Kimi K2.6",
		Provider:              ProviderBerget,
		APIModel:              "moonshotai/Kimi-K2.6",
		CostPer1MIn:           0.75,
		CostPer1MOut:          3.50,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	BergetGemma431B: {
		ID:                    BergetGemma431B,
		Name:                  "Gemma 4 31B",
		Provider:              ProviderBerget,
		APIModel:              "google/gemma-4-31B-it",
		CostPer1MIn:           0.25,
		CostPer1MOut:          0.50,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsAttachments:   true,
		SupportsStructuredOut: true,
	},
	BergetLlama3370B: {
		ID:                    BergetLlama3370B,
		Name:                  "Llama 3.3 70B Instruct",
		Provider:              ProviderBerget,
		APIModel:              "meta-llama/Llama-3.3-70B-Instruct",
		CostPer1MIn:           0.90,
		CostPer1MOut:          0.90,
		ContextWindow:         131_072,
		DefaultMaxTokens:      8192,
		SupportsStructuredOut: true,
	},
}

// BergetEmbeddingModels maps Berget embedding model IDs to their configurations.
// CostPer1MTokens is EUR per 1M tokens.
var BergetEmbeddingModels = map[ID]EmbeddingModel{
	BergetE5LargeInstruct: {
		ID:              BergetE5LargeInstruct,
		Name:            "Multilingual E5 Large Instruct",
		Provider:        ProviderBerget,
		APIModel:        "intfloat/multilingual-e5-large-instruct",
		CostPer1MTokens: 0.03,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
	},
	BergetE5Large: {
		ID:              BergetE5Large,
		Name:            "Multilingual E5 Large",
		Provider:        ProviderBerget,
		APIModel:        "intfloat/multilingual-e5-large",
		CostPer1MTokens: 0.03,
		MaxInputTokens:  512,
		EmbeddingDims:   1024,
	},
}

// BergetRerankerModels maps Berget reranker model IDs to their configurations.
// CostPer1MTokens is EUR per 1M tokens.
var BergetRerankerModels = map[ID]RerankerModel{
	BergetBGERerankerV2M3: {
		ID:              BergetBGERerankerV2M3,
		Name:            "BGE Reranker v2 m3",
		Provider:        ProviderBerget,
		APIModel:        "BAAI/bge-reranker-v2-m3",
		CostPer1MTokens: 0.10,
		MaxQueryTokens:  512,
		MaxTotalTokens:  8192,
	},
}

// BergetTranscriptionModels maps Berget speech-to-text model IDs to their
// configurations.
//
// Berget bills transcription at EUR 0.000033 / audio second; the
// TranscriptionModel struct has no per-second field, so CostPer1MIn holds the
// per-minute equivalent (0.000033 * 60), matching the AssemblyAI convention in
// this package.
var BergetTranscriptionModels = map[ID]TranscriptionModel{
	BergetKBWhisperLarge: {
		ID:                  BergetKBWhisperLarge,
		Name:                "KB Whisper Large (Swedish)",
		Provider:            ProviderBerget,
		APIModel:            "KBLab/kb-whisper-large",
		CostPer1MIn:         0.00198,
		SupportedFormats:    []string{"flac", "mp3", "mp4", "mpeg", "mpga", "m4a", "ogg", "wav", "webm"},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
	BergetNBWhisperLarge: {
		ID:                  BergetNBWhisperLarge,
		Name:                "NB Whisper Large (Norwegian)",
		Provider:            ProviderBerget,
		APIModel:            "NbAiLab/nb-whisper-large",
		CostPer1MIn:         0.00198,
		SupportedFormats:    []string{"flac", "mp3", "mp4", "mpeg", "mpga", "m4a", "ogg", "wav", "webm"},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
	BergetFasterWhisperLargeV3: {
		ID:                  BergetFasterWhisperLargeV3,
		Name:                "Faster Whisper Large v3",
		Provider:            ProviderBerget,
		APIModel:            "Systran/faster-whisper-large-v3",
		CostPer1MIn:         0.00198,
		SupportedFormats:    []string{"flac", "mp3", "mp4", "mpeg", "mpga", "m4a", "ogg", "wav", "webm"},
		SupportsTimestamps:  true,
		SupportsTranslation: true,
	},
}
