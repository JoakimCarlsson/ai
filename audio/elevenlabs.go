package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultElevenLabsBaseURL = "https://api.elevenlabs.io/v1"
	defaultVoiceID           = "EXAVITQu4vr4xnSDxMaL" // Rachel voice
)

// ElevenLabsClient implements the AudioGenerationClient interface for ElevenLabs API.
type ElevenLabsClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

func newElevenLabsClient(options audioGenerationClientOptions) ElevenLabsClient {
	baseURL := defaultElevenLabsBaseURL
	for _, opt := range options.elevenLabsOptions {
		opts := &elevenLabsOptions{}
		opt(opts)
		if opts.baseURL != "" {
			baseURL = opts.baseURL
		}
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	modelID := "eleven_multilingual_v2"
	if options.model.APIModel != "" {
		modelID = options.model.APIModel
	}

	return ElevenLabsClient{
		apiKey:  options.apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		model: modelID,
	}
}

type elevenLabsTTSRequest struct {
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

type elevenLabsVoiceResponse struct {
	Voices []elevenLabsVoice `json:"voices"`
}

type elevenLabsVoice struct {
	VoiceID                 string            `json:"voice_id"`
	Name                    string            `json:"name"`
	Category                string            `json:"category"`
	Description             string            `json:"description"`
	PreviewURL              string            `json:"preview_url"`
	Labels                  map[string]string `json:"labels"`
	HighQualityBaseModelIDs []string          `json:"high_quality_base_model_ids,omitempty"`
}

type elevenLabsErrorResponse struct {
	Detail struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"detail"`
}

func (c ElevenLabsClient) generate(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*AudioResponse, error) {
	opts := &GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}

	voiceID := defaultVoiceID
	if opts.VoiceID != "" {
		voiceID = opts.VoiceID
	}

	outputFormat := "mp3_44100_128"
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	reqBody := elevenLabsTTSRequest{
		Text:         text,
		ModelID:      c.model,
		OutputFormat: outputFormat,
	}

	if opts.Stability != nil || opts.SimilarityBoost != nil || opts.Style != nil || opts.SpeakerBoost != nil {
		reqBody.VoiceSettings = &voiceSettings{}
		if opts.Stability != nil {
			reqBody.VoiceSettings.Stability = *opts.Stability
		}
		if opts.SimilarityBoost != nil {
			reqBody.VoiceSettings.SimilarityBoost = *opts.SimilarityBoost
		}
		if opts.Style != nil {
			reqBody.VoiceSettings.Style = *opts.Style
		}
		if opts.SpeakerBoost != nil {
			reqBody.VoiceSettings.SpeakerBoost = *opts.SpeakerBoost
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/text-to-speech/%s", c.baseURL, voiceID)
	if opts.OptimizeStreamingLatency != nil {
		url = fmt.Sprintf("%s?optimize_streaming_latency=%d", url, *opts.OptimizeStreamingLatency)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

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

	return &AudioResponse{
		AudioData:   audioData,
		ContentType: contentType,
		Usage: AudioUsage{
			Characters: charCount,
		},
		Model: c.model,
	}, nil
}

func (c ElevenLabsClient) stream(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan AudioChunk, error) {
	opts := &GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}

	voiceID := defaultVoiceID
	if opts.VoiceID != "" {
		voiceID = opts.VoiceID
	}

	outputFormat := "mp3_44100_128"
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}

	reqBody := elevenLabsTTSRequest{
		Text:         text,
		ModelID:      c.model,
		OutputFormat: outputFormat,
	}

	if opts.Stability != nil || opts.SimilarityBoost != nil || opts.Style != nil || opts.SpeakerBoost != nil {
		reqBody.VoiceSettings = &voiceSettings{}
		if opts.Stability != nil {
			reqBody.VoiceSettings.Stability = *opts.Stability
		}
		if opts.SimilarityBoost != nil {
			reqBody.VoiceSettings.SimilarityBoost = *opts.SimilarityBoost
		}
		if opts.Style != nil {
			reqBody.VoiceSettings.Style = *opts.Style
		}
		if opts.SpeakerBoost != nil {
			reqBody.VoiceSettings.SpeakerBoost = *opts.SpeakerBoost
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		ch := make(chan AudioChunk, 1)
		ch <- AudioChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		close(ch)
		return ch, nil
	}

	url := fmt.Sprintf("%s/text-to-speech/%s/stream", c.baseURL, voiceID)
	if opts.OptimizeStreamingLatency != nil {
		url = fmt.Sprintf("%s?optimize_streaming_latency=%d", url, *opts.OptimizeStreamingLatency)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		ch := make(chan AudioChunk, 1)
		ch <- AudioChunk{Error: fmt.Errorf("failed to create request: %w", err)}
		close(ch)
		return ch, nil
	}

	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		ch := make(chan AudioChunk, 1)
		ch <- AudioChunk{Error: fmt.Errorf("request failed: %w", err)}
		close(ch)
		return ch, nil
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		ch := make(chan AudioChunk, 1)
		ch <- AudioChunk{Error: c.parseError(resp)}
		close(ch)
		return ch, nil
	}

	chunkChan := make(chan AudioChunk, 10)

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				chunkChan <- AudioChunk{Data: data, Done: false}
			}

			if err == io.EOF {
				chunkChan <- AudioChunk{Done: true}
				break
			}

			if err != nil {
				chunkChan <- AudioChunk{Error: fmt.Errorf("stream read error: %w", err)}
				break
			}

			select {
			case <-ctx.Done():
				chunkChan <- AudioChunk{Error: ctx.Err()}
				return
			default:
			}
		}
	}()

	return chunkChan, nil
}

func (c ElevenLabsClient) listVoices(ctx context.Context) ([]Voice, error) {
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

	var voiceResp elevenLabsVoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&voiceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	voices := make([]Voice, len(voiceResp.Voices))
	for i, v := range voiceResp.Voices {
		voices[i] = Voice{
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

func (c ElevenLabsClient) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("audio generation failed with status %d", resp.StatusCode)
	}

	var errResp elevenLabsErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("audio generation failed with status %d: %s", resp.StatusCode, string(body))
	}

	if errResp.Detail.Message != "" {
		return fmt.Errorf("audio generation failed: %s", errResp.Detail.Message)
	}

	return fmt.Errorf("audio generation failed with status %d", resp.StatusCode)
}
