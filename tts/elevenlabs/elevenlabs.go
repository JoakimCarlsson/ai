// Package elevenlabs provides an ElevenLabs implementation of the [tts.Generation]
// interface. The concrete [Client] also implements [tts.ForcedAlignmentProvider],
// reachable via type assertion on the value returned from [NewGeneration].
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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tts"
)

const (
	defaultBaseURL      = "https://api.elevenlabs.io/v1"
	defaultVoiceID      = "EXAVITQu4vr4xnSDxMaL"
	defaultOutputFormat = "mp3_44100_128"
	defaultModelID      = "eleven_multilingual_v2"
)

// Options configures the ElevenLabs TTS client.
type Options struct {
	apiKey       string
	model        model.AudioModel
	timeout      *time.Duration
	baseURL      string
	voiceID      string
	outputFormat string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with ElevenLabs.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the TTS model.
func WithModel(
	m model.AudioModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request. Defaults to 10 minutes.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBaseURL sets a custom base URL for the ElevenLabs API.
func WithBaseURL(
	baseURL string,
) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithVoiceID sets the voice ID. Defaults to Rachel.
func WithVoiceID(
	voiceID string,
) Option {
	return func(o *Options) { o.voiceID = voiceID }
}

// WithOutputFormat sets the audio output format (e.g. "mp3_44100_128", "pcm_16000").
func WithOutputFormat(format string) Option {
	return func(o *Options) { o.outputFormat = format }
}

// Client implements [tts.Generation] and [tts.ForcedAlignmentProvider] against the
// ElevenLabs API.
type Client struct {
	apiKey       string
	model        model.AudioModel
	baseURL      string
	httpClient   *http.Client
	modelID      string
	voiceID      string
	outputFormat string
}

// NewGeneration constructs an ElevenLabs TTS client. The returned [tts.Generation]
// is wrapped with [tts.WithTracing]; the wrapper preserves [tts.ForcedAlignmentProvider]
// and [tts.StreamingTextProvider] support so type assertions against the returned
// value succeed.
func NewGeneration(opts ...Option) tts.Generation {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 10 * time.Minute
	if options.timeout != nil {
		timeout = *options.timeout
	}

	baseURL := defaultBaseURL
	if options.baseURL != "" {
		baseURL = options.baseURL
	}
	voiceID := defaultVoiceID
	if options.voiceID != "" {
		voiceID = options.voiceID
	}
	outputFormat := defaultOutputFormat
	if options.outputFormat != "" {
		outputFormat = options.outputFormat
	}
	modelID := defaultModelID
	if options.model.APIModel != "" {
		modelID = options.model.APIModel
	}

	return tts.WithTracing(&Client{
		apiKey:       options.apiKey,
		model:        options.model,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: timeout},
		modelID:      modelID,
		voiceID:      voiceID,
		outputFormat: outputFormat,
	}, tts.TracingAttrs{
		Voice:        voiceID,
		OutputFormat: outputFormat,
	})
}

// Model returns the configured TTS model.
func (c *Client) Model() model.AudioModel { return c.model }

type ttsRequest struct {
	Text          string         `json:"text"`
	ModelID       string         `json:"model_id"`
	VoiceSettings *voiceSettings `json:"voice_settings,omitempty"`
	OutputFormat  string         `json:"output_format,omitempty"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	SpeakerBoost    bool    `json:"use_speaker_boost,omitempty"`
}

type voiceResponse struct {
	Voices []voiceResponseItem `json:"voices"`
}

type voiceResponseItem struct {
	VoiceID                 string            `json:"voice_id"`
	Name                    string            `json:"name"`
	Category                string            `json:"category"`
	Description             string            `json:"description"`
	PreviewURL              string            `json:"preview_url"`
	Labels                  map[string]string `json:"labels"`
	HighQualityBaseModelIDs []string          `json:"high_quality_base_model_ids,omitempty"`
}

type errorResponse struct {
	Detail struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"detail"`
}

type alignmentResponse struct {
	Characters                 []string  `json:"characters"`
	CharacterStartTimesSeconds []float64 `json:"character_start_times_seconds"`
	CharacterEndTimesSeconds   []float64 `json:"character_end_times_seconds"`
}

type ttsWithTimestampsResponse struct {
	AudioBase64         string            `json:"audio_base64"`
	Alignment           alignmentResponse `json:"alignment"`
	NormalizedAlignment alignmentResponse `json:"normalized_alignment"`
}

type charAlignmentItem struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type wordAlignmentItem struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Loss  float64 `json:"loss"`
}

type forcedAlignmentResponse struct {
	Characters []charAlignmentItem `json:"characters"`
	Words      []wordAlignmentItem `json:"words"`
	Loss       float64             `json:"loss"`
}

func (c *Client) buildVoiceSettings(
	opts *tts.GenerationOptions,
) *voiceSettings {
	if opts.Stability == nil && opts.SimilarityBoost == nil &&
		opts.Style == nil && opts.SpeakerBoost == nil {
		return nil
	}
	vs := &voiceSettings{}
	if opts.Stability != nil {
		vs.Stability = *opts.Stability
	}
	if opts.SimilarityBoost != nil {
		vs.SimilarityBoost = *opts.SimilarityBoost
	}
	if opts.Style != nil {
		vs.Style = *opts.Style
	}
	if opts.SpeakerBoost != nil {
		vs.SpeakerBoost = *opts.SpeakerBoost
	}
	return vs
}

// GenerateAudio creates audio from text. If alignment is enabled, calls the
// /with-timestamps endpoint and populates the Alignment fields in the response.
func (c *Client) GenerateAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (*tts.Response, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}

	if opts.EnableAlignment {
		return c.generateWithTimestamps(ctx, text, opts)
	}
	return c.generateStandard(ctx, text, opts)
}

func (c *Client) generateStandard(
	ctx context.Context,
	text string,
	opts *tts.GenerationOptions,
) (*tts.Response, error) {
	outputFormat := c.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	reqBody := ttsRequest{
		Text:          text,
		ModelID:       c.modelID,
		VoiceSettings: c.buildVoiceSettings(opts),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/text-to-speech/%s?output_format=%s",
		c.baseURL, c.voiceID, outputFormat)
	if opts.OptimizeStreamingLatency != nil {
		url = fmt.Sprintf(
			"%s&optimize_streaming_latency=%d",
			url,
			*opts.OptimizeStreamingLatency,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

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

	charCount := int64(0)
	if charCountStr := resp.Header.Get(
		"x-character-count",
	); charCountStr != "" {
		if count, err := strconv.ParseInt(charCountStr, 10, 64); err == nil {
			charCount = count
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
		Model:       c.modelID,
	}, nil
}

func (c *Client) generateWithTimestamps(
	ctx context.Context,
	text string,
	opts *tts.GenerationOptions,
) (*tts.Response, error) {
	outputFormat := c.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	reqBody := ttsRequest{
		Text:          text,
		ModelID:       c.modelID,
		VoiceSettings: c.buildVoiceSettings(opts),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/text-to-speech/%s/with-timestamps?output_format=%s",
		c.baseURL, c.voiceID, outputFormat)
	if opts.OptimizeStreamingLatency != nil {
		url = fmt.Sprintf(
			"%s&optimize_streaming_latency=%d",
			url,
			*opts.OptimizeStreamingLatency,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var timestampsResp ttsWithTimestampsResponse
	if err := json.NewDecoder(resp.Body).Decode(&timestampsResp); err != nil {
		return nil, fmt.Errorf("failed to decode timestamps response: %w", err)
	}

	audioData, err := base64.StdEncoding.DecodeString(
		timestampsResp.AudioBase64,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 audio: %w", err)
	}

	contentType := contentTypeForFormat(outputFormat)

	return &tts.Response{
		AudioData:   audioData,
		ContentType: contentType,
		Usage:       tts.Usage{Characters: int64(len(text))},
		Model:       c.modelID,
		Alignment:   toAlignmentData(timestampsResp.Alignment),
		NormalizedAlignment: toAlignmentData(
			timestampsResp.NormalizedAlignment,
		),
	}, nil
}

// StreamAudio opens a WebSocket to ElevenLabs' stream-input endpoint and
// emits audio chunks (and alignment data when [tts.WithAlignmentEnabled] is set)
// as the server produces them. Lower TTFB than the HTTP streaming path; the
// caller-facing semantics are unchanged.
//
// Note: [tts.WithOptimizeStreamingLatency] is documented for ElevenLabs' HTTP
// endpoint only and has no effect on the WS path.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return c.streamWS(ctx, text, opts)
}

const (
	wsHandshakeTimeout = 10 * time.Second
	wsReadDeadline     = 60 * time.Second
	wsReadLimit        = 16 * 1024 * 1024
)

type wsBeginMessage struct {
	Text          string         `json:"text"`
	VoiceSettings *voiceSettings `json:"voice_settings,omitempty"`
}

type wsTextMessage struct {
	Text  string `json:"text"`
	Flush bool   `json:"flush,omitempty"`
}

type wsServerMessage struct {
	Audio               string             `json:"audio"`
	IsFinal             *bool              `json:"isFinal"`
	Alignment           *alignmentResponse `json:"alignment"`
	NormalizedAlignment *alignmentResponse `json:"normalizedAlignment"`
	Error               string             `json:"error"`
	Message             string             `json:"message"`
	Code                int                `json:"code"`
}

func (c *Client) streamWS(
	ctx context.Context,
	text string,
	opts *tts.GenerationOptions,
) (<-chan tts.Chunk, error) {
	conn, send, err := c.dialStreamWS(ctx, opts)
	if err != nil {
		return nil, err
	}

	textMsg, err := json.Marshal(wsTextMessage{Text: text + " "})
	if err != nil {
		_ = conn.Close()
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to marshal text: %w", err)}
		close(ch)
		return ch, nil
	}
	if err := send(websocket.TextMessage, textMsg); err != nil {
		_ = conn.Close()
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to send text: %w", err)}
		close(ch)
		return ch, nil
	}

	if err := send(websocket.TextMessage, []byte(`{"text":""}`)); err != nil {
		_ = conn.Close()
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to send eos: %w", err)}
		close(ch)
		return ch, nil
	}

	chunkChan := make(chan tts.Chunk, 10)
	go runStreamReader(ctx, conn, chunkChan)
	return chunkChan, nil
}

// dialStreamWS opens the WS to ElevenLabs, sends the BOS frame, and returns the
// connection along with a goroutine-safe send function. The caller is responsible
// for sending text frames, the EOS frame, and starting the reader.
func (c *Client) dialStreamWS(
	ctx context.Context,
	opts *tts.GenerationOptions,
) (*websocket.Conn, func(int, []byte) error, error) {
	outputFormat := c.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	wsURL, err := c.buildStreamURL(outputFormat, opts.EnableAlignment)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build ws url: %w", err)
	}

	hdr := http.Header{}
	hdr.Set("xi-api-key", c.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: wsHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, wsURL, hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial websocket: %w", err)
	}

	conn.SetReadLimit(wsReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	})

	var writeMu sync.Mutex
	send := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	bos, err := json.Marshal(wsBeginMessage{
		Text:          " ",
		VoiceSettings: c.buildVoiceSettings(opts),
	})
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to marshal bos: %w", err)
	}
	if err := send(websocket.TextMessage, bos); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to send bos: %w", err)
	}

	return conn, send, nil
}

// StreamAudioFromText opens a WebSocket to ElevenLabs and pumps text frames as
// they arrive on textIn. Closing textIn signals end-of-input; the WS sends an
// EOS frame and the audio channel closes once the server returns the final
// frame. This is the streaming-text path used to forward LLM deltas with low
// time-to-first-byte. Implements [tts.StreamingTextProvider].
func (c *Client) StreamAudioFromText(
	ctx context.Context,
	textIn <-chan string,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}

	conn, send, err := c.dialStreamWS(ctx, opts)
	if err != nil {
		return nil, err
	}

	chunkChan := make(chan tts.Chunk, 10)

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = send(websocket.TextMessage, []byte(`{"text":""}`))
				return
			case piece, ok := <-textIn:
				if !ok {
					_ = send(websocket.TextMessage, []byte(`{"text":""}`))
					return
				}
				if piece == "" {
					continue
				}
				if !strings.HasSuffix(piece, " ") {
					piece += " "
				}
				msg, mErr := json.Marshal(
					wsTextMessage{Text: piece, Flush: true},
				)
				if mErr != nil {
					select {
					case chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to marshal text: %w", mErr)}:
					case <-ctx.Done():
					}
					_ = send(websocket.TextMessage, []byte(`{"text":""}`))
					return
				}
				if sErr := send(websocket.TextMessage, msg); sErr != nil {
					select {
					case chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to send text: %w", sErr)}:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()

	go runStreamReader(ctx, conn, chunkChan)
	return chunkChan, nil
}

func (c *Client) buildStreamURL(
	outputFormat string,
	syncAlignment bool,
) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if base.Scheme == "http" {
		scheme = "ws"
	}
	q := url.Values{}
	q.Set("model_id", c.modelID)
	if outputFormat != "" {
		q.Set("output_format", outputFormat)
	}
	q.Set("inactivity_timeout", "20")
	q.Set("auto_mode", "false")
	if syncAlignment {
		q.Set("sync_alignment", "true")
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     base.Host,
		Path:     base.Path + "/text-to-speech/" + c.voiceID + "/stream-input",
		RawQuery: q.Encode(),
	}
	return u.String(), nil
}

func runStreamReader(
	ctx context.Context,
	conn *websocket.Conn,
	out chan<- tts.Chunk,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, msg, err := conn.ReadMessage()
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

			var srv wsServerMessage
			if err := json.Unmarshal(msg, &srv); err != nil {
				select {
				case out <- tts.Chunk{Error: fmt.Errorf("failed to decode ws frame: %w", err)}:
				case <-ctx.Done():
				}
				return
			}

			if srv.Error != "" || srv.Message != "" {
				detail := srv.Message
				if detail == "" {
					detail = srv.Error
				}
				select {
				case out <- tts.Chunk{Error: fmt.Errorf("audio generation failed: %s", detail)}:
				case <-ctx.Done():
				}
				return
			}

			if srv.Audio != "" {
				audioBytes, decErr := base64.StdEncoding.DecodeString(srv.Audio)
				if decErr != nil {
					select {
					case out <- tts.Chunk{Error: fmt.Errorf("failed to decode base64 audio: %w", decErr)}:
					case <-ctx.Done():
					}
					return
				}
				chunk := tts.Chunk{Data: audioBytes}
				if srv.Alignment != nil {
					chunk.Alignment = toAlignmentData(*srv.Alignment)
				}
				if srv.NormalizedAlignment != nil {
					chunk.NormalizedAlignment = toAlignmentData(
						*srv.NormalizedAlignment,
					)
				}
				select {
				case out <- chunk:
				case <-ctx.Done():
					return
				}
			}

			if srv.IsFinal != nil && *srv.IsFinal {
				select {
				case out <- tts.Chunk{Done: true}:
				case <-ctx.Done():
				}
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
	}()

	select {
	case <-readDone:
	case <-ctx.Done():
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

// ListVoices retrieves the list of available voices from ElevenLabs.
func (c *Client) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	url := fmt.Sprintf("%s/voices", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var voiceResp voiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&voiceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	voices := make([]tts.Voice, len(voiceResp.Voices))
	for i, v := range voiceResp.Voices {
		voices[i] = tts.Voice{
			VoiceID:     v.VoiceID,
			Name:        v.Name,
			Category:    v.Category,
			Description: v.Description,
			PreviewURL:  v.PreviewURL,
			Labels:      v.Labels,
		}
	}
	return voices, nil
}

// GenerateForcedAlignment aligns an existing audio file with its transcript.
// This makes [Client] satisfy [tts.ForcedAlignmentProvider].
func (c *Client) GenerateForcedAlignment(
	ctx context.Context,
	audioFile []byte,
	transcript string,
) (*tts.ForcedAlignmentData, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileWriter, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := fileWriter.Write(audioFile); err != nil {
		return nil, fmt.Errorf("failed to write audio file: %w", err)
	}

	if err := writer.WriteField("text", transcript); err != nil {
		return nil, fmt.Errorf("failed to write text field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/forced-alignment", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var alignmentResp forcedAlignmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&alignmentResp); err != nil {
		return nil, fmt.Errorf(
			"failed to decode forced alignment response: %w",
			err,
		)
	}

	characters := make([]tts.CharAlignment, len(alignmentResp.Characters))
	for i, char := range alignmentResp.Characters {
		characters[i] = tts.CharAlignment(char)
	}

	words := make([]tts.WordAlignment, len(alignmentResp.Words))
	for i, word := range alignmentResp.Words {
		words[i] = tts.WordAlignment(word)
	}

	return &tts.ForcedAlignmentData{
		Characters: characters,
		Words:      words,
		Loss:       alignmentResp.Loss,
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
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf(
			"audio generation failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	if errResp.Detail.Message != "" {
		return fmt.Errorf("audio generation failed: %s", errResp.Detail.Message)
	}

	return fmt.Errorf("audio generation failed with status %d", resp.StatusCode)
}

func toAlignmentData(a alignmentResponse) *tts.AlignmentData {
	if len(a.Characters) == 0 {
		return nil
	}
	return &tts.AlignmentData{
		Characters:                 a.Characters,
		CharacterStartTimesSeconds: a.CharacterStartTimesSeconds,
		CharacterEndTimesSeconds:   a.CharacterEndTimesSeconds,
	}
}

func contentTypeForFormat(format string) string {
	if len(format) >= 3 {
		switch format[:3] {
		case "mp3":
			return "audio/mpeg"
		case "pcm":
			return "audio/pcm"
		case "wav":
			return "audio/wav"
		}
	}
	return "audio/mpeg"
}
