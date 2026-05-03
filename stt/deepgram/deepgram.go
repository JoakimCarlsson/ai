// Package deepgram provides a Deepgram implementation of the [stt.SpeechToText] interface,
// supporting both batch transcription and a real-time streaming session over WebSocket.
package deepgram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
)

const (
	defaultBaseURL              = "https://api.deepgram.com/v1"
	streamDefaultEndpointingMs  = 300
	streamKeepAlive             = 5 * time.Second
	streamReadDeadline          = 30 * time.Second
	streamHandshakeTimeout      = 10 * time.Second
)

// Options configures the Deepgram client.
type Options struct {
	apiKey   string
	model    model.TranscriptionModel
	timeout  *time.Duration
	language string

	punctuate           *bool
	diarize             *bool
	smartFormat         *bool
	streamEndpointingMs *int
	numerals            *bool
	profanityFilter     *bool
	dictation           *bool
	vadEvents           *bool
	streamInterim       *bool
	keyterms            []string
	keywords            []string
	redact              []string
	search              []string
	replace             []string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Deepgram.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration for batch transcription requests.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithLanguage sets the default language; per-call [stt.WithLanguage] overrides.
func WithLanguage(language string) Option { return func(o *Options) { o.language = language } }

// WithPunctuate enables automatic punctuation.
func WithPunctuate(enabled bool) Option { return func(o *Options) { o.punctuate = &enabled } }

// WithDiarize enables speaker diarization.
func WithDiarize(enabled bool) Option { return func(o *Options) { o.diarize = &enabled } }

// WithSmartFormat enables smart formatting.
func WithSmartFormat(enabled bool) Option { return func(o *Options) { o.smartFormat = &enabled } }

// WithStreamEndpointingMs sets the silence window (ms) Deepgram waits before emitting
// is_final on a streaming session.
func WithStreamEndpointingMs(ms int) Option {
	return func(o *Options) { o.streamEndpointingMs = &ms }
}

// WithNumerals converts spoken numbers to numeric format ("nine hundred" → "900").
func WithNumerals(enabled bool) Option { return func(o *Options) { o.numerals = &enabled } }

// WithProfanityFilter removes profanity from the transcript.
func WithProfanityFilter(enabled bool) Option {
	return func(o *Options) { o.profanityFilter = &enabled }
}

// WithDictation auto-formats spoken punctuation commands ("period" → ".", "new line" → "\n").
func WithDictation(enabled bool) Option { return func(o *Options) { o.dictation = &enabled } }

// WithVADEvents enables SpeechStarted / UtteranceEnd events on streaming sessions.
func WithVADEvents(enabled bool) Option { return func(o *Options) { o.vadEvents = &enabled } }

// WithStreamInterimResults toggles emission of interim transcripts. Defaults to true.
func WithStreamInterimResults(enabled bool) Option {
	return func(o *Options) { o.streamInterim = &enabled }
}

// WithKeyterms boosts recognition of specific words or phrases (Nova-3+).
func WithKeyterms(terms ...string) Option { return func(o *Options) { o.keyterms = terms } }

// WithKeywords boosts or suppresses recognition of specific words. Format
// "keyword" or "keyword:intensifier" (e.g. "claude:2"). For models older than Nova-3.
func WithKeywords(words ...string) Option { return func(o *Options) { o.keywords = words } }

// WithRedact redacts sensitive content categories from transcripts.
func WithRedact(categories ...string) Option { return func(o *Options) { o.redact = categories } }

// WithSearch runs acoustic pattern matching for the given terms.
func WithSearch(terms ...string) Option { return func(o *Options) { o.search = terms } }

// WithReplace substitutes terms in the transcript. Each entry is "find:replace".
func WithReplace(pairs ...string) Option { return func(o *Options) { o.replace = pairs } }

// Client implements [stt.SpeechToText] against the Deepgram API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewSpeechToText constructs a Deepgram speech-to-text client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 120 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return stt.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel { return c.options.model }

// SupportsStreaming reports true; Deepgram supports real-time streaming over WebSocket.
func (c *Client) SupportsStreaming() bool { return true }

type batchResponse struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
				Words      []struct {
					Word       string  `json:"word"`
					Start      float64 `json:"start"`
					End        float64 `json:"end"`
					Confidence float64 `json:"confidence"`
					Speaker    *int    `json:"speaker,omitempty"`
				} `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
	Metadata struct {
		Duration  float64        `json:"duration"`
		Channels  int            `json:"channels"`
		ModelInfo map[string]any `json:"model_info"`
		RequestID string         `json:"request_id"`
	} `json:"metadata"`
}

// Transcribe converts audio to text in batch mode.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{}
	for _, opt := range options {
		opt(&opts)
	}

	params := url.Values{}
	params.Set("model", c.options.model.APIModel)

	lang := c.options.language
	if opts.Language != "" {
		lang = opts.Language
	}
	if lang != "" {
		params.Set("language", lang)
	}

	if c.options.punctuate != nil && *c.options.punctuate {
		params.Set("punctuate", "true")
	}
	if c.options.diarize != nil && *c.options.diarize {
		params.Set("diarize", "true")
	}
	if c.options.smartFormat != nil && *c.options.smartFormat {
		params.Set("smart_format", "true")
	}

	reqURL := c.baseURL + "/listen?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(audioFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create transcription request: %w", err)
	}
	req.Header.Set("Content-Type", "audio/mpeg")
	req.Header.Set("Authorization", "Token "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make transcription request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcription response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcription API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var dgResp batchResponse
	if err := json.Unmarshal(body, &dgResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transcription response: %w", err)
	}

	return c.mapBatchResponse(&dgResp), nil
}

// Translate is not supported by Deepgram.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return nil, fmt.Errorf("deepgram does not support translation")
}

func (c *Client) mapBatchResponse(dgResp *batchResponse) *stt.Response {
	result := &stt.Response{
		Duration: dgResp.Metadata.Duration,
		Model:    c.options.model.APIModel,
		Usage:    stt.Usage{DurationSec: dgResp.Metadata.Duration},
	}

	if len(dgResp.Results.Channels) > 0 &&
		len(dgResp.Results.Channels[0].Alternatives) > 0 {
		alt := dgResp.Results.Channels[0].Alternatives[0]
		result.Text = alt.Transcript

		words := make([]stt.Word, len(alt.Words))
		for i, w := range alt.Words {
			words[i] = stt.Word{Word: w.Word, Start: w.Start, End: w.End}
		}
		result.Words = words
	}

	return result
}

// StreamTranscribe opens a Deepgram live transcription WebSocket. Reader parses
// Results frames into stt.StreamResult; writer forwards audio frames as binary
// messages and runs a KeepAlive ticker. Either ctx cancellation or audio channel
// close triggers a clean shutdown via CloseStream.
func (c *Client) StreamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...stt.Option,
) (<-chan stt.StreamResult, error) {
	opts := stt.Options{}
	for _, o := range options {
		o(&opts)
	}

	endpointing := streamDefaultEndpointingMs
	if c.options.streamEndpointingMs != nil {
		endpointing = *c.options.streamEndpointingMs
	}
	interim := true
	if c.options.streamInterim != nil {
		interim = *c.options.streamInterim
	}
	sampleRate := opts.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	channels := opts.Channels
	if channels == 0 {
		channels = 1
	}
	lang := c.options.language
	if opts.Language != "" {
		lang = opts.Language
	}

	q := url.Values{}
	q.Set("encoding", "linear16")
	q.Set("sample_rate", strconv.Itoa(sampleRate))
	q.Set("channels", strconv.Itoa(channels))
	q.Set("interim_results", strconv.FormatBool(interim))
	q.Set("endpointing", strconv.Itoa(endpointing))
	q.Set("model", c.options.model.APIModel)
	if lang != "" {
		q.Set("language", lang)
	}
	if c.options.punctuate != nil && *c.options.punctuate {
		q.Set("punctuate", "true")
	}
	if c.options.smartFormat != nil && *c.options.smartFormat {
		q.Set("smart_format", "true")
	}
	if c.options.diarize != nil && *c.options.diarize {
		q.Set("diarize", "true")
	}
	if c.options.numerals != nil && *c.options.numerals {
		q.Set("numerals", "true")
	}
	if c.options.profanityFilter != nil && *c.options.profanityFilter {
		q.Set("profanity_filter", "true")
	}
	if c.options.dictation != nil && *c.options.dictation {
		q.Set("dictation", "true")
	}
	if c.options.vadEvents != nil && *c.options.vadEvents {
		q.Set("vad_events", "true")
	}
	for _, kt := range c.options.keyterms {
		q.Add("keyterm", kt)
	}
	for _, kw := range c.options.keywords {
		q.Add("keywords", kw)
	}
	for _, r := range c.options.redact {
		q.Add("redact", r)
	}
	for _, s := range c.options.search {
		q.Add("search", s)
	}
	for _, rp := range c.options.replace {
		q.Add("replace", rp)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.deepgram.com",
		Path:     "/v1/listen",
		RawQuery: q.Encode(),
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Token "+c.options.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: streamHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	out := make(chan stt.StreamResult)
	done := make(chan struct{})

	var writeMu sync.Mutex
	send := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	_ = conn.SetReadDeadline(time.Now().Add(streamReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(streamReadDeadline))
	})

	go runReader(conn, out, done)
	go runWriter(ctx, conn, audio, out, done, send)

	return out, nil
}

func runReader(conn *websocket.Conn, out chan<- stt.StreamResult, done chan<- struct{}) {
	defer close(done)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !isCleanWSClose(err) {
				out <- stt.StreamResult{Error: err}
			}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(streamReadDeadline))
		pr, ok := parseStream(msg)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- stt.StreamResult,
	done <-chan struct{},
	send func(int, []byte) error,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	keepalive := time.NewTicker(streamKeepAlive)
	defer keepalive.Stop()

	closeStream := func() {
		_ = send(websocket.TextMessage, []byte(`{"type":"CloseStream"}`))
	}

	audioOpen := true
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			closeStream()
			<-done
			return
		case <-keepalive.C:
			if err := send(websocket.TextMessage, []byte(`{"type":"KeepAlive"}`)); err != nil {
				out <- stt.StreamResult{Error: err}
				_ = conn.Close()
				<-done
				return
			}
		case frame, ok := <-audio:
			if !ok {
				if audioOpen {
					closeStream()
					audioOpen = false
				}
				audio = nil
				continue
			}
			if err := send(websocket.BinaryMessage, frame); err != nil {
				out <- stt.StreamResult{Error: err}
				_ = conn.Close()
				<-done
				return
			}
		}
	}
}

func isCleanWSClose(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}

type streamResp struct {
	Type        string `json:"type"`
	IsFinal     bool   `json:"is_final"`
	SpeechFinal bool   `json:"speech_final"`
	Channel     struct {
		Alternatives []struct {
			Transcript string  `json:"transcript"`
			Confidence float64 `json:"confidence"`
			Words      []struct {
				Word  string  `json:"word"`
				Start float64 `json:"start"`
				End   float64 `json:"end"`
			} `json:"words"`
		} `json:"alternatives"`
	} `json:"channel"`
}

func parseStream(raw []byte) (stt.StreamResult, bool) {
	var resp streamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return stt.StreamResult{}, false
	}
	if resp.Type != "" && resp.Type != "Results" {
		return stt.StreamResult{}, false
	}
	if len(resp.Channel.Alternatives) == 0 {
		return stt.StreamResult{}, false
	}
	alt := resp.Channel.Alternatives[0]
	if alt.Transcript == "" {
		return stt.StreamResult{}, false
	}
	words := make([]stt.Word, len(alt.Words))
	for i, w := range alt.Words {
		words[i] = stt.Word{Word: w.Word, Start: w.Start, End: w.End}
	}
	return stt.StreamResult{
		Text:       alt.Transcript,
		Confidence: alt.Confidence,
		IsFinal:    resp.IsFinal || resp.SpeechFinal,
		WordCount:  len(alt.Words),
		Words:      words,
	}, true
}
