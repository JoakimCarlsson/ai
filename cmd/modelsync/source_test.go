package main

import (
	"strings"
	"testing"
)

func TestRatePrefersTheUnqualifiedStandardRate(t *testing.T) {
	prices := []apiPrice{
		{
			Metric:   "input_tokens",
			Unit:     "per_1m_tokens",
			Amount:   1,
			Currency: "USD",
			Dims:     map[string]string{"tier": "batch"},
		},
		{
			Metric:   "input_tokens",
			Unit:     "per_1m_tokens",
			Amount:   4,
			Currency: "USD",
			Dims:     map[string]string{"region": "westus", "tier": "standard"},
		},
		{
			Metric:   "input_tokens",
			Unit:     "per_1k_tokens",
			Amount:   0.002,
			Currency: "USD",
			Dims: map[string]string{
				"deployment": "global",
				"tier":       "standard",
			},
		},
	}

	got, ok := rate(prices, "USD", tokenUnits, "input_tokens")
	if !ok {
		t.Fatal("no rate picked")
	}
	if got != 2 {
		t.Errorf("rate = %v, want 2 (the global standard rate per 1M)", got)
	}
}

func TestRateSkipsRatesThatArePricedForSomethingElse(t *testing.T) {
	prices := []apiPrice{
		{
			Metric:   "input_tokens",
			Unit:     "per_1m_tokens",
			Amount:   0.1,
			Currency: "USD",
			Dims:     map[string]string{"fine_tuned": "true"},
		},
		{
			Metric:   "input_tokens",
			Unit:     "per_hour",
			Amount:   0.2,
			Currency: "USD",
		},
		{
			Metric:   "input_tokens",
			Unit:     "per_1m_tokens",
			Amount:   3,
			Currency: "EUR",
		},
	}

	if _, ok := rate(prices, "USD", tokenUnits, "input_tokens"); ok {
		t.Error("want no rate: nothing here prices a plain USD request")
	}
	if got, ok := rate(prices, "EUR", tokenUnits, "input_tokens"); !ok ||
		got != 3 {
		t.Errorf("rate = %v (%v), want 3", got, ok)
	}
}

func TestRateFallsBackThroughMetricsInOrder(t *testing.T) {
	prices := []apiPrice{{
		Metric:   "usage",
		Unit:     "per_1k_tokens",
		Amount:   0.001,
		Currency: "USD",
	}}

	got, ok := rate(prices, "USD", tokenUnits, "input_tokens", "usage")
	if !ok || got != 1 {
		t.Errorf("rate = %v (%v), want 1 from the usage metric", got, ok)
	}
}

func TestModelForChat(t *testing.T) {
	m := apiModel{
		ID:   "demo-1",
		Name: "Demo 1",
		Kind: "chat",
		Prices: []apiPrice{
			{
				Metric:   "input_tokens",
				Unit:     "per_1m_tokens",
				Amount:   2,
				Currency: "USD",
			},
			{
				Metric:   "output_tokens",
				Unit:     "per_1m_tokens",
				Amount:   8,
				Currency: "USD",
			},
		},
		Attrs:  map[string]string{"api_id": "demo/demo-1"},
		Limits: map[string]int64{"context_window": 200000},
		Lists: map[string][]string{
			"features":          {"reasoning", "structured_outputs"},
			"input_modalities":  {"image", "text"},
			"output_modalities": {"text"},
		},
	}

	got := modelFor(chat("demo", "llm/demo", "demo"), m)

	if got.apiModel != "demo/demo-1" {
		t.Errorf("apiModel = %q, want demo/demo-1", got.apiModel)
	}
	for field, want := range map[string]string{
		"CostPer1MIn":             "2",
		"CostPer1MOut":            "8",
		"ContextWindow":           "200000",
		"CanReason":               "true",
		"SupportsAttachments":     "true",
		"SupportsStructuredOut":   "true",
		"SupportsImageGeneration": "false",
	} {
		if got.fields[field] != want {
			t.Errorf("%s = %q, want %q", field, got.fields[field], want)
		}
	}
	if _, ok := got.fields["DefaultMaxTokens"]; ok {
		t.Error(
			"DefaultMaxTokens written from a source that does not publish it",
		)
	}
	if got.seed["DefaultMaxTokens"] != "8192" {
		t.Errorf("seeded DefaultMaxTokens = %q, want 8192",
			got.seed["DefaultMaxTokens"])
	}
	if got.seed["Name"] != `"Demo 1"` {
		t.Errorf("seeded Name = %q, want \"Demo 1\"", got.seed["Name"])
	}
}

// TestModelForChatCarriesTheLifecycleTheProviderPublishes confirms all five
// lifecycle fields the source publishes reach the catalog entry.
func TestModelForChatCarriesTheLifecycleTheProviderPublishes(t *testing.T) {
	m := apiModel{
		ID: "old-model", Name: "Old", Kind: "chat",
		Attrs: map[string]string{
			"state":                   "deprecated",
			"release_date":            "2024-05-13",
			"last_updated":            "2024-12-17",
			"retirement_date":         "2026-09-28",
			"recommended_replacement": "new-model",
		},
	}

	got := modelFor(chat("demo", "llm/demo", "demo"), m)

	for _, want := range []struct{ field, value string }{
		{"State", `"deprecated"`},
		{"ReleaseDate", `"2024-05-13"`},
		{"LastUpdated", `"2024-12-17"`},
		{"RetirementDate", `"2026-09-28"`},
		{"ReplacedBy", `"new-model"`},
	} {
		if got.fields[want.field] != want.value {
			t.Errorf(
				"%s = %q, want %q",
				want.field,
				got.fields[want.field],
				want.value,
			)
		}
	}
}

// TestALifecycleFieldTheSourceOmitsIsLeftOut confirms a lifecycle field the
// source does not publish stays absent from the catalog entry rather than
// being written as an empty string.
func TestALifecycleFieldTheSourceOmitsIsLeftOut(t *testing.T) {
	m := apiModel{
		ID: "quiet-model", Name: "Quiet", Kind: "chat",
		Attrs: map[string]string{"state": "active"},
	}

	got := modelFor(chat("demo", "llm/demo", "demo"), m)

	if got.fields["State"] != `"active"` {
		t.Errorf("State = %q, want active", got.fields["State"])
	}
	for _, field := range []string{
		"ReleaseDate",
		"LastUpdated",
		"RetirementDate",
		"ReplacedBy",
	} {
		if v, present := got.fields[field]; present {
			t.Errorf(
				"%s = %q, want it absent -- the source publishes none",
				field,
				v,
			)
		}
	}
}

// TestTheTwoDatesAreCarriedSeparately confirms ReleaseDate and LastUpdated
// are each written to their own field, whichever combination the source
// publishes, rather than one substituting for the other.
func TestTheTwoDatesAreCarriedSeparately(t *testing.T) {
	for _, tc := range []struct {
		name             string
		attrs            map[string]string
		release, updated string
	}{
		{
			"only a release date",
			map[string]string{"release_date": "2023-11-06"},
			`"2023-11-06"`,
			"",
		},
		{
			"only an update date",
			map[string]string{"last_updated": "2025-04-14"},
			"",
			`"2025-04-14"`,
		},
		{
			"both",
			map[string]string{
				"release_date": "2024-05-13",
				"last_updated": "2024-12-17",
			},
			`"2024-05-13"`,
			`"2024-12-17"`,
		},
		{"neither", map[string]string{}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := modelFor(chat("demo", "llm/demo", "demo"),
				apiModel{ID: "m", Name: "M", Kind: "chat", Attrs: tc.attrs})

			if got.fields["ReleaseDate"] != tc.release {
				t.Errorf(
					"ReleaseDate = %q, want %q",
					got.fields["ReleaseDate"],
					tc.release,
				)
			}
			if got.fields["LastUpdated"] != tc.updated {
				t.Errorf(
					"LastUpdated = %q, want %q",
					got.fields["LastUpdated"],
					tc.updated,
				)
			}
		})
	}
}

func TestModelForTranscriptionPricesByDirection(t *testing.T) {
	m := apiModel{
		ID:   "whisper",
		Name: "Whisper",
		Kind: "transcription",
		Prices: []apiPrice{
			{
				Metric:   "audio",
				Unit:     "per_second",
				Amount:   0.00002,
				Currency: "EUR",
				Dims:     map[string]string{"direction": "input"},
			},
			{
				Metric:   "audio",
				Unit:     "per_second",
				Amount:   0.00004,
				Currency: "EUR",
				Dims:     map[string]string{"direction": "output"},
			},
		},
		Attrs: map[string]string{"max_file_size": "2 GB"},
		Lists: map[string][]string{"features": {"word_timestamps"}},
	}

	got := modelFor(transcription("demo", "stt/demo", "demo"), m)

	if got.fields["Currency"] != `"EUR"` {
		t.Errorf("Currency = %q, want EUR", got.fields["Currency"])
	}
	if got.fields["CostPer1MIn"] != "0.0012" {
		t.Errorf("CostPer1MIn = %q, want the per-minute input rate 0.0012",
			got.fields["CostPer1MIn"])
	}
	if got.fields["MaxFileSizeMB"] != "2048" {
		t.Errorf("MaxFileSizeMB = %q, want 2048", got.fields["MaxFileSizeMB"])
	}
	if got.fields["SupportsWordTimestamps"] != "true" {
		t.Error("word timestamps not carried over from the feature list")
	}
}

// toolCall is a standard per-1k-calls USD rate told apart by its detail dim,
// as the source publishes a hosted tool's rates.
func toolCall(amount float64, detail string) apiPrice {
	return apiPrice{
		Metric:   "tool_call",
		Unit:     "per_1k_calls",
		Amount:   amount,
		Currency: "USD",
		Dims:     map[string]string{"detail": detail, "tier": "standard"},
	}
}

// TestModelForToolReadsThePerThousandRate confirms the per-1k-calls rate the
// source publishes passes through unscaled.
func TestModelForToolReadsThePerThousandRate(t *testing.T) {
	m := apiModel{
		ID:   "web-search",
		Name: "Web search",
		Kind: "tool",
		Prices: []apiPrice{
			toolCall(10, "web search (all models)"),
			toolCall(25, "web search preview (non-reasoning models)"),
		},
	}

	got := modelFor(hostedTool("demo", "hostedtool/demo", "demo"), m)

	if got.fields["CostPer1KCalls"] != "10" {
		t.Errorf(
			"CostPer1KCalls = %q, want 10 -- per thousand calls, as published",
			got.fields["CostPer1KCalls"],
		)
	}
	if got.fields["Currency"] != `"USD"` {
		t.Errorf("Currency = %q, want USD", got.fields["Currency"])
	}
}

// TestModelForToolPrefersThePlainWebSearchRate confirms the plain web-search
// rate is chosen over the image and preview variants the source lists beside
// it, even when a variant is cheaper or sorts first, so the choice does not
// rest on rate's tie-break.
func TestModelForToolPrefersThePlainWebSearchRate(t *testing.T) {
	m := apiModel{
		ID:   "web-search",
		Name: "Web search",
		Kind: "tool",
		Prices: []apiPrice{
			toolCall(8, "image web search (all models)"),
			toolCall(12, "web search (all models)"),
			toolCall(25, "web search preview (non-reasoning models)"),
			toolCall(9, "web search preview (reasoning models)"),
		},
	}

	got := modelFor(hostedTool("demo", "hostedtool/demo", "demo"), m)

	if got.fields["CostPer1KCalls"] != "12" {
		t.Errorf(
			"CostPer1KCalls = %q, want 12 -- the plain web search rate",
			got.fields["CostPer1KCalls"],
		)
	}
}

// TestModelForToolFallsBackWithoutAPlainRate confirms a tool whose rates all
// name a variant still gets one, chosen by rate's ordinary tie-break.
func TestModelForToolFallsBackWithoutAPlainRate(t *testing.T) {
	m := apiModel{
		ID:   "web-search",
		Name: "Web search",
		Kind: "tool",
		Prices: []apiPrice{
			toolCall(25, "web search preview (non-reasoning models)"),
			toolCall(10, "web search preview (reasoning models)"),
		},
	}

	got := modelFor(hostedTool("demo", "hostedtool/demo", "demo"), m)

	if got.fields["CostPer1KCalls"] != "10" {
		t.Errorf(
			"CostPer1KCalls = %q, want 10 from the tie-break",
			got.fields["CostPer1KCalls"],
		)
	}
}

// TestAToolWithNoPerCallRateWritesNone confirms a tool with no per-call rate
// leaves CostPer1KCalls absent rather than writing a zero.
func TestAToolWithNoPerCallRateWritesNone(t *testing.T) {
	m := apiModel{
		ID: "agent-kit", Name: "Agent Kit", Kind: "tool",
		Prices: []apiPrice{
			{
				Metric:   "storage",
				Unit:     "per_gb_day",
				Amount:   0.1,
				Currency: "USD",
				Dims: map[string]string{
					"detail": "file storage",
					"tier":   "standard",
				},
			},
		},
	}

	got := modelFor(hostedTool("demo", "hostedtool/demo", "demo"), m)

	if v, present := got.fields["CostPer1KCalls"]; present {
		t.Errorf(
			"CostPer1KCalls = %q, want it absent -- no per-call rate",
			v,
		)
	}
}

func TestModelForEmbeddingSortsDimensionsNumerically(t *testing.T) {
	m := apiModel{
		ID:    "embed",
		Name:  "Embed",
		Kind:  "embedding",
		Attrs: map[string]string{"default_embedding_dimension": "1024"},
		Lists: map[string][]string{
			"embedding_dimensions": {"1024", "1536", "256", "512"},
			"output_dtypes":        {"float", "int8"},
		},
	}

	got := modelFor(embedding("demo", "embeddings/demo", "demo"), m)

	want := "[]int{256, 512, 1024, 1536}"
	if got.fields["SupportedDimensions"] != want {
		t.Errorf("SupportedDimensions = %q, want %q",
			got.fields["SupportedDimensions"], want)
	}
	if got.fields["EmbeddingDims"] != "1024" {
		t.Errorf("EmbeddingDims = %q, want 1024", got.fields["EmbeddingDims"])
	}
	if got.fields["SupportsOutputDtype"] != "true" {
		t.Error("output dtypes published but SupportsOutputDtype not set")
	}
}

func TestOpenRouterNamesAreQualified(t *testing.T) {
	tgt := openRouter(chat("openrouter", "llm/openrouter", "openrouter"))
	if got := tgt.displayName("AionLabs: Aion-2.0"); got !=
		"OpenRouter – Aion-2.0" {
		t.Errorf("displayName = %q", got)
	}
}

func TestSliceBreaksLongLiterals(t *testing.T) {
	fields := map[string]string{}
	setStrings(fields, "SupportedFormats", []string{"mp3", "wav"})
	if fields["SupportedFormats"] != `[]string{"mp3", "wav"}` {
		t.Errorf("short list not written on one line: %q",
			fields["SupportedFormats"])
	}

	setStrings(fields, "SupportedResponseFormats", []string{
		"diarized_json", "json", "srt", "text", "verbose_json", "vtt",
	})
	long := fields["SupportedResponseFormats"]
	if !strings.HasPrefix(long, "[]string{\n") ||
		!strings.HasSuffix(long, ",\n}") {
		t.Errorf("long list not broken over lines: %q", long)
	}
}
