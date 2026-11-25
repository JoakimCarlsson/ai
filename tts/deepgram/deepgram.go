// Package deepgram provides a Deepgram Aura and Flux TTS implementation of the
// [tts.Generation] interface. Aura models use /v1/speak and Flux models
// /v2/speak.
package deepgram

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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/tts"
)

const (
	defaultBaseURL  = "https://api.deepgram.com/v1"
	defaultModelStr = "aura-asteria-en"
)

// modelFamilies maps each catalog model in [Models], which names a family of
// voices, to the prefix and default voice that make a full Deepgram model
// string such as "aura-2-thalia-en".
var modelFamilies = map[string]struct{ prefix, defaultVoice string }{
	Aura1:   {prefix: "aura-", defaultVoice: "asteria-en"},
	Aura2:   {prefix: "aura-2-", defaultVoice: "thalia-en"},
	FluxTTS: {prefix: "flux-", defaultVoice: "haley-en"},
}

// Options configures the Deepgram TTS client.
type Options struct {
	apiKey     string
	model      tts.AudioModel
	timeout    *time.Duration
	baseURL    string
	modelName  string
	voice      string
	encoding   string
	container  string
	sampleRate int
	bitRate    int
	speed      *float64
	mipOptOut  bool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Deepgram.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the TTS model from the model package.
func WithModel(
	m tts.AudioModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBaseURL sets a custom base URL for the Deepgram API.
func WithBaseURL(
	baseURL string,
) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithModelName overrides the model identifier (e.g. "aura-2-thalia-en"). Takes
// precedence over [WithModel] and [WithVoice].
func WithModelName(
	name string,
) Option {
	return func(o *Options) { o.modelName = name }
}

// WithVoice picks the voice within the family [WithModel] selects, as voice
// and language (e.g. "thalia-en" with [Aura2] gives "aura-2-thalia-en").
// Without it each family uses a default voice.
func WithVoice(voice string) Option {
	return func(o *Options) { o.voice = voice }
}

// WithSpeed sets the speaking rate, from 0.7 to 1.5 on Aura-2; 1.0 is normal.
func WithSpeed(speed float64) Option {
	return func(o *Options) { o.speed = &speed }
}

// WithMIPOptOut opts the requests out of the Deepgram Model Improvement
// Program, so the audio is not kept for training.
func WithMIPOptOut(optOut bool) Option {
	return func(o *Options) { o.mipOptOut = optOut }
}

// WithEncoding sets the audio encoding (e.g. "mp3", "linear16", "mulaw", "alaw",
// "opus", "flac", "aac"). Default is "mp3".
func WithEncoding(encoding string) Option {
	return func(o *Options) { o.encoding = encoding }
}

// WithContainer sets the audio container (e.g. "wav", "ogg", "none").
func WithContainer(container string) Option {
	return func(o *Options) { o.container = container }
}

// WithSampleRate sets the audio sample rate in Hz.
func WithSampleRate(
	rate int,
) Option {
	return func(o *Options) { o.sampleRate = rate }
}

// WithBitRate sets the audio bit rate in bps for compressed encodings.
func WithBitRate(
	rate int,
) Option {
	return func(o *Options) { o.bitRate = rate }
}

// Client implements [tts.Generation] against the Deepgram Aura TTS API.
type Client struct {
	options    Options
	httpClient *http.Client
	resolved   string
}

// NewGeneration constructs a Deepgram Aura TTS client.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{baseURL: defaultBaseURL}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	resolved := resolveModelName(options)

	return tts.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		resolved:   resolved,
	}, tts.TracingAttrs{
		OutputFormat: options.encoding,
		SampleRate:   options.sampleRate,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() tts.AudioModel { return c.options.model }

type ttsRequest struct {
	Text string `json:"text"`
}

type errorResponse struct {
	ErrCode string `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// resolveModelName returns the full Deepgram model string for options:
// [WithModelName] if set, else the [WithModel] family joined with the
// [WithVoice] voice (or the family default), else the model's APIModel as is,
// else the package default.
func resolveModelName(options Options) string {
	if options.modelName != "" {
		return options.modelName
	}
	apiModel := options.model.APIModel
	if apiModel == "" {
		return defaultModelStr
	}
	family, ok := modelFamilies[apiModel]
	if !ok {
		return apiModel
	}
	voice := options.voice
	if voice == "" {
		voice = family.defaultVoice
	}
	return family.prefix + voice
}

// isFlux reports whether the resolved model is a Flux TTS model, which
// Deepgram serves on /v2/speak rather than /v1/speak.
func (c *Client) isFlux() bool {
	return strings.HasPrefix(c.resolved, "flux-")
}

// speakBase returns the API base for the resolved model: the configured base
// URL, with its /v1 version swapped for /v2 (or /v2 appended) for Flux.
func (c *Client) speakBase() string {
	if !c.isFlux() {
		return c.options.baseURL
	}
	return strings.TrimSuffix(c.options.baseURL, "/v1") + "/v2"
}

// setCommonQuery adds the model and the options shared by the REST and
// WebSocket endpoints to q.
func (c *Client) setCommonQuery(q url.Values) {
	q.Set("model", c.resolved)
	if c.options.sampleRate != 0 {
		q.Set("sample_rate", strconv.Itoa(c.options.sampleRate))
	}
	if c.options.bitRate != 0 && !c.isFlux() {
		q.Set("bit_rate", strconv.Itoa(c.options.bitRate))
	}
	if c.options.speed != nil {
		q.Set("speed", strconv.FormatFloat(*c.options.speed, 'f', -1, 64))
	}
	if c.options.mipOptOut {
		q.Set("mip_opt_out", "true")
	}
}

// buildURL returns the REST /speak address with the configured query.
func (c *Client) buildURL() string {
	q := url.Values{}
	c.setCommonQuery(q)
	if c.options.bitRate != 0 && c.isFlux() {
		q.Set("bit_rate", strconv.Itoa(c.options.bitRate))
	}
	if c.options.encoding != "" {
		q.Set("encoding", c.options.encoding)
	}
	if c.options.container != "" {
		q.Set("container", c.options.container)
	}
	return fmt.Sprintf("%s/speak?%s", c.speakBase(), q.Encode())
}

func (c *Client) newRequest(
	ctx context.Context,
	text string,
) (*http.Request, error) {
	body, err := json.Marshal(ttsRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.buildURL(),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.options.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// GenerateAudio creates audio from text and returns the complete audio data,
// from /v1/speak for Aura or /v2/speak for Flux.
func (c *Client) GenerateAudio(
	ctx context.Context,
	text string,
	_ ...tts.GenerationOption,
) (*tts.Response, error) {
	req, err := c.newRequest(ctx, text)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	charCount := int64(len(text))
	if v := resp.Header.Get("dg-char-count"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			charCount = n
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}

	return &tts.Response{
		AudioData:   audioData,
		ContentType: contentType,
		Usage:       tts.Usage{Characters: charCount},
		Model:       c.resolved,
	}, nil
}

// StreamAudio opens a WebSocket to Deepgram's /speak endpoint (/v2/speak for
// Flux), sends a Speak/Flush sequence, and emits audio chunks as binary frames
// arrive, ending with a Done chunk once the turn's audio is complete.
//
// WS streaming requires a raw audio encoding (linear16, mulaw, alaw); if no
// encoding is configured the implementation defaults to linear16 for the WS
// path. Set [WithEncoding] explicitly to override.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	conn, send, err := c.dialStreamWS(ctx)
	if err != nil {
		return nil, err
	}

	speak, err := json.Marshal(speakMessage{Type: "Speak", Text: text})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to marshal speak message: %w", err)
	}
	if err := send(speak); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send speak: %w", err)
	}
	if err := send([]byte(`{"type":"Flush"}`)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send flush: %w", err)
	}

	finalFrame := "Flushed"
	if c.isFlux() {
		finalFrame = "SpeechMetadata"
	}
	chunkChan := make(chan tts.Chunk, 10)
	go runStreamReader(ctx, conn, chunkChan, send, finalFrame)
	return chunkChan, nil
}

// dialStreamWS opens the WS to Deepgram's /speak endpoint (/v2/speak for
// Flux) and returns the connection along with a goroutine-safe send function.
func (c *Client) dialStreamWS(
	ctx context.Context,
) (*websocket.Conn, func([]byte) error, error) {
	wsURL, err := c.buildStreamURL()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build ws url: %w", err)
	}

	hdr := http.Header{}
	hdr.Set("Authorization", "Token "+c.options.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: wsHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, wsURL, hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial websocket: %w", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	})

	var writeMu sync.Mutex
	send := func(data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	return conn, send, nil
}

// StreamAudioFromText opens a WebSocket to Deepgram and sends a Speak frame for
// each piece received on textIn. On Aura each piece is flushed at once so its
// audio starts generating, and closing textIn sends Close. On Flux the server
// streams audio as text arrives, so the pieces form one turn; closing textIn
// sends Flush to end it and Close, and the stream ends with a Done chunk once
// the server reports the session totals. Implements
// [tts.StreamingTextProvider].
//
// Like [StreamAudio], this requires a raw audio encoding (linear16, mulaw, alaw);
// the WS endpoint does not accept compressed encodings.
func (c *Client) StreamAudioFromText(
	ctx context.Context,
	textIn <-chan string,
	_ ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	conn, send, err := c.dialStreamWS(ctx)
	if err != nil {
		return nil, err
	}

	flux := c.isFlux()
	chunkChan := make(chan tts.Chunk, 10)
	fail := func(err error) {
		select {
		case chunkChan <- tts.Chunk{Error: err}:
		case <-ctx.Done():
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = send([]byte(`{"type":"Close"}`))
				return
			case piece, ok := <-textIn:
				if !ok {
					if flux {
						_ = send([]byte(`{"type":"Flush"}`))
					}
					_ = send([]byte(`{"type":"Close"}`))
					return
				}
				if piece == "" {
					continue
				}
				speak, mErr := json.Marshal(
					speakMessage{Type: "Speak", Text: piece},
				)
				if mErr != nil {
					fail(fmt.Errorf("failed to marshal speak: %w", mErr))
					_ = send([]byte(`{"type":"Close"}`))
					return
				}
				if sErr := send(speak); sErr != nil {
					fail(fmt.Errorf("failed to send speak: %w", sErr))
					return
				}
				if flux {
					continue
				}
				if fErr := send([]byte(`{"type":"Flush"}`)); fErr != nil {
					fail(fmt.Errorf("failed to send flush: %w", fErr))
					return
				}
			}
		}
	}()

	finalFrame := ""
	if flux {
		finalFrame = "SessionMetadata"
	}
	go runStreamReader(ctx, conn, chunkChan, send, finalFrame)
	return chunkChan, nil
}

const (
	wsHandshakeTimeout = 10 * time.Second
	wsReadDeadline     = 60 * time.Second
)

type speakMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type serverFrame struct {
	Type        string `json:"type"`
	SequenceID  int    `json:"sequence_id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// buildStreamURL returns the WebSocket /speak address with the configured
// query, defaulting to raw linear16 audio, which the WebSocket requires.
// Flux rejects the container parameter, so it is sent to Aura only.
func (c *Client) buildStreamURL() (string, error) {
	base, err := url.Parse(c.speakBase())
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if base.Scheme == "http" {
		scheme = "ws"
	}
	q := url.Values{}
	c.setCommonQuery(q)

	encoding := c.options.encoding
	if encoding == "" {
		encoding = "linear16"
	}
	q.Set("encoding", encoding)
	if !c.isFlux() {
		q.Set("container", "none")
	}

	u := url.URL{
		Scheme:   scheme,
		Host:     base.Host,
		Path:     base.Path + "/speak",
		RawQuery: q.Encode(),
	}
	return u.String(), nil
}

// runStreamReader forwards binary audio frames from conn to out until the
// connection closes, an Error frame arrives, or ctx is done, then closes out
// and conn. When a text frame of type finalFrame arrives it sends a Done chunk
// and closes the session; an empty finalFrame waits for the server to close.
func runStreamReader(
	ctx context.Context,
	conn *websocket.Conn,
	out chan<- tts.Chunk,
	send func([]byte) error,
	finalFrame string,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if !isCleanWSClose(err) && !errors.Is(err, net.ErrClosed) {
					select {
					case out <- tts.Chunk{Error: err}:
					case <-ctx.Done():
					}
				}
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

			switch msgType {
			case websocket.BinaryMessage:
				data := make([]byte, len(msg))
				copy(data, msg)
				select {
				case out <- tts.Chunk{Data: data}:
				case <-ctx.Done():
					return
				}
			case websocket.TextMessage:
				var srv serverFrame
				if err := json.Unmarshal(msg, &srv); err != nil {
					continue
				}
				switch {
				case srv.Type == "Error":
					select {
					case out <- tts.Chunk{Error: fmt.Errorf("deepgram: %s: %s", srv.Code, srv.Description)}:
					case <-ctx.Done():
					}
					return
				case finalFrame != "" && srv.Type == finalFrame:
					select {
					case out <- tts.Chunk{Done: true}:
					case <-ctx.Done():
					}
					_ = send([]byte(`{"type":"Close"}`))
					_ = conn.WriteControl(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(
							websocket.CloseNormalClosure,
							"",
						),
						time.Now().Add(2*time.Second),
					)
					return
				}
			}
		}
	}()

	select {
	case <-readDone:
	case <-ctx.Done():
		_ = send([]byte(`{"type":"Close"}`))
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(2*time.Second),
		)
		<-readDone
	}
}

func isCleanWSClose(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}

// ListVoices returns the static set of Aura voices known to deps/ai. Deepgram
// does not expose a public list-voices endpoint.
func (c *Client) ListVoices(_ context.Context) ([]tts.Voice, error) {
	return []tts.Voice{
		{VoiceID: "aura-2-thalia-en", Name: "Thalia", Category: "aura-2"},
		{VoiceID: "aura-2-andromeda-en", Name: "Andromeda", Category: "aura-2"},
		{VoiceID: "aura-2-helena-en", Name: "Helena", Category: "aura-2"},
		{VoiceID: "aura-asteria-en", Name: "Asteria", Category: "aura"},
		{VoiceID: "aura-luna-en", Name: "Luna", Category: "aura"},
		{VoiceID: "aura-stella-en", Name: "Stella", Category: "aura"},
		{VoiceID: "aura-zeus-en", Name: "Zeus", Category: "aura"},
	}, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(
			"audio generation failed with status %d",
			resp.StatusCode,
		)
	}

	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err == nil &&
		errResp.ErrMsg != "" {
		return fmt.Errorf("audio generation failed: %s", errResp.ErrMsg)
	}

	return fmt.Errorf(
		"audio generation failed with status %d: %s",
		resp.StatusCode,
		string(body),
	)
}
