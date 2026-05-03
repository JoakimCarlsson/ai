package stt

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
)

const (
	deepgramStreamDefaultEndpointingMs = 300
	deepgramStreamKeepAlive            = 5 * time.Second
	deepgramStreamReadDeadline         = 30 * time.Second
	deepgramStreamHandshakeTimeout     = 10 * time.Second
)

type deepgramOptions struct {
	punctuate           *bool
	diarize             *bool
	smartFormat         *bool
	language            string
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

// DeepgramOption configures Deepgram-specific transcription behavior.
type DeepgramOption func(*deepgramOptions)

type deepgramClient struct {
	providerOptions transcriptionClientOptions
	options         deepgramOptions
	httpClient      *http.Client
	baseURL         string
}

// DeepgramClient is the Deepgram implementation of SpeechToTextClient.
type DeepgramClient SpeechToTextClient

type deepgramResponse struct {
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

func newDeepgramClient(
	opts transcriptionClientOptions,
) DeepgramClient {
	dgOpts := deepgramOptions{}
	for _, o := range opts.deepgramOptions {
		o(&dgOpts)
	}

	timeout := 120 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &deepgramClient{
		providerOptions: opts,
		options:         dgOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.deepgram.com/v1",
	}
}

func (d *deepgramClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	params := url.Values{}
	params.Set(
		"model",
		d.providerOptions.model.APIModel,
	)

	lang := d.options.language
	if opts.Language != "" {
		lang = opts.Language
	}
	if lang != "" {
		params.Set("language", lang)
	}

	if d.options.punctuate != nil && *d.options.punctuate {
		params.Set("punctuate", "true")
	}
	if d.options.diarize != nil && *d.options.diarize {
		params.Set("diarize", "true")
	}
	if d.options.smartFormat != nil && *d.options.smartFormat {
		params.Set("smart_format", "true")
	}

	reqURL := d.baseURL + "/listen?" + params.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		reqURL,
		bytes.NewReader(audioFile),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create transcription request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "audio/mpeg")
	req.Header.Set(
		"Authorization",
		"Token "+d.providerOptions.apiKey,
	)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make transcription request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read transcription response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"transcription API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var dgResp deepgramResponse
	if err := json.Unmarshal(body, &dgResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal transcription response: %w",
			err,
		)
	}

	return d.mapResponse(&dgResp), nil
}

func (d *deepgramClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"deepgram does not support translation",
	)
}

func (d *deepgramClient) mapResponse(
	dgResp *deepgramResponse,
) *Response {
	result := &Response{
		Duration: dgResp.Metadata.Duration,
		Model:    d.providerOptions.model.APIModel,
		Usage: Usage{
			DurationSec: dgResp.Metadata.Duration,
		},
	}

	if len(dgResp.Results.Channels) > 0 &&
		len(dgResp.Results.Channels[0].Alternatives) > 0 {
		alt := dgResp.Results.Channels[0].Alternatives[0]
		result.Text = alt.Transcript

		words := make([]Word, len(alt.Words))
		for i, w := range alt.Words {
			words[i] = Word{
				Word:  w.Word,
				Start: w.Start,
				End:   w.End,
			}
		}
		result.Words = words
	}

	return result
}

// WithDeepgramPunctuate enables automatic punctuation.
func WithDeepgramPunctuate(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.punctuate = &enabled
	}
}

// WithDeepgramDiarize enables speaker diarization.
func WithDeepgramDiarize(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.diarize = &enabled
	}
}

// WithDeepgramSmartFormat enables smart formatting.
func WithDeepgramSmartFormat(
	enabled bool,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.smartFormat = &enabled
	}
}

// WithDeepgramLanguage sets the default language for stt.
func WithDeepgramLanguage(
	language string,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.language = language
	}
}

// WithDeepgramStreamEndpointingMs sets the silence window (ms) Deepgram waits
// before emitting is_final on a streaming session. Streaming-only.
func WithDeepgramStreamEndpointingMs(
	ms int,
) DeepgramOption {
	return func(options *deepgramOptions) {
		options.streamEndpointingMs = &ms
	}
}

// WithDeepgramNumerals converts spoken numbers to numeric format
// ("nine hundred" → "900").
func WithDeepgramNumerals(enabled bool) DeepgramOption {
	return func(options *deepgramOptions) {
		options.numerals = &enabled
	}
}

// WithDeepgramProfanityFilter removes profanity from the transcript.
func WithDeepgramProfanityFilter(enabled bool) DeepgramOption {
	return func(options *deepgramOptions) {
		options.profanityFilter = &enabled
	}
}

// WithDeepgramDictation auto-formats spoken punctuation commands
// ("period" → ".", "new line" → "\n").
func WithDeepgramDictation(enabled bool) DeepgramOption {
	return func(options *deepgramOptions) {
		options.dictation = &enabled
	}
}

// WithDeepgramVADEvents enables SpeechStarted / UtteranceEnd events on
// streaming sessions. Streaming-only.
func WithDeepgramVADEvents(enabled bool) DeepgramOption {
	return func(options *deepgramOptions) {
		options.vadEvents = &enabled
	}
}

// WithDeepgramStreamInterimResults toggles emission of interim transcripts.
// Streaming-only; defaults to true.
func WithDeepgramStreamInterimResults(enabled bool) DeepgramOption {
	return func(options *deepgramOptions) {
		options.streamInterim = &enabled
	}
}

// WithDeepgramKeyterms boosts recognition of specific words or phrases
// (Nova-3+). Up to 100 keyterms per request.
func WithDeepgramKeyterms(terms ...string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.keyterms = terms
	}
}

// WithDeepgramKeywords boosts or suppresses recognition of specific words.
// Format: "keyword" or "keyword:intensifier" (e.g. "claude:2"). For models
// older than Nova-3.
func WithDeepgramKeywords(words ...string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.keywords = words
	}
}

// WithDeepgramRedact redacts sensitive content categories from transcripts.
// Common values: "pci", "numbers", "ssn".
func WithDeepgramRedact(categories ...string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.redact = categories
	}
}

// WithDeepgramSearch runs acoustic pattern matching for the given terms.
func WithDeepgramSearch(terms ...string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.search = terms
	}
}

// WithDeepgramReplace substitutes terms in the transcript. Each entry is
// "find:replace" (find must be lowercase).
func WithDeepgramReplace(pairs ...string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.replace = pairs
	}
}

// streamTranscribe opens a Deepgram live transcription WebSocket. Reader
// parses Results frames into StreamResult; writer forwards audio frames as
// binary messages and runs a KeepAlive ticker. Either ctx cancellation or
// audio channel close triggers a clean shutdown via CloseStream.
func (d *deepgramClient) streamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	opts := Options{}
	for _, o := range options {
		o(&opts)
	}

	endpointing := deepgramStreamDefaultEndpointingMs
	if d.options.streamEndpointingMs != nil {
		endpointing = *d.options.streamEndpointingMs
	}
	interim := true
	if d.options.streamInterim != nil {
		interim = *d.options.streamInterim
	}
	sampleRate := opts.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	channels := opts.Channels
	if channels == 0 {
		channels = 1
	}
	lang := d.options.language
	if opts.Language != "" {
		lang = opts.Language
	}

	q := url.Values{}
	q.Set("encoding", "linear16")
	q.Set("sample_rate", strconv.Itoa(sampleRate))
	q.Set("channels", strconv.Itoa(channels))
	q.Set("interim_results", strconv.FormatBool(interim))
	q.Set("endpointing", strconv.Itoa(endpointing))
	q.Set("model", d.providerOptions.model.APIModel)
	if lang != "" {
		q.Set("language", lang)
	}
	if d.options.punctuate != nil && *d.options.punctuate {
		q.Set("punctuate", "true")
	}
	if d.options.smartFormat != nil && *d.options.smartFormat {
		q.Set("smart_format", "true")
	}
	if d.options.diarize != nil && *d.options.diarize {
		q.Set("diarize", "true")
	}
	if d.options.numerals != nil && *d.options.numerals {
		q.Set("numerals", "true")
	}
	if d.options.profanityFilter != nil && *d.options.profanityFilter {
		q.Set("profanity_filter", "true")
	}
	if d.options.dictation != nil && *d.options.dictation {
		q.Set("dictation", "true")
	}
	if d.options.vadEvents != nil && *d.options.vadEvents {
		q.Set("vad_events", "true")
	}
	for _, kt := range d.options.keyterms {
		q.Add("keyterm", kt)
	}
	for _, kw := range d.options.keywords {
		q.Add("keywords", kw)
	}
	for _, r := range d.options.redact {
		q.Add("redact", r)
	}
	for _, s := range d.options.search {
		q.Add("search", s)
	}
	for _, rp := range d.options.replace {
		q.Add("replace", rp)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.deepgram.com",
		Path:     "/v1/listen",
		RawQuery: q.Encode(),
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Token "+d.providerOptions.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: deepgramStreamHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	out := make(chan StreamResult)
	done := make(chan struct{})

	var writeMu sync.Mutex
	send := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	_ = conn.SetReadDeadline(time.Now().Add(deepgramStreamReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(deepgramStreamReadDeadline))
	})

	go runDeepgramReader(conn, out, done)
	go runDeepgramWriter(ctx, conn, audio, out, done, send)

	return out, nil
}

func runDeepgramReader(
	conn *websocket.Conn,
	out chan<- StreamResult,
	done chan<- struct{},
) {
	defer close(done)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !isCleanWSClose(err) {
				out <- StreamResult{Error: err}
			}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(deepgramStreamReadDeadline))
		pr, ok := parseDeepgramStream(msg)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runDeepgramWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- StreamResult,
	done <-chan struct{},
	send func(int, []byte) error,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	keepalive := time.NewTicker(deepgramStreamKeepAlive)
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
				out <- StreamResult{Error: err}
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
				out <- StreamResult{Error: err}
				_ = conn.Close()
				<-done
				return
			}
		}
	}
}

func isCleanWSClose(err error) bool {
	return websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}

type deepgramStreamResp struct {
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

func parseDeepgramStream(raw []byte) (StreamResult, bool) {
	var resp deepgramStreamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return StreamResult{}, false
	}
	if resp.Type != "" && resp.Type != "Results" {
		return StreamResult{}, false
	}
	if len(resp.Channel.Alternatives) == 0 {
		return StreamResult{}, false
	}
	alt := resp.Channel.Alternatives[0]
	if alt.Transcript == "" {
		return StreamResult{}, false
	}
	words := make([]Word, len(alt.Words))
	for i, w := range alt.Words {
		words[i] = Word{Word: w.Word, Start: w.Start, End: w.End}
	}
	return StreamResult{
		Text:       alt.Transcript,
		Confidence: alt.Confidence,
		IsFinal:    resp.IsFinal || resp.SpeechFinal,
		WordCount:  len(alt.Words),
		Words:      words,
	}, true
}
