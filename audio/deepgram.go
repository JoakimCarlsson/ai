package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultDeepgramBaseURL = "https://api.deepgram.com/v1"
	defaultDeepgramModel   = "aura-asteria-en"
)

// DeepgramClient implements the GenerationClient interface for the Deepgram
// Aura text-to-speech API.
type DeepgramClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
	encoding   string
	container  string
	sampleRate int
	bitRate    int
}

func newDeepgramClient(
	options audioGenerationClientOptions,
) DeepgramClient {
	baseURL := defaultDeepgramBaseURL
	clientModel := defaultDeepgramModel
	if options.model.APIModel != "" {
		clientModel = options.model.APIModel
	}

	var encoding, container string
	var sampleRate, bitRate int

	for _, opt := range options.deepgramOptions {
		opts := &deepgramOptions{}
		opt(opts)
		if opts.baseURL != "" {
			baseURL = opts.baseURL
		}
		if opts.model != "" {
			clientModel = opts.model
		}
		if opts.encoding != "" {
			encoding = opts.encoding
		}
		if opts.container != "" {
			container = opts.container
		}
		if opts.sampleRate != 0 {
			sampleRate = opts.sampleRate
		}
		if opts.bitRate != 0 {
			bitRate = opts.bitRate
		}
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return DeepgramClient{
		apiKey:     options.apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		model:      clientModel,
		encoding:   encoding,
		container:  container,
		sampleRate: sampleRate,
		bitRate:    bitRate,
	}
}

type deepgramTTSRequest struct {
	Text string `json:"text"`
}

type deepgramErrorResponse struct {
	ErrCode string `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

func (c DeepgramClient) buildURL() string {
	q := url.Values{}
	q.Set("model", c.model)
	if c.encoding != "" {
		q.Set("encoding", c.encoding)
	}
	if c.container != "" {
		q.Set("container", c.container)
	}
	if c.sampleRate != 0 {
		q.Set("sample_rate", strconv.Itoa(c.sampleRate))
	}
	if c.bitRate != 0 {
		q.Set("bit_rate", strconv.Itoa(c.bitRate))
	}
	return fmt.Sprintf("%s/speak?%s", c.baseURL, q.Encode())
}

func (c DeepgramClient) newRequest(
	ctx context.Context,
	text string,
) (*http.Request, error) {
	body, err := json.Marshal(deepgramTTSRequest{Text: text})
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

	req.Header.Set("Authorization", "Token "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c DeepgramClient) generate(
	ctx context.Context,
	text string,
	_ ...GenerationOption,
) (*Response, error) {
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

	return &Response{
		AudioData:   audioData,
		ContentType: contentType,
		Usage:       Usage{Characters: charCount},
		Model:       c.model,
	}, nil
}

func (c DeepgramClient) stream(
	ctx context.Context,
	text string,
	_ ...GenerationOption,
) (<-chan Chunk, error) {
	req, err := c.newRequest(ctx, text)
	if err != nil {
		return nil, err
	}

	chunkChan := make(chan Chunk, 10)

	go func() {
		defer close(chunkChan)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			chunkChan <- Chunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			chunkChan <- Chunk{Error: c.parseError(resp)}
			return
		}

		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				chunkChan <- Chunk{Data: data, Done: false}
			}

			if err == io.EOF {
				chunkChan <- Chunk{Done: true}
				return
			}

			if err != nil {
				chunkChan <- Chunk{
					Error: fmt.Errorf("stream read error: %w", err),
				}
				return
			}

			select {
			case <-ctx.Done():
				chunkChan <- Chunk{Error: ctx.Err()}
				return
			default:
			}
		}
	}()

	return chunkChan, nil
}

// listVoices returns the set of Aura voices known to deps/ai. Deepgram does
// not expose a public list-voices endpoint; the canonical reference is
// https://developers.deepgram.com/docs/tts-models.
func (c DeepgramClient) listVoices(_ context.Context) ([]Voice, error) {
	return []Voice{
		{VoiceID: "aura-2-thalia-en", Name: "Thalia", Category: "aura-2"},
		{VoiceID: "aura-2-andromeda-en", Name: "Andromeda", Category: "aura-2"},
		{VoiceID: "aura-2-helena-en", Name: "Helena", Category: "aura-2"},
		{VoiceID: "aura-asteria-en", Name: "Asteria", Category: "aura"},
		{VoiceID: "aura-luna-en", Name: "Luna", Category: "aura"},
		{VoiceID: "aura-stella-en", Name: "Stella", Category: "aura"},
		{VoiceID: "aura-zeus-en", Name: "Zeus", Category: "aura"},
	}, nil
}

func (c DeepgramClient) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(
			"audio generation failed with status %d",
			resp.StatusCode,
		)
	}

	var errResp deepgramErrorResponse
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
