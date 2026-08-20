package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
)

const (
	// sourceRepo is the catalog this tool is generated from.
	sourceRepo = "https://github.com/JoakimCarlsson/model-sync"
	// sourceURL is the raw api.json that repository publishes.
	sourceURL = "https://raw.githubusercontent.com/JoakimCarlsson/" +
		"model-sync/main/api.json"
)

// apiIndex is the model-sync api.json document: one entry per provider, each
// holding every model that provider publishes.
type apiIndex struct {
	Providers []apiProvider `json:"providers"`
}

type apiProvider struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Models []apiModel `json:"models"`
}

// apiModel is one model as api.json describes it. Everything beyond the
// identity fields is carried in the four open maps, so a new key the source
// starts publishing needs no change here to be readable.
type apiModel struct {
	ID       string              `json:"id"`
	Provider string              `json:"provider"`
	Name     string              `json:"name"`
	Kind     string              `json:"kind"`
	Prices   []apiPrice          `json:"prices"`
	Attrs    map[string]string   `json:"attrs"`
	Limits   map[string]int64    `json:"limits"`
	Lists    map[string][]string `json:"lists"`
}

// apiPrice is one published rate. The same metric appears many times per
// model, once per priced combination of tier, region, deployment and the rest,
// which dims names.
type apiPrice struct {
	Metric   string            `json:"metric"`
	Unit     string            `json:"unit"`
	Amount   float64           `json:"amount"`
	Currency string            `json:"currency"`
	Dims     map[string]string `json:"dims"`
}

func fetchIndex(ctx context.Context, url string) (*apiIndex, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"fetching %s: unexpected status %s",
			url,
			resp.Status,
		)
	}

	var index apiIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", url, err)
	}
	return &index, nil
}

// models returns the models one provider publishes for one kind.
func (idx *apiIndex) models(provider, kind string) []apiModel {
	var out []apiModel
	for _, p := range idx.Providers {
		if p.ID != provider {
			continue
		}
		for _, m := range p.Models {
			if m.Kind == kind {
				out = append(out, m)
			}
		}
	}
	return out
}

// apiID is the identifier a request names the model with, which is not always
// the catalog key api.json files the model under.
func (m apiModel) apiID() string {
	if id := m.Attrs["api_id"]; id != "" {
		return id
	}
	return m.ID
}

func (m apiModel) feature(name string) bool {
	return slices.Contains(m.Lists["features"], name)
}

func (m apiModel) anyFeature(names ...string) bool {
	for _, n := range names {
		if m.feature(n) {
			return true
		}
	}
	return false
}

func (m apiModel) takesModality(names ...string) bool {
	for _, n := range names {
		if slices.Contains(m.Lists["input_modalities"], n) {
			return true
		}
	}
	return false
}

func (m apiModel) emitsModality(name string) bool {
	return slices.Contains(m.Lists["output_modalities"], name)
}

// list returns the first of the named lists the model publishes, so a caller
// can state a preference order between equivalent keys.
func (m apiModel) list(keys ...string) []string {
	for _, k := range keys {
		if v := m.Lists[k]; len(v) > 0 {
			return v
		}
	}
	return nil
}

// limit returns the first of the named limits the model publishes, so a
// caller can state a preference order between equivalent keys.
func (m apiModel) limit(keys ...string) int64 {
	for _, k := range keys {
		if v := m.Limits[k]; v > 0 {
			return v
		}
	}
	return 0
}

// currency is the code the model's rates are quoted in. A provider bills in
// one currency, so the first rate settles it; a model with no published rate
// is assumed to be priced in USD.
func (m apiModel) currency() string {
	for _, p := range m.Prices {
		if p.Currency != "" {
			return p.Currency
		}
	}
	return "USD"
}

// Unit tables convert a published rate to the unit a catalog field holds. A
// unit absent from the table means the rate cannot fill that field.
var (
	tokenUnits = map[string]float64{
		"per_1m_tokens": 1,
		"per_1k_tokens": 1000,
	}
	charUnits = map[string]float64{
		"per_1m_characters": 1,
		"per_1k_characters": 1000,
	}
	minuteUnits = map[string]float64{
		"per_minute": 1,
		"per_second": 60,
		"per_hour":   1.0 / 60,
	}
	imageUnits = map[string]float64{"per_image": 1}
)

// variantDims mark a rate as belonging to something other than plain
// inference: a fine-tune, a training run, a promotion, a rate that has not
// taken effect yet. None of them describe what a request costs today.
var variantDims = map[string]bool{
	"batch":            true,
	"commitment":       true,
	"data_logging":     true,
	"discount":         true,
	"effective_from":   true,
	"effective_until":  true,
	"fine_tuned":       true,
	"legacy":           true,
	"method":           true,
	"model_grader":     true,
	"plan":             true,
	"promotional":      true,
	"service_tier":     true,
	"stage":            true,
	"surface":          true,
	"training":         true,
	"underlying_model": true,
}

// defaultDims are the dimension values a rate carries when it describes the
// ordinary way to call a model. A rate that only pins these is as unqualified
// as one that pins nothing.
var defaultDims = map[string]string{
	"deployment": "global",
	"modality":   "text",
	"tier":       "standard",
}

// rate picks the rate a catalog field should hold from everything the source
// publishes for a model.
//
// Metrics are tried in the given order and the first one with a usable rate
// wins. Within a metric the least qualified rate wins: a rate that names no
// region, context band or endpoint describes the model itself, while one that
// names several describes a corner of its pricing table. Ties go to the lower
// amount, then to the dimensions in name order, so a run is reproducible.
func rate(
	prices []apiPrice,
	currency string,
	units map[string]float64,
	metrics ...string,
) (float64, bool) {
	for _, metric := range metrics {
		var best apiPrice
		var bestAmount float64
		bestScore := -1

		for _, p := range prices {
			scale, ok := units[p.Unit]
			if !ok || p.Metric != metric || p.Currency != currency {
				continue
			}
			if !usableDims(p.Dims) {
				continue
			}

			amount := p.Amount * scale
			score := dimScore(p.Dims)
			if bestScore >= 0 && !better(
				score, amount, p.Dims,
				bestScore, bestAmount, best.Dims,
			) {
				continue
			}
			best, bestAmount, bestScore = p, amount, score
		}

		if bestScore >= 0 {
			return bestAmount, true
		}
	}
	return 0, false
}

// usableDims rejects a rate that is priced for something other than an
// ordinary request: a variant of the model, or a tier that has to be asked for.
func usableDims(dims map[string]string) bool {
	if tier, ok := dims["tier"]; ok && tier != defaultDims["tier"] {
		return false
	}
	for k := range dims {
		if variantDims[k] {
			return false
		}
	}
	return true
}

// dimScore counts how far a rate is qualified beyond the default way to call
// the model.
func dimScore(dims map[string]string) int {
	score := 0
	for k, v := range dims {
		if want, ok := defaultDims[k]; ok && v == want {
			continue
		}
		score++
	}
	return score
}

func better(
	score int, amount float64, dims map[string]string,
	bestScore int, bestAmount float64, bestDims map[string]string,
) bool {
	if score != bestScore {
		return score < bestScore
	}
	if amount != bestAmount {
		return amount < bestAmount
	}
	return dimKey(dims) < dimKey(bestDims)
}

func dimKey(dims map[string]string) string {
	pairs := make([]string, 0, len(dims))
	for k, v := range dims {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// withDim narrows a model's rates to those a dimension either names as want or
// leaves unstated, so an input rate is not filled from an output one.
func withDim(prices []apiPrice, dim, want string) []apiPrice {
	out := make([]apiPrice, 0, len(prices))
	for _, p := range prices {
		if v, ok := p.Dims[dim]; !ok || v == want {
			out = append(out, p)
		}
	}
	return out
}

// entry converts one source model into the Go literals a catalog entry of the
// target's kind is written with.
//
// Only fields the source actually describes are set. A field it says nothing
// about is left out, so the value the catalog already holds survives the
// regeneration, and target defaults fill it for a model seen for the first
// time.
func modelFor(t target, m apiModel) model {
	currency := m.currency()
	fields := map[string]string{
		"Provider": quote(t.pkg),
		"APIModel": quote(m.apiID()),
		"Currency": quote(currency),
	}
	seed := map[string]string{"Name": quote(t.displayName(m.Name))}

	switch t.kind {
	case kindChat:
		chatFieldsFor(m, currency, fields, seed)
	case kindImage:
		imageFieldsFor(m, currency, fields, seed)
	case kindSpeech:
		speechFieldsFor(m, currency, fields)
	case kindTranscription:
		transcriptionFieldsFor(m, currency, fields)
	case kindEmbedding:
		embeddingFieldsFor(m, currency, fields)
	case kindRerank:
		rerankFieldsFor(m, currency, fields)
	}

	return model{apiModel: m.apiID(), fields: fields, seed: seed}
}

func chatFieldsFor(
	m apiModel,
	currency string,
	fields, seed map[string]string,
) {
	setRate(fields, "CostPer1MIn", m.Prices, currency, tokenUnits,
		"input_tokens", "usage")
	setRate(fields, "CostPer1MOut", m.Prices, currency, tokenUnits,
		"output_tokens")
	setRate(fields, "CostPer1MInCached", m.Prices, currency, tokenUnits,
		"cached_input_tokens")
	setRate(fields, "CostPer1MOutCached", m.Prices, currency, tokenUnits,
		"cache_write_tokens")

	window := m.limit("context_window", "top_provider_context_window")
	setInt(fields, "ContextWindow", window)

	if out := m.limit("max_output_tokens"); out > 0 {
		fields["DefaultMaxTokens"] = integer(out)
	} else if window > 0 {
		seed["DefaultMaxTokens"] = integer(min(window/4, 8192))
	}

	fields["CanReason"] = boolean(
		m.feature("reasoning") ||
			m.Attrs["reasoning_mandatory"] == "true" ||
			len(m.Lists["reasoning_efforts"]) > 0,
	)
	fields["SupportsAttachments"] = boolean(
		m.takesModality("image", "audio", "video", "file"),
	)
	fields["SupportsStructuredOut"] = boolean(m.feature("structured_outputs"))
	fields["SupportsImageGeneration"] = boolean(m.emitsModality("image"))
}

func imageFieldsFor(
	m apiModel,
	currency string,
	fields, seed map[string]string,
) {
	if v, ok := rate(
		m.Prices, currency, imageUnits, "image_output", "usage",
	); ok && v > 0 {
		seed["Pricing"] = imagePricing(v)
	}

	setInt(fields, "MaxPromptTokens", m.limit(
		"max_input_tokens",
		"context_window",
		"top_provider_context_window",
	))
	setStrings(fields, "SupportedSizes", m.list("sizes", "image_sizes"))
	setStrings(fields, "SupportedAspectRatios", m.Lists["aspect_ratios"])
	setStrings(fields, "SupportedQualities", m.Lists["qualities"])
	fields["SupportsStreaming"] = boolean(m.feature("streaming"))
}

func speechFieldsFor(m apiModel, currency string, fields map[string]string) {
	setRate(fields, "CostPer1MChars", m.Prices, currency, charUnits,
		"speech", "audio", "input_characters", "input_tokens", "usage")
	setInt(fields, "MaxCharacters", m.limit("character_limit"))
	setStrings(fields, "SupportedFormats", m.Lists["output_formats"])
	fields["SupportsStreaming"] = boolean(m.feature("streaming"))
}

func transcriptionFieldsFor(
	m apiModel,
	currency string,
	fields map[string]string,
) {
	in := withDim(m.Prices, "direction", "input")
	if !setRate(fields, "CostPer1MIn", in, currency, tokenUnits,
		"input_tokens", "usage") {
		setRate(fields, "CostPer1MIn", in, currency, minuteUnits,
			"audio", "audio_input", "usage")
	}

	out := withDim(m.Prices, "direction", "output")
	if !setRate(fields, "CostPer1MOut", out, currency, tokenUnits,
		"output_tokens") {
		setRate(fields, "CostPer1MOut", out, currency, minuteUnits,
			"output_tokens")
	}

	setInt(fields, "MaxFileSizeMB", megabytes(m.Attrs["max_file_size"]))
	setStrings(fields, "SupportedFormats", m.Lists["audio_formats"])
	setStrings(fields, "SupportedResponseFormats", m.Lists["response_formats"])
	fields["SupportsTimestamps"] = boolean(
		m.anyFeature("timestamps", "word_timestamps", "utterances"),
	)
	fields["SupportsWordTimestamps"] = boolean(m.feature("word_timestamps"))
	fields["SupportsDiarization"] = boolean(
		m.anyFeature("speaker_diarization", "diarization"),
	)
	fields["SupportsTranslation"] = boolean(
		m.anyFeature("translation", "live_translation"),
	)
	fields["SupportsStreaming"] = boolean(
		m.anyFeature("streaming", "realtime"),
	)
}

func embeddingFieldsFor(
	m apiModel,
	currency string,
	fields map[string]string,
) {
	setRate(fields, "CostPer1MTokens", m.Prices, currency, tokenUnits,
		"input_tokens", "usage")
	setInt(fields, "MaxInputTokens", m.limit(
		"max_tokens_per_request",
		"context_window",
		"top_provider_context_window",
	))
	setInt(fields, "EmbeddingDims", number(
		m.Attrs["default_embedding_dimension"],
	))
	setInts(fields, "SupportedDimensions", m.Lists["embedding_dimensions"])
	setInt(fields, "MaxBatchSize", m.limit(
		"max_inputs_per_request",
		"max_texts_per_call",
	))
	fields["SupportsOutputDtype"] = boolean(
		len(m.Lists["output_dtypes"]) > 0,
	)
}

func rerankFieldsFor(m apiModel, currency string, fields map[string]string) {
	setRate(fields, "CostPer1MTokens", m.Prices, currency, tokenUnits,
		"input_tokens", "usage")
	setInt(fields, "MaxQueryTokens", m.limit("max_query_tokens"))
	setInt(fields, "MaxTotalTokens", m.limit(
		"context_window",
		"max_tokens_per_request",
		"top_provider_context_window",
	))
}

func setRate(
	fields map[string]string,
	field string,
	prices []apiPrice,
	currency string,
	units map[string]float64,
	metrics ...string,
) bool {
	v, ok := rate(prices, currency, units, metrics...)
	if !ok {
		return false
	}
	fields[field] = amount(v)
	return true
}

func setInt(fields map[string]string, field string, v int64) {
	if v > 0 {
		fields[field] = integer(v)
	}
}

func setStrings(fields map[string]string, field string, vals []string) {
	if len(vals) == 0 {
		return
	}
	quoted := make([]string, 0, len(vals))
	for _, v := range vals {
		quoted = append(quoted, quote(v))
	}
	fields[field] = slice("[]string", quoted)
}

// setInts writes a list the source publishes as strings as a Go []int, in
// ascending order rather than the lexical order api.json sorts it in.
func setInts(fields map[string]string, field string, vals []string) {
	nums := make([]int64, 0, len(vals))
	for _, v := range vals {
		if n := number(v); n > 0 {
			nums = append(nums, n)
		}
	}
	if len(nums) == 0 {
		return
	}
	slices.Sort(nums)

	rendered := make([]string, 0, len(nums))
	for _, n := range nums {
		rendered = append(rendered, integer(n))
	}
	fields[field] = slice("[]int", rendered)
}

// sliceWidth is the longest single-line slice literal a catalog is written
// with. A longer one is broken over one element per line, so the 80 column
// limit the repository formats to never has to break it afterwards.
const sliceWidth = 40

func slice(typeExpr string, elems []string) string {
	body := strings.Join(elems, ", ")
	if len(body) <= sliceWidth {
		return typeExpr + "{" + body + "}"
	}
	return typeExpr + "{\n" + strings.Join(elems, ",\n") + ",\n}"
}

// imagePricing renders a flat per-image rate. The source publishes one rate
// per model rather than a size and quality table, so the table an entry holds
// is only ever seeded, never overwritten.
func imagePricing(perImage float64) string {
	return "map[string]map[string]float64{\n\"default\": {\"default\": " +
		amount(perImage) + "},\n}"
}
