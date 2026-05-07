// Package azure provides an Azure Speech Services implementation of the
// [stt.SpeechToText] interface using the Fast Transcription REST API.
//
// The Fast Transcription endpoint accepts audio up to ~2 hours, returns a
// single response with combined-phrases plus phrase- and word-level timing,
// and supports diarization. It is the closest analogue to Whisper batch
// transcription. WebSocket real-time transcription (Microsoft Cognitive
// Services Speech Protocol) is not yet implemented.
package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
)

const defaultAPIVersion = "2024-11-15"

// Options configures the Azure Speech STT client.
type Options struct {
	apiKey      string
	model       model.TranscriptionModel
	timeout     *time.Duration
	region      string
	endpoint    string
	apiVersion  string
	locales     []string
	channels    []int
	diarize     bool
	maxSpeakers int
	profanity   string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the Azure Speech subscription key.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithRegion sets the Azure region (e.g. "eastus", "westeurope"). Required
// unless [WithEndpoint] is used to point at a custom or sovereign-cloud host.
func WithRegion(region string) Option {
	return func(o *Options) { o.region = region }
}

// WithEndpoint overrides the host derived from [WithRegion]. Useful for
// sovereign clouds (Azure China at `*.api.cognitive.azure.cn`, Azure
// Government at `*.api.cognitive.azure.us`) and private endpoints. The value
// should be the scheme+host only, e.g. "https://eastus.api.cognitive.microsoft.com".
func WithEndpoint(endpoint string) Option {
	return func(o *Options) { o.endpoint = endpoint }
}

// WithAPIVersion overrides the Fast Transcription API version
// (default "2024-11-15").
func WithAPIVersion(v string) Option {
	return func(o *Options) { o.apiVersion = v }
}

// WithLocales sets the BCP-47 locales the server will recognise (e.g.
// "en-US", "es-ES"). Default: ["en-US"]. Per-call [stt.WithLanguage]
// overrides this with a single locale.
func WithLocales(locales ...string) Option {
	return func(o *Options) { o.locales = locales }
}

// WithChannels selects which audio channels to transcribe (0-indexed).
// Default: all channels merged.
func WithChannels(channels ...int) Option {
	return func(o *Options) { o.channels = channels }
}

// WithDiarization enables speaker diarization with up to maxSpeakers speakers.
func WithDiarization(maxSpeakers int) Option {
	return func(o *Options) {
		o.diarize = true
		o.maxSpeakers = maxSpeakers
	}
}

// WithProfanityFilterMode sets the profanity-filter behaviour.
// Valid values: "None", "Masked", "Removed", "Tags".
func WithProfanityFilterMode(mode string) Option {
	return func(o *Options) { o.profanity = mode }
}

// Client implements [stt.SpeechToText] against Azure Speech Fast Transcription.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewSpeechToText constructs an Azure Speech STT client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	options := Options{
		region:     "eastus",
		apiVersion: defaultAPIVersion,
		locales:    []string{"en-US"},
	}
	for _, o := range opts {
		o(&options)
	}

	timeout := 5 * time.Minute
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return stt.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
	}, stt.TracingAttrs{})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel { return c.options.model }

// SupportsStreaming reports whether StreamTranscribe is available. The Fast
// Transcription REST path is request/response only; the realtime WebSocket
// path is a separate sibling not yet implemented in this module.
func (c *Client) SupportsStreaming() bool { return false }

// StreamTranscribe is unimplemented; returns [stt.ErrStreamingNotSupported].
func (c *Client) StreamTranscribe(
	_ context.Context,
	_ <-chan []byte,
	_ ...stt.Option,
) (<-chan stt.StreamResult, error) {
	return nil, stt.ErrStreamingNotSupported
}

// Translate is not supported by Azure Fast Transcription.
func (c *Client) Translate(
	_ context.Context,
	_ []byte,
	_ ...stt.Option,
) (*stt.Response, error) {
	return nil, fmt.Errorf("azure speech: translation is not supported by the fast transcription api")
}

type definitionPayload struct {
	Locales         []string            `json:"locales"`
	Channels        []int               `json:"channels,omitempty"`
	ProfanityFilter string              `json:"profanityFilterMode,omitempty"`
	Diarization     *diarizationPayload `json:"diarization,omitempty"`
}

type diarizationPayload struct {
	MaxSpeakers int  `json:"maxSpeakers"`
	Enabled     bool `json:"enabled"`
}

type wordPayload struct {
	Text                 string  `json:"text"`
	OffsetMilliseconds   int64   `json:"offsetMilliseconds"`
	DurationMilliseconds int64   `json:"durationMilliseconds"`
	Confidence           float64 `json:"confidence"`
}

type phrasePayload struct {
	Speaker              int           `json:"speaker"`
	Channel              int           `json:"channel"`
	OffsetMilliseconds   int64         `json:"offsetMilliseconds"`
	DurationMilliseconds int64         `json:"durationMilliseconds"`
	Text                 string        `json:"text"`
	Words                []wordPayload `json:"words"`
	Locale               string        `json:"locale"`
	Confidence           float64       `json:"confidence"`
}

type combinedPhrasePayload struct {
	Channel int    `json:"channel"`
	Text    string `json:"text"`
}

type fastTranscriptionResponse struct {
	DurationMilliseconds int64                   `json:"durationMilliseconds"`
	CombinedPhrases      []combinedPhrasePayload `json:"combinedPhrases"`
	Phrases              []phrasePayload         `json:"phrases"`
}

// Transcribe submits the audio file to Azure Fast Transcription and returns
// the combined transcript with phrase- and word-level timing.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{}
	for _, o := range options {
		o(&opts)
	}

	if c.options.apiKey == "" {
		return nil, fmt.Errorf("azure speech: api key is required")
	}

	locales := c.options.locales
	if opts.Language != "" {
		locales = []string{opts.Language}
	}

	def := definitionPayload{
		Locales:         locales,
		Channels:        c.options.channels,
		ProfanityFilter: c.options.profanity,
	}
	if c.options.diarize {
		def.Diarization = &diarizationPayload{
			MaxSpeakers: c.options.maxSpeakers,
			Enabled:     true,
		}
	}

	body, contentType, err := c.buildMultipart(audioFile, def, opts.Filename)
	if err != nil {
		return nil, fmt.Errorf("azure speech: build request: %w", err)
	}

	endpoint, err := c.buildURL()
	if err != nil {
		return nil, fmt.Errorf("azure speech: build url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("azure speech: new request: %w", err)
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.options.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("azure speech: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"azure speech: status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(respBody)),
		)
	}

	var ft fastTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&ft); err != nil {
		return nil, fmt.Errorf("azure speech: decode response: %w", err)
	}

	return mapResponse(&ft, c.options.model.APIModel, locales), nil
}

func (c *Client) buildURL() (string, error) {
	host := c.options.endpoint
	if host == "" {
		if c.options.region == "" {
			return "", fmt.Errorf("region or endpoint is required")
		}
		host = fmt.Sprintf("https://%s.api.cognitive.microsoft.com", c.options.region)
	}
	host = strings.TrimRight(host, "/")
	return fmt.Sprintf(
		"%s/speechtotext/transcriptions:transcribe?api-version=%s",
		host,
		c.options.apiVersion,
	), nil
}

func (c *Client) buildMultipart(
	audio []byte,
	def definitionPayload,
	filename string,
) (io.Reader, string, error) {
	defJSON, err := json.Marshal(def)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	defHdr := textproto.MIMEHeader{}
	defHdr.Set("Content-Disposition", `form-data; name="definition"`)
	defHdr.Set("Content-Type", "application/json")
	defPart, err := mw.CreatePart(defHdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := defPart.Write(defJSON); err != nil {
		return nil, "", err
	}

	if filename == "" {
		filename = "audio.wav"
	}
	audioHdr := textproto.MIMEHeader{}
	audioHdr.Set(
		"Content-Disposition",
		fmt.Sprintf(`form-data; name="audio"; filename="%s"`, filename),
	)
	audioHdr.Set("Content-Type", "application/octet-stream")
	audioPart, err := mw.CreatePart(audioHdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := audioPart.Write(audio); err != nil {
		return nil, "", err
	}

	if err := mw.Close(); err != nil {
		return nil, "", err
	}

	return &buf, mw.FormDataContentType(), nil
}

func mapResponse(ft *fastTranscriptionResponse, apiModel string, locales []string) *stt.Response {
	var combined strings.Builder
	for i, p := range ft.CombinedPhrases {
		if i > 0 {
			combined.WriteByte(' ')
		}
		combined.WriteString(p.Text)
	}

	segments := make([]stt.Segment, 0, len(ft.Phrases))
	var words []stt.Word
	for i, p := range ft.Phrases {
		seg := stt.Segment{
			ID:    i,
			Start: float64(p.OffsetMilliseconds) / 1000.0,
			End:   float64(p.OffsetMilliseconds+p.DurationMilliseconds) / 1000.0,
			Text:  p.Text,
		}
		if p.Speaker > 0 {
			seg.Speaker = fmt.Sprintf("speaker-%d", p.Speaker)
		}
		segments = append(segments, seg)

		for _, w := range p.Words {
			words = append(words, stt.Word{
				Word:  w.Text,
				Start: float64(w.OffsetMilliseconds) / 1000.0,
				End:   float64(w.OffsetMilliseconds+w.DurationMilliseconds) / 1000.0,
			})
		}
	}

	language := ""
	if len(locales) > 0 {
		language = locales[0]
	}

	return &stt.Response{
		Text:     combined.String(),
		Language: language,
		Duration: float64(ft.DurationMilliseconds) / 1000.0,
		Segments: segments,
		Words:    words,
		Model:    apiModel,
	}
}
