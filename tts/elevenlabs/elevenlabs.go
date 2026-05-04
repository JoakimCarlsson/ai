// Package elevenlabs provides an ElevenLabs implementation of the [tts.Generation]
// interface. The concrete [Client] also implements [tts.ForcedAlignmentProvider],
// reachable via type assertion on the value returned from [NewGeneration].
package elevenlabs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

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
// support so type assertion against the returned value succeeds.
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
	if charCountStr := resp.Header.Get("x-character-count"); charCountStr != "" {
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

// StreamAudio creates audio from text and returns a channel of audio chunks.
// If alignment is enabled, calls the /stream/with-timestamps endpoint.
func (c *Client) StreamAudio(
	ctx context.Context,
	text string,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}

	if opts.EnableAlignment {
		return c.streamWithTimestamps(ctx, text, opts)
	}
	return c.streamStandard(ctx, text, opts)
}

func (c *Client) streamStandard(
	ctx context.Context,
	text string,
	opts *tts.GenerationOptions,
) (<-chan tts.Chunk, error) {
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
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		close(ch)
		return ch, nil
	}

	url := fmt.Sprintf("%s/text-to-speech/%s/stream?output_format=%s",
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
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to create request: %w", err)}
		close(ch)
		return ch, nil
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	chunkChan := make(chan tts.Chunk, 10)

	go func() {
		defer close(chunkChan)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			chunkChan <- tts.Chunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			chunkChan <- tts.Chunk{Error: c.parseError(resp)}
			return
		}

		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				chunkChan <- tts.Chunk{Data: data, Done: false}
			}

			if err == io.EOF {
				chunkChan <- tts.Chunk{Done: true}
				break
			}

			if err != nil {
				chunkChan <- tts.Chunk{Error: fmt.Errorf("stream read error: %w", err)}
				break
			}

			select {
			case <-ctx.Done():
				chunkChan <- tts.Chunk{Error: ctx.Err()}
				return
			default:
			}
		}
	}()

	return chunkChan, nil
}

type streamChunkWithTimestamps struct {
	AudioBase64         string            `json:"audio_base64"`
	Alignment           alignmentResponse `json:"alignment"`
	NormalizedAlignment alignmentResponse `json:"normalized_alignment"`
}

func (c *Client) streamWithTimestamps(
	ctx context.Context,
	text string,
	opts *tts.GenerationOptions,
) (<-chan tts.Chunk, error) {
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
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		close(ch)
		return ch, nil
	}

	url := fmt.Sprintf(
		"%s/text-to-speech/%s/stream/with-timestamps?output_format=%s",
		c.baseURL,
		c.voiceID,
		outputFormat,
	)
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
		ch := make(chan tts.Chunk, 1)
		ch <- tts.Chunk{Error: fmt.Errorf("failed to create request: %w", err)}
		close(ch)
		return ch, nil
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	chunkChan := make(chan tts.Chunk, 10)

	go func() {
		defer close(chunkChan)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			chunkChan <- tts.Chunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			chunkChan <- tts.Chunk{Error: c.parseError(resp)}
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk streamChunkWithTimestamps
			err := decoder.Decode(&chunk)

			if err == io.EOF {
				chunkChan <- tts.Chunk{Done: true}
				break
			}

			if err != nil {
				chunkChan <- tts.Chunk{Error: fmt.Errorf("stream decode error: %w", err)}
				break
			}

			audioData, err := base64.StdEncoding.DecodeString(chunk.AudioBase64)
			if err != nil {
				chunkChan <- tts.Chunk{Error: fmt.Errorf("failed to decode base64 audio: %w", err)}
				break
			}

			chunkChan <- tts.Chunk{
				Data:                audioData,
				Done:                false,
				Alignment:           toAlignmentData(chunk.Alignment),
				NormalizedAlignment: toAlignmentData(chunk.NormalizedAlignment),
			}

			select {
			case <-ctx.Done():
				chunkChan <- tts.Chunk{Error: ctx.Err()}
				return
			default:
			}
		}
	}()

	return chunkChan, nil
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
