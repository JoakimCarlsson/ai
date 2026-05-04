// Package elevenlabs provides an ElevenLabs Scribe implementation of the [stt.SpeechToText]
// interface, supporting both batch transcription and a real-time streaming session over WebSocket.
package elevenlabs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
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
	defaultBaseURL         = "https://api.elevenlabs.io/v1"
	streamReadDeadline     = 30 * time.Second
	streamHandshakeTimeout = 10 * time.Second
)

// Options configures the ElevenLabs Scribe client.
type Options struct {
	apiKey  string
	model   model.TranscriptionModel
	timeout *time.Duration

	diarize                     *bool
	numSpeakers                 *int
	timestampGranularity        string
	tagAudioEvents              *bool
	streamVADSilenceMs          *int
	streamLanguageCode          string
	streamKeyterms              []string
	streamNoVerbatim            *bool
	streamIncludeTimestamps     *bool
	streamIncludeLanguageDetect *bool
	streamVADThreshold          *float64
	streamMinSpeechDurationMs   *int
	streamMinSilenceDurationMs  *int
	streamTimestampsGranularity string
	streamDisableLogging        *bool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with ElevenLabs.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration for batch transcription requests.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithDiarize enables speaker diarization.
func WithDiarize(enabled bool) Option { return func(o *Options) { o.diarize = &enabled } }

// WithNumSpeakers sets the expected number of speakers (0-32).
func WithNumSpeakers(n int) Option { return func(o *Options) { o.numSpeakers = &n } }

// WithTimestampGranularity sets timestamp level ("none", "word", "character") for batch.
func WithTimestampGranularity(g string) Option {
	return func(o *Options) { o.timestampGranularity = g }
}

// WithTagAudioEvents enables audio event detection (laughter, music, etc.).
func WithTagAudioEvents(enabled bool) Option { return func(o *Options) { o.tagAudioEvents = &enabled } }

// WithStreamVADSilenceMs sets the silence window (ms) ElevenLabs' VAD waits before emitting
// committed_transcript on a streaming session.
func WithStreamVADSilenceMs(ms int) Option { return func(o *Options) { o.streamVADSilenceMs = &ms } }

// WithStreamLanguageCode sets the language hint for streaming sessions.
func WithStreamLanguageCode(code string) Option {
	return func(o *Options) { o.streamLanguageCode = code }
}

// WithStreamKeyterms boosts recognition of specific words or phrases during streaming.
func WithStreamKeyterms(terms ...string) Option {
	return func(o *Options) { o.streamKeyterms = terms }
}

// WithStreamNoVerbatim strips filler words ("um", "uh", …) from streaming transcripts.
func WithStreamNoVerbatim(enabled bool) Option {
	return func(o *Options) { o.streamNoVerbatim = &enabled }
}

// WithStreamIncludeTimestamps emits word-level timing data on streaming transcripts.
func WithStreamIncludeTimestamps(enabled bool) Option {
	return func(o *Options) { o.streamIncludeTimestamps = &enabled }
}

// WithStreamIncludeLanguageDetection enables automatic language detection on streaming sessions.
func WithStreamIncludeLanguageDetection(enabled bool) Option {
	return func(o *Options) { o.streamIncludeLanguageDetect = &enabled }
}

// WithStreamVADThreshold sets VAD sensitivity (0.0–1.0) for streaming sessions.
func WithStreamVADThreshold(threshold float64) Option {
	return func(o *Options) { o.streamVADThreshold = &threshold }
}

// WithStreamMinSpeechDurationMs sets the minimum duration of speech (ms) before VAD considers it valid.
func WithStreamMinSpeechDurationMs(ms int) Option {
	return func(o *Options) { o.streamMinSpeechDurationMs = &ms }
}

// WithStreamMinSilenceDurationMs sets the minimum duration of silence (ms) before VAD declares end of speech.
func WithStreamMinSilenceDurationMs(ms int) Option {
	return func(o *Options) { o.streamMinSilenceDurationMs = &ms }
}

// WithStreamTimestampsGranularity sets timestamp resolution for streaming.
// Valid values: "none", "word", "character".
func WithStreamTimestampsGranularity(g string) Option {
	return func(o *Options) { o.streamTimestampsGranularity = g }
}

// WithStreamDisableLogging opts the streaming session out of ElevenLabs' server-side logging.
func WithStreamDisableLogging(enabled bool) Option {
	return func(o *Options) { o.streamDisableLogging = &enabled }
}

// Client implements [stt.SpeechToText] against the ElevenLabs Scribe API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewSpeechToText constructs an ElevenLabs Scribe speech-to-text client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get tracing
// spans and metrics.
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
	}, stt.TracingAttrs{
		Language: options.streamLanguageCode,
	})
}

// Model returns the configured transcription model.
func (c *Client) Model() model.TranscriptionModel { return c.options.model }

// SupportsStreaming reports true; ElevenLabs Scribe supports real-time streaming.
func (c *Client) SupportsStreaming() bool { return true }

type batchResponse struct {
	LanguageCode        string  `json:"language_code"`
	LanguageProbability float64 `json:"language_probability"`
	Text                string  `json:"text"`
	Words               []struct {
		Text      string  `json:"text"`
		Start     float64 `json:"start"`
		End       float64 `json:"end"`
		Type      string  `json:"type"`
		SpeakerID string  `json:"speaker_id,omitempty"`
	} `json:"words"`
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

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("model_id", c.options.model.APIModel); err != nil {
		return nil, fmt.Errorf("failed to write model_id field: %w", err)
	}
	if opts.Language != "" {
		if err := writer.WriteField("language_code", opts.Language); err != nil {
			return nil, fmt.Errorf("failed to write language_code field: %w", err)
		}
	}
	if c.options.diarize != nil && *c.options.diarize {
		if err := writer.WriteField("diarize", "true"); err != nil {
			return nil, fmt.Errorf("failed to write diarize field: %w", err)
		}
	}
	if c.options.numSpeakers != nil {
		if err := writer.WriteField("num_speakers", fmt.Sprintf("%d", *c.options.numSpeakers)); err != nil {
			return nil, fmt.Errorf("failed to write num_speakers field: %w", err)
		}
	}

	granularity := "word"
	if c.options.timestampGranularity != "" {
		granularity = c.options.timestampGranularity
	}
	if err := writer.WriteField("timestamps_granularity", granularity); err != nil {
		return nil, fmt.Errorf("failed to write timestamps field: %w", err)
	}

	filename := "audio.mp3"
	if opts.Filename != "" {
		filename = opts.Filename
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioFile); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/speech-to-text", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create STT request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make STT request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read STT response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("STT API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var elResp batchResponse
	if err := json.Unmarshal(body, &elResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal STT response: %w", err)
	}

	return c.mapBatchResponse(&elResp), nil
}

// Translate is not supported by ElevenLabs Scribe.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return nil, fmt.Errorf("elevenlabs scribe does not support translation")
}

func (c *Client) mapBatchResponse(elResp *batchResponse) *stt.Response {
	result := &stt.Response{
		Text:     elResp.Text,
		Language: elResp.LanguageCode,
		Model:    c.options.model.APIModel,
	}

	var words []stt.Word
	for _, w := range elResp.Words {
		if w.Type != "word" {
			continue
		}
		words = append(words, stt.Word{Word: w.Text, Start: w.Start, End: w.End})
	}
	result.Words = words

	return result
}

// StreamTranscribe opens an ElevenLabs Scribe v2 Realtime WebSocket. Audio frames are
// base64-encoded and wrapped in JSON input_audio_chunk events (per ElevenLabs' protocol —
// they don't accept raw binary).
func (c *Client) StreamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...stt.Option,
) (<-chan stt.StreamResult, error) {
	opts := stt.Options{}
	for _, o := range options {
		o(&opts)
	}

	sampleRate := opts.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	lang := c.options.streamLanguageCode
	if lang == "" {
		lang = opts.Language
	}

	q := url.Values{}
	q.Set("audio_format", fmt.Sprintf("pcm_%d", sampleRate))
	if lang != "" {
		q.Set("language_code", lang)
	}
	for _, kt := range c.options.streamKeyterms {
		q.Add("keyterms", kt)
	}
	if c.options.streamNoVerbatim != nil {
		q.Set("no_verbatim", strconv.FormatBool(*c.options.streamNoVerbatim))
	}
	if c.options.streamIncludeTimestamps != nil {
		q.Set("include_timestamps", strconv.FormatBool(*c.options.streamIncludeTimestamps))
	}
	if c.options.streamIncludeLanguageDetect != nil {
		q.Set("include_language_detection",
			strconv.FormatBool(*c.options.streamIncludeLanguageDetect))
	}
	if c.options.streamVADThreshold != nil {
		q.Set("vad_threshold",
			strconv.FormatFloat(*c.options.streamVADThreshold, 'f', -1, 64))
	}
	if c.options.streamMinSpeechDurationMs != nil {
		q.Set("min_speech_duration_ms", strconv.Itoa(*c.options.streamMinSpeechDurationMs))
	}
	if c.options.streamMinSilenceDurationMs != nil {
		q.Set("min_silence_duration_ms", strconv.Itoa(*c.options.streamMinSilenceDurationMs))
	}
	if c.options.streamTimestampsGranularity != "" {
		q.Set("timestamps_granularity", c.options.streamTimestampsGranularity)
	}
	if c.options.streamDisableLogging != nil {
		q.Set("disable_logging", strconv.FormatBool(*c.options.streamDisableLogging))
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.elevenlabs.io",
		Path:     "/v1/speech-to-text/realtime",
		RawQuery: q.Encode(),
	}
	hdr := http.Header{}
	hdr.Set("xi-api-key", c.options.apiKey)

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
			if !isCleanWSClose(err) && !errors.Is(err, net.ErrClosed) {
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

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			_ = conn.Close()
			<-done
			return
		case frame, ok := <-audio:
			if !ok {
				_ = conn.Close()
				<-done
				return
			}
			payload := fmt.Sprintf(
				`{"message_type":"input_audio_chunk","audio_base_64":"%s"}`,
				base64.StdEncoding.EncodeToString(frame),
			)
			if err := send(websocket.TextMessage, []byte(payload)); err != nil {
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
	MessageType string  `json:"message_type"`
	Text        string  `json:"text"`
	Transcript  string  `json:"transcript"`
	Confidence  float64 `json:"confidence"`
	Words       []struct {
		Text       string  `json:"text"`
		Start      float64 `json:"start"`
		End        float64 `json:"end"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
}

func parseStream(raw []byte) (stt.StreamResult, bool) {
	var resp streamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return stt.StreamResult{}, false
	}
	isFinal := false
	switch resp.MessageType {
	case "partial_transcript":
		isFinal = false
	case "committed_transcript", "committed_transcript_with_timestamps":
		isFinal = true
	default:
		return stt.StreamResult{}, false
	}
	text := resp.Transcript
	if text == "" {
		text = resp.Text
	}
	if text == "" {
		return stt.StreamResult{}, false
	}
	words := make([]stt.Word, len(resp.Words))
	for i, w := range resp.Words {
		words[i] = stt.Word{Word: w.Text, Start: w.Start, End: w.End}
	}
	return stt.StreamResult{
		Text:       text,
		Confidence: resp.Confidence,
		IsFinal:    isFinal,
		WordCount:  len(resp.Words),
		Words:      words,
	}, true
}
