// Package deepgram provides a Deepgram Aura implementation of the [tts.Generation] interface.
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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
)

const (
	defaultBaseURL  = "https://api.deepgram.com/v1"
	defaultModelStr = "aura-asteria-en"
)

// Options configures the Deepgram TTS client.
type Options struct {
	apiKey     string
	model      model.AudioModel
	timeout    *time.Duration
	baseURL    string
	modelName  string
	encoding   string
	container  string
	sampleRate int
	bitRate    int
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
	m model.AudioModel,
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
// precedence over [WithModel].
func WithModelName(
	name string,
) Option {
	return func(o *Options) { o.modelName = name }
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

	resolved := defaultModelStr
	if options.model.APIModel != "" {
		resolved = options.model.APIModel
	}
	if options.modelName != "" {
		resolved = options.modelName
	}

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
func (c *Client) Model() model.AudioModel { return c.options.model }

type ttsRequest struct {
	Text string `json:"text"`
}

type errorResponse struct {
	ErrCode string `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

func (c *Client) buildURL() string {
	q := url.Values{}
	q.Set("model", c.resolved)
	if c.options.encoding != "" {
		q.Set("encoding", c.options.encoding)
	}
	if c.options.container != "" {
		q.Set("container", c.options.container)
	}
	if c.options.sampleRate != 0 {
		q.Set("sample_rate", strconv.Itoa(c.options.sampleRate))
	}
	if c.options.bitRate != 0 {
		q.Set("bit_rate", strconv.Itoa(c.options.bitRate))
	}
	return fmt.Sprintf("%s/speak?%s", c.options.baseURL, q.Encode())
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

// GenerateAudio creates audio from text and returns the complete audio data.
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

// StreamAudio opens a WebSocket to Deepgram's /v1/speak endpoint, sends a
// Speak/Flush sequence, and emits audio chunks as binary frames arrive.
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

	chunkChan := make(chan tts.Chunk, 10)
	go runStreamReader(ctx, conn, chunkChan, send, true)
	return chunkChan, nil
}

// dialStreamWS opens the WS to Deepgram's /v1/speak endpoint and returns the
// connection along with a goroutine-safe send function.
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
// each piece received on textIn (with an immediate Flush so audio for that piece
// starts generating). Closing textIn sends Close, which triggers a final Flushed
// frame before the connection terminates. Implements [tts.StreamingTextProvider].
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

	chunkChan := make(chan tts.Chunk, 10)

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = send([]byte(`{"type":"Close"}`))
				return
			case piece, ok := <-textIn:
				if !ok {
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
					select {
					case chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to marshal speak: %w", mErr)}:
					case <-ctx.Done():
					}
					_ = send([]byte(`{"type":"Close"}`))
					return
				}
				if sErr := send(speak); sErr != nil {
					select {
					case chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to send speak: %w", sErr)}:
					case <-ctx.Done():
					}
					return
				}
				if fErr := send([]byte(`{"type":"Flush"}`)); fErr != nil {
					select {
					case chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to send flush: %w", fErr)}:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()

	go runStreamReader(ctx, conn, chunkChan, send, false)
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

func (c *Client) buildStreamURL() (string, error) {
	base, err := url.Parse(c.options.baseURL)
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if base.Scheme == "http" {
		scheme = "ws"
	}
	q := url.Values{}
	q.Set("model", c.resolved)

	encoding := c.options.encoding
	if encoding == "" {
		encoding = "linear16"
	}
	q.Set("encoding", encoding)
	q.Set("container", "none")

	if c.options.sampleRate != 0 {
		q.Set("sample_rate", strconv.Itoa(c.options.sampleRate))
	}
	if c.options.bitRate != 0 {
		q.Set("bit_rate", strconv.Itoa(c.options.bitRate))
	}

	u := url.URL{
		Scheme:   scheme,
		Host:     base.Host,
		Path:     base.Path + "/speak",
		RawQuery: q.Encode(),
	}
	return u.String(), nil
}

func runStreamReader(
	ctx context.Context,
	conn *websocket.Conn,
	out chan<- tts.Chunk,
	send func([]byte) error,
	closeOnFlushed bool,
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
				switch srv.Type {
				case "Flushed":
					if !closeOnFlushed {
						// streaming-text path: each Flushed is just a per-chunk
						// boundary. Keep the connection open until the caller
						// closes textIn (which sends a Close frame).
						continue
					}
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
				case "Warning":
					// non-fatal; continue receiving audio
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
