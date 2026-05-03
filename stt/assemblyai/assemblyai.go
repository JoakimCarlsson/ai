// Package assemblyai provides an AssemblyAI implementation of the [stt.SpeechToText]
// interface, supporting both upload+poll batch transcription and the v3 Universal-Streaming
// real-time WebSocket API.
package assemblyai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	defaultBaseURL                  = "https://api.assemblyai.com/v2"
	streamDefaultEndOfTurnSilenceMs = 700
	streamReadDeadline              = 30 * time.Second
	streamHandshakeTimeout          = 10 * time.Second
	streamMinChunkMs                = 100
)

// Options configures the AssemblyAI client.
type Options struct {
	apiKey  string
	model   model.TranscriptionModel
	timeout *time.Duration

	pollInterval                       time.Duration
	maxPollDuration                    time.Duration
	speakerLabels                      bool
	streamEndOfTurnSilenceMs           *int
	streamSpeechModel                  string
	streamFormatTurns                  *bool
	streamEndOfTurnConfidenceThreshold *float64
	streamMaxTurnSilence               *int
	streamKeyterms                     []string
	streamPunctuationFilter            *bool
	streamWordFinalizationMaxWaitMs    *int
	streamExtraSessionInformation      *bool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with AssemblyAI.
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the transcription model.
func WithModel(m model.TranscriptionModel) Option { return func(o *Options) { o.model = m } }

// WithTimeout sets the maximum duration for HTTP requests (upload, create, poll).
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithPollInterval sets the interval between polling attempts.
func WithPollInterval(d time.Duration) Option { return func(o *Options) { o.pollInterval = d } }

// WithMaxPollDuration sets the maximum duration to wait for transcription completion.
func WithMaxPollDuration(d time.Duration) Option { return func(o *Options) { o.maxPollDuration = d } }

// WithSpeakerLabels enables speaker diarization.
func WithSpeakerLabels(enabled bool) Option { return func(o *Options) { o.speakerLabels = enabled } }

// WithStreamEndOfTurnSilenceMs sets the silence threshold (ms) before AssemblyAI emits an
// end-of-turn Turn event on a streaming session.
func WithStreamEndOfTurnSilenceMs(ms int) Option {
	return func(o *Options) { o.streamEndOfTurnSilenceMs = &ms }
}

// WithStreamSpeechModel overrides the streaming speech model.
func WithStreamSpeechModel(m string) Option { return func(o *Options) { o.streamSpeechModel = m } }

// WithStreamFormatTurns toggles automatic punctuation/casing on streaming turn transcripts.
// Defaults to true.
func WithStreamFormatTurns(enabled bool) Option {
	return func(o *Options) { o.streamFormatTurns = &enabled }
}

// WithStreamEndOfTurnConfidenceThreshold sets the confidence threshold (0.0–1.0) for
// end-of-turn detection.
func WithStreamEndOfTurnConfidenceThreshold(threshold float64) Option {
	return func(o *Options) { o.streamEndOfTurnConfidenceThreshold = &threshold }
}

// WithStreamMaxTurnSilenceMs caps the longest silence (ms) within a turn before AssemblyAI
// forces end-of-turn.
func WithStreamMaxTurnSilenceMs(ms int) Option {
	return func(o *Options) { o.streamMaxTurnSilence = &ms }
}

// WithStreamKeyterms boosts recognition of specific words or phrases during streaming.
func WithStreamKeyterms(terms ...string) Option {
	return func(o *Options) { o.streamKeyterms = terms }
}

// WithStreamPunctuationFilter toggles AssemblyAI's punctuation filter on streaming transcripts.
func WithStreamPunctuationFilter(enabled bool) Option {
	return func(o *Options) { o.streamPunctuationFilter = &enabled }
}

// WithStreamWordFinalizationMaxWaitMs caps how long AssemblyAI waits before finalising the
// last word of a turn.
func WithStreamWordFinalizationMaxWaitMs(ms int) Option {
	return func(o *Options) { o.streamWordFinalizationMaxWaitMs = &ms }
}

// WithStreamExtraSessionInformation enables additional session metadata events.
func WithStreamExtraSessionInformation(enabled bool) Option {
	return func(o *Options) { o.streamExtraSessionInformation = &enabled }
}

// Client implements [stt.SpeechToText] against the AssemblyAI API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewSpeechToText constructs an AssemblyAI speech-to-text client. The returned
// [stt.SpeechToText] is wrapped with [stt.WithTracing], so callers always get
// tracing spans and metrics.
func NewSpeechToText(opts ...Option) stt.SpeechToText {
	options := Options{
		pollInterval:    3 * time.Second,
		maxPollDuration: 5 * time.Minute,
	}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
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

// SupportsStreaming reports true; AssemblyAI supports v3 Universal-Streaming.
func (c *Client) SupportsStreaming() bool { return true }

type uploadResponse struct {
	UploadURL string `json:"upload_url"`
}

type transcriptRequest struct {
	AudioURL      string `json:"audio_url"`
	LanguageCode  string `json:"language_code,omitempty"`
	SpeakerLabels bool   `json:"speaker_labels,omitempty"`
	SpeechModel   string `json:"speech_model,omitempty"`
}

type transcriptResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Text   string `json:"text"`
	Words  []struct {
		Text       string  `json:"text"`
		Start      int64   `json:"start"`
		End        int64   `json:"end"`
		Confidence float64 `json:"confidence"`
		Speaker    string  `json:"speaker,omitempty"`
	} `json:"words"`
	Utterances []struct {
		Text       string  `json:"text"`
		Start      int64   `json:"start"`
		End        int64   `json:"end"`
		Confidence float64 `json:"confidence"`
		Speaker    string  `json:"speaker"`
	} `json:"utterances"`
	AudioDuration float64 `json:"audio_duration"`
	Error         string  `json:"error"`
}

// Transcribe converts audio to text by uploading + polling.
func (c *Client) Transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	opts := stt.Options{}
	for _, opt := range options {
		opt(&opts)
	}

	uploadURL, err := c.upload(ctx, audioFile)
	if err != nil {
		return nil, err
	}

	transcriptReq := transcriptRequest{
		AudioURL:      uploadURL,
		SpeakerLabels: c.options.speakerLabels,
	}

	apiModel := c.options.model.APIModel
	if apiModel != "" && apiModel != "best" {
		transcriptReq.SpeechModel = apiModel
	}

	if opts.Language != "" {
		transcriptReq.LanguageCode = opts.Language
	}

	transcriptID, err := c.createTranscript(ctx, transcriptReq)
	if err != nil {
		return nil, err
	}

	result, err := c.pollTranscript(ctx, transcriptID)
	if err != nil {
		return nil, err
	}

	return c.mapResponse(result), nil
}

// Translate is not supported by AssemblyAI.
func (c *Client) Translate(
	ctx context.Context,
	audioFile []byte,
	options ...stt.Option,
) (*stt.Response, error) {
	return nil, fmt.Errorf("assemblyai does not support translation")
}

func (c *Client) upload(ctx context.Context, audioFile []byte) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/upload", bytes.NewReader(audioFile),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Authorization", c.options.apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp uploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal upload response: %w", err)
	}
	return uploadResp.UploadURL, nil
}

func (c *Client) createTranscript(ctx context.Context, transcriptReq transcriptRequest) (string, error) {
	jsonBody, err := json.Marshal(transcriptReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transcript request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/transcript", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create transcript request: %w", err)
	}
	req.Header.Set("Authorization", c.options.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create transcript: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read transcript response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("transcript API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tr transcriptResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("failed to unmarshal transcript response: %w", err)
	}
	return tr.ID, nil
}

func (c *Client) pollTranscript(ctx context.Context, transcriptID string) (*transcriptResponse, error) {
	deadline := time.Now().Add(c.options.maxPollDuration)
	pollURL := fmt.Sprintf("%s/transcript/%s", c.baseURL, transcriptID)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create poll request: %w", err)
		}
		req.Header.Set("Authorization", c.options.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to poll transcript: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read poll response: %w", err)
		}

		var result transcriptResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal poll response: %w", err)
		}

		switch result.Status {
		case "completed":
			return &result, nil
		case "error":
			return nil, fmt.Errorf("transcription failed: %s", result.Error)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.options.pollInterval):
		}
	}

	return nil, fmt.Errorf("transcription timed out after %s", c.options.maxPollDuration)
}

func (c *Client) mapResponse(result *transcriptResponse) *stt.Response {
	resp := &stt.Response{
		Text:     result.Text,
		Duration: result.AudioDuration,
		Model:    c.options.model.APIModel,
		Usage:    stt.Usage{DurationSec: result.AudioDuration},
	}

	words := make([]stt.Word, len(result.Words))
	for i, w := range result.Words {
		words[i] = stt.Word{
			Word:  w.Text,
			Start: float64(w.Start) / 1000.0,
			End:   float64(w.End) / 1000.0,
		}
	}
	resp.Words = words

	return resp
}

// StreamTranscribe opens an AssemblyAI v3 Universal-Streaming WebSocket. Reader parses
// Turn frames into stt.StreamResult; writer forwards audio frames as binary messages.
// Auth is via query-param token (v3 convention).
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
	endOfTurn := streamDefaultEndOfTurnSilenceMs
	if c.options.streamEndOfTurnSilenceMs != nil {
		endOfTurn = *c.options.streamEndOfTurnSilenceMs
	}

	speechModel := c.options.streamSpeechModel
	if speechModel == "" {
		speechModel = c.options.model.APIModel
	}
	if speechModel == "" {
		speechModel = "universal-streaming-english"
	}
	formatTurns := true
	if c.options.streamFormatTurns != nil {
		formatTurns = *c.options.streamFormatTurns
	}
	q := url.Values{}
	q.Set("token", c.options.apiKey)
	q.Set("sample_rate", strconv.Itoa(sampleRate))
	q.Set("encoding", "pcm_s16le")
	q.Set("format_turns", strconv.FormatBool(formatTurns))
	q.Set("speech_model", speechModel)
	q.Set("min_end_of_turn_silence_when_confident", strconv.Itoa(endOfTurn))
	if c.options.streamEndOfTurnConfidenceThreshold != nil {
		q.Set("end_of_turn_confidence_threshold",
			strconv.FormatFloat(*c.options.streamEndOfTurnConfidenceThreshold, 'f', -1, 64))
	}
	if c.options.streamMaxTurnSilence != nil {
		q.Set("max_turn_silence", strconv.Itoa(*c.options.streamMaxTurnSilence))
	}
	if c.options.streamPunctuationFilter != nil {
		q.Set("punctuation_filter", strconv.FormatBool(*c.options.streamPunctuationFilter))
	}
	if c.options.streamWordFinalizationMaxWaitMs != nil {
		q.Set("word_finalization_max_wait_time",
			strconv.Itoa(*c.options.streamWordFinalizationMaxWaitMs))
	}
	if c.options.streamExtraSessionInformation != nil {
		q.Set("enable_extra_session_information",
			strconv.FormatBool(*c.options.streamExtraSessionInformation))
	}
	for _, kt := range c.options.streamKeyterms {
		q.Add("keyterms_prompt", kt)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "streaming.assemblyai.com",
		Path:     "/v3/ws",
		RawQuery: q.Encode(),
	}

	dialer := websocket.Dialer{HandshakeTimeout: streamHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, u.String(), nil)
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

	minChunkBytes := sampleRate * 2 * streamMinChunkMs / 1000

	go runReader(conn, out, done)
	go runWriter(ctx, conn, audio, out, done, send, minChunkBytes)

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
	minChunkBytes int,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 0, minChunkBytes*2)
	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		err := send(websocket.BinaryMessage, buf)
		buf = buf[:0]
		return err
	}

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			_ = flush()
			return
		case frame, ok := <-audio:
			if !ok {
				_ = flush()
				return
			}
			buf = append(buf, frame...)
			if len(buf) >= minChunkBytes {
				if err := flush(); err != nil {
					out <- stt.StreamResult{Error: err}
					_ = conn.Close()
					<-done
					return
				}
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
	Type            string  `json:"type"`
	Transcript      string  `json:"transcript"`
	EndOfTurn       bool    `json:"end_of_turn"`
	TurnIsFormatted bool    `json:"turn_is_formatted"`
	EndOfTurnConf   float64 `json:"end_of_turn_confidence"`
	Words           []struct {
		Text        string  `json:"text"`
		Start       int64   `json:"start"`
		End         int64   `json:"end"`
		Confidence  float64 `json:"confidence"`
		WordIsFinal bool    `json:"word_is_final"`
	} `json:"words"`
}

func parseStream(raw []byte) (stt.StreamResult, bool) {
	var resp streamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return stt.StreamResult{}, false
	}
	if resp.Type != "Turn" {
		return stt.StreamResult{}, false
	}
	if resp.Transcript == "" {
		return stt.StreamResult{}, false
	}
	words := make([]stt.Word, len(resp.Words))
	for i, w := range resp.Words {
		words[i] = stt.Word{
			Word:  w.Text,
			Start: float64(w.Start) / 1000.0,
			End:   float64(w.End) / 1000.0,
		}
	}
	conf := resp.EndOfTurnConf
	if conf == 0 && len(resp.Words) > 0 {
		conf = resp.Words[len(resp.Words)-1].Confidence
	}
	return stt.StreamResult{
		Text:       resp.Transcript,
		Confidence: conf,
		IsFinal:    resp.EndOfTurn,
		WordCount:  len(resp.Words),
		Words:      words,
	}, true
}
