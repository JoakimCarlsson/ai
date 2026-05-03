package stt

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
)

const (
	elevenLabsStreamReadDeadline     = 30 * time.Second
	elevenLabsStreamHandshakeTimeout = 10 * time.Second
)

type elevenLabsOptions struct {
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

// ElevenLabsOption configures ElevenLabs Scribe-specific transcription behavior.
type ElevenLabsOption func(*elevenLabsOptions)

type elevenLabsClient struct {
	providerOptions transcriptionClientOptions
	options         elevenLabsOptions
	httpClient      *http.Client
	baseURL         string
}

// ElevenLabsClient is the ElevenLabs Scribe implementation of SpeechToTextClient.
type ElevenLabsClient SpeechToTextClient

type elScribeResponse struct {
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

func newElevenLabsClient(
	opts transcriptionClientOptions,
) ElevenLabsClient {
	elOpts := elevenLabsOptions{}
	for _, o := range opts.elevenLabsOptions {
		o(&elOpts)
	}

	timeout := 120 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &elevenLabsClient{
		providerOptions: opts,
		options:         elOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.elevenlabs.io/v1",
	}
}

func (e *elevenLabsClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField(
		"model_id",
		e.providerOptions.model.APIModel,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to write model_id field: %w",
			err,
		)
	}

	if opts.Language != "" {
		if err := writer.WriteField(
			"language_code",
			opts.Language,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write language_code field: %w",
				err,
			)
		}
	}

	if e.options.diarize != nil && *e.options.diarize {
		if err := writer.WriteField(
			"diarize",
			"true",
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write diarize field: %w",
				err,
			)
		}
	}

	if e.options.numSpeakers != nil {
		if err := writer.WriteField(
			"num_speakers",
			fmt.Sprintf("%d", *e.options.numSpeakers),
		); err != nil {
			return nil, fmt.Errorf(
				"failed to write num_speakers field: %w",
				err,
			)
		}
	}

	granularity := "word"
	if e.options.timestampGranularity != "" {
		granularity = e.options.timestampGranularity
	}
	if err := writer.WriteField(
		"timestamps_granularity",
		granularity,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to write timestamps field: %w",
			err,
		)
	}

	filename := "tts.mp3"
	if opts.Filename != "" {
		filename = opts.Filename
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create form file: %w",
			err,
		)
	}
	if _, err := part.Write(audioFile); err != nil {
		return nil, fmt.Errorf(
			"failed to write audio data: %w",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf(
			"failed to close multipart writer: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		e.baseURL+"/speech-to-text",
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create STT request: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)
	req.Header.Set(
		"xi-api-key",
		e.providerOptions.apiKey,
	)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make STT request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read STT response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"STT API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var elResp elScribeResponse
	if err := json.Unmarshal(body, &elResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal STT response: %w",
			err,
		)
	}

	return e.mapResponse(&elResp), nil
}

func (e *elevenLabsClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"elevenlabs scribe does not support translation",
	)
}

func (e *elevenLabsClient) mapResponse(
	elResp *elScribeResponse,
) *Response {
	result := &Response{
		Text:     elResp.Text,
		Language: elResp.LanguageCode,
		Model:    e.providerOptions.model.APIModel,
	}

	var words []Word
	for _, w := range elResp.Words {
		if w.Type != "word" {
			continue
		}
		words = append(words, Word{
			Word:  w.Text,
			Start: w.Start,
			End:   w.End,
		})
	}
	result.Words = words

	return result
}

// WithElevenLabsDiarize enables speaker diarization.
func WithElevenLabsDiarize(
	enabled bool,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.diarize = &enabled
	}
}

// WithElevenLabsNumSpeakers sets the expected number of speakers (0-32).
func WithElevenLabsNumSpeakers(
	n int,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.numSpeakers = &n
	}
}

// WithElevenLabsTimestampGranularity sets timestamp level ("none", "word", "character").
func WithElevenLabsTimestampGranularity(
	granularity string,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.timestampGranularity = granularity
	}
}

// WithElevenLabsTagAudioEvents enables audio event detection (laughter, music, etc.).
func WithElevenLabsTagAudioEvents(
	enabled bool,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.tagAudioEvents = &enabled
	}
}

// WithElevenLabsStreamVADSilenceMs sets the silence window (ms) ElevenLabs'
// VAD waits before emitting committed_transcript on a streaming session.
// Streaming-only; ignored on batch Transcribe calls.
func WithElevenLabsStreamVADSilenceMs(
	ms int,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamVADSilenceMs = &ms
	}
}

// WithElevenLabsStreamLanguageCode sets the language hint for streaming
// sessions. Streaming-only; ignored on batch Transcribe calls.
func WithElevenLabsStreamLanguageCode(
	code string,
) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamLanguageCode = code
	}
}

// WithElevenLabsStreamKeyterms boosts recognition of specific words or
// phrases during a streaming session.
func WithElevenLabsStreamKeyterms(terms ...string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamKeyterms = terms
	}
}

// WithElevenLabsStreamNoVerbatim strips filler words ("um", "uh", …) from
// streaming transcripts.
func WithElevenLabsStreamNoVerbatim(enabled bool) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamNoVerbatim = &enabled
	}
}

// WithElevenLabsStreamIncludeTimestamps emits word-level timing data on
// streaming transcripts.
func WithElevenLabsStreamIncludeTimestamps(enabled bool) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamIncludeTimestamps = &enabled
	}
}

// WithElevenLabsStreamIncludeLanguageDetection enables automatic language
// detection on streaming sessions.
func WithElevenLabsStreamIncludeLanguageDetection(enabled bool) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamIncludeLanguageDetect = &enabled
	}
}

// WithElevenLabsStreamVADThreshold sets VAD sensitivity (0.0–1.0) for
// streaming sessions.
func WithElevenLabsStreamVADThreshold(threshold float64) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamVADThreshold = &threshold
	}
}

// WithElevenLabsStreamMinSpeechDurationMs sets the minimum duration of
// speech (ms) before VAD considers it valid.
func WithElevenLabsStreamMinSpeechDurationMs(ms int) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamMinSpeechDurationMs = &ms
	}
}

// WithElevenLabsStreamMinSilenceDurationMs sets the minimum duration of
// silence (ms) before VAD declares end of speech.
func WithElevenLabsStreamMinSilenceDurationMs(ms int) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamMinSilenceDurationMs = &ms
	}
}

// WithElevenLabsStreamTimestampsGranularity sets timestamp resolution for
// streaming. Valid values: "none", "word", "character".
func WithElevenLabsStreamTimestampsGranularity(g string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamTimestampsGranularity = g
	}
}

// WithElevenLabsStreamDisableLogging opts the streaming session out of
// ElevenLabs' server-side logging.
func WithElevenLabsStreamDisableLogging(enabled bool) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.streamDisableLogging = &enabled
	}
}

// streamTranscribe opens an ElevenLabs Scribe v2 Realtime WebSocket. Audio
// frames are base64-encoded and wrapped in JSON input_audio_chunk events
// (per ElevenLabs' protocol — they don't accept raw binary). Reader parses
// partial_transcript / committed_transcript events into StreamResult.
func (e *elevenLabsClient) streamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	opts := Options{}
	for _, o := range options {
		o(&opts)
	}

	sampleRate := opts.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	lang := e.options.streamLanguageCode
	if lang == "" {
		lang = opts.Language
	}

	q := url.Values{}
	q.Set("audio_format", fmt.Sprintf("pcm_%d", sampleRate))
	if lang != "" {
		q.Set("language_code", lang)
	}
	for _, kt := range e.options.streamKeyterms {
		q.Add("keyterms", kt)
	}
	if e.options.streamNoVerbatim != nil {
		q.Set("no_verbatim", strconv.FormatBool(*e.options.streamNoVerbatim))
	}
	if e.options.streamIncludeTimestamps != nil {
		q.Set("include_timestamps", strconv.FormatBool(*e.options.streamIncludeTimestamps))
	}
	if e.options.streamIncludeLanguageDetect != nil {
		q.Set("include_language_detection",
			strconv.FormatBool(*e.options.streamIncludeLanguageDetect))
	}
	if e.options.streamVADThreshold != nil {
		q.Set("vad_threshold",
			strconv.FormatFloat(*e.options.streamVADThreshold, 'f', -1, 64))
	}
	if e.options.streamMinSpeechDurationMs != nil {
		q.Set("min_speech_duration_ms", strconv.Itoa(*e.options.streamMinSpeechDurationMs))
	}
	if e.options.streamMinSilenceDurationMs != nil {
		q.Set("min_silence_duration_ms", strconv.Itoa(*e.options.streamMinSilenceDurationMs))
	}
	if e.options.streamTimestampsGranularity != "" {
		q.Set("timestamps_granularity", e.options.streamTimestampsGranularity)
	}
	if e.options.streamDisableLogging != nil {
		q.Set("disable_logging", strconv.FormatBool(*e.options.streamDisableLogging))
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.elevenlabs.io",
		Path:     "/v1/speech-to-text/realtime",
		RawQuery: q.Encode(),
	}
	hdr := http.Header{}
	hdr.Set("xi-api-key", e.providerOptions.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: elevenLabsStreamHandshakeTimeout}
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

	_ = conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
	})

	go runElevenLabsReader(conn, out, done)
	go runElevenLabsWriter(ctx, conn, audio, out, done, send)

	return out, nil
}

func runElevenLabsReader(
	conn *websocket.Conn,
	out chan<- StreamResult,
	done chan<- struct{},
) {
	defer close(done)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !isCleanWSClose(err) && !errors.Is(err, net.ErrClosed) {
				out <- StreamResult{Error: err}
			}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
		pr, ok := parseElevenLabsStream(msg)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runElevenLabsWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- StreamResult,
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
				out <- StreamResult{Error: err}
				_ = conn.Close()
				<-done
				return
			}
		}
	}
}

type elevenLabsStreamResp struct {
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

func parseElevenLabsStream(raw []byte) (StreamResult, bool) {
	var resp elevenLabsStreamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return StreamResult{}, false
	}
	isFinal := false
	switch resp.MessageType {
	case "partial_transcript":
		isFinal = false
	case "committed_transcript", "committed_transcript_with_timestamps":
		isFinal = true
	default:
		return StreamResult{}, false
	}
	text := resp.Transcript
	if text == "" {
		text = resp.Text
	}
	if text == "" {
		return StreamResult{}, false
	}
	words := make([]Word, len(resp.Words))
	for i, w := range resp.Words {
		words[i] = Word{Word: w.Text, Start: w.Start, End: w.End}
	}
	return StreamResult{
		Text:       text,
		Confidence: resp.Confidence,
		IsFinal:    isFinal,
		WordCount:  len(resp.Words),
		Words:      words,
	}, true
}
