package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type assemblyAIOptions struct {
	pollInterval    time.Duration
	maxPollDuration time.Duration
	speakerLabels   bool
}

// AssemblyAIOption configures AssemblyAI-specific transcription behavior.
type AssemblyAIOption func(*assemblyAIOptions)

type assemblyAIClient struct {
	providerOptions transcriptionClientOptions
	options         assemblyAIOptions
	httpClient      *http.Client
	baseURL         string
}

// AssemblyAIClient is the AssemblyAI implementation of SpeechToTextClient.
type AssemblyAIClient SpeechToTextClient

type aaiUploadResponse struct {
	UploadURL string `json:"upload_url"`
}

type aaiTranscriptRequest struct {
	AudioURL      string `json:"audio_url"`
	LanguageCode  string `json:"language_code,omitempty"`
	SpeakerLabels bool   `json:"speaker_labels,omitempty"`
	SpeechModel   string `json:"speech_model,omitempty"`
}

type aaiTranscriptResponse struct {
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

func newAssemblyAIClient(
	opts transcriptionClientOptions,
) AssemblyAIClient {
	aaiOpts := assemblyAIOptions{
		pollInterval:    3 * time.Second,
		maxPollDuration: 5 * time.Minute,
	}
	for _, o := range opts.assemblyAIOptions {
		o(&aaiOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &assemblyAIClient{
		providerOptions: opts,
		options:         aaiOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.assemblyai.com/v2",
	}
}

func (a *assemblyAIClient) transcribe(
	ctx context.Context,
	audioFile []byte,
	options ...Option,
) (*Response, error) {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}

	uploadURL, err := a.upload(ctx, audioFile)
	if err != nil {
		return nil, err
	}

	transcriptReq := aaiTranscriptRequest{
		AudioURL:      uploadURL,
		SpeakerLabels: a.options.speakerLabels,
	}

	apiModel := a.providerOptions.model.APIModel
	if apiModel != "" && apiModel != "best" {
		transcriptReq.SpeechModel = apiModel
	}

	if opts.Language != "" {
		transcriptReq.LanguageCode = opts.Language
	}

	transcriptID, err := a.createTranscript(
		ctx,
		transcriptReq,
	)
	if err != nil {
		return nil, err
	}

	result, err := a.pollTranscript(ctx, transcriptID)
	if err != nil {
		return nil, err
	}

	return a.mapResponse(result), nil
}

func (a *assemblyAIClient) translate(
	_ context.Context,
	_ []byte,
	_ ...Option,
) (*Response, error) {
	return nil, fmt.Errorf(
		"assemblyai does not support translation",
	)
}

func (a *assemblyAIClient) upload(
	ctx context.Context,
	audioFile []byte,
) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		a.baseURL+"/upload",
		bytes.NewReader(audioFile),
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create upload request: %w",
			err,
		)
	}

	req.Header.Set(
		"Authorization",
		a.providerOptions.apiKey,
	)
	req.Header.Set(
		"Content-Type",
		"application/octet-stream",
	)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"failed to upload audio: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(
			"failed to read upload response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"upload API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var uploadResp aaiUploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf(
			"failed to unmarshal upload response: %w",
			err,
		)
	}

	return uploadResp.UploadURL, nil
}

func (a *assemblyAIClient) createTranscript(
	ctx context.Context,
	transcriptReq aaiTranscriptRequest,
) (string, error) {
	jsonBody, err := json.Marshal(transcriptReq)
	if err != nil {
		return "", fmt.Errorf(
			"failed to marshal transcript request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		a.baseURL+"/transcript",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create transcript request: %w",
			err,
		)
	}

	req.Header.Set(
		"Authorization",
		a.providerOptions.apiKey,
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create transcript: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(
			"failed to read transcript response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"transcript API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var transcriptResp aaiTranscriptResponse
	if err := json.Unmarshal(body, &transcriptResp); err != nil {
		return "", fmt.Errorf(
			"failed to unmarshal transcript response: %w",
			err,
		)
	}

	return transcriptResp.ID, nil
}

func (a *assemblyAIClient) pollTranscript(
	ctx context.Context,
	transcriptID string,
) (*aaiTranscriptResponse, error) {
	deadline := time.Now().Add(a.options.maxPollDuration)
	pollURL := fmt.Sprintf(
		"%s/transcript/%s",
		a.baseURL,
		transcriptID,
	)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(
			ctx,
			"GET",
			pollURL,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create poll request: %w",
				err,
			)
		}

		req.Header.Set(
			"Authorization",
			a.providerOptions.apiKey,
		)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to poll transcript: %w",
				err,
			)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to read poll response: %w",
				err,
			)
		}

		var result aaiTranscriptResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal poll response: %w",
				err,
			)
		}

		switch result.Status {
		case "completed":
			return &result, nil
		case "error":
			return nil, fmt.Errorf(
				"transcription failed: %s",
				result.Error,
			)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(a.options.pollInterval):
		}
	}

	return nil, fmt.Errorf(
		"transcription timed out after %s",
		a.options.maxPollDuration,
	)
}

func (a *assemblyAIClient) mapResponse(
	result *aaiTranscriptResponse,
) *Response {
	resp := &Response{
		Text:     result.Text,
		Duration: result.AudioDuration,
		Model:    a.providerOptions.model.APIModel,
		Usage: Usage{
			DurationSec: result.AudioDuration,
		},
	}

	words := make([]Word, len(result.Words))
	for i, w := range result.Words {
		words[i] = Word{
			Word:  w.Text,
			Start: float64(w.Start) / 1000.0,
			End:   float64(w.End) / 1000.0,
		}
	}
	resp.Words = words

	return resp
}

// WithAssemblyAIPollInterval sets the interval between polling attempts.
func WithAssemblyAIPollInterval(
	d time.Duration,
) AssemblyAIOption {
	return func(options *assemblyAIOptions) {
		options.pollInterval = d
	}
}

// WithAssemblyAIMaxPollDuration sets the maximum duration to wait for transcription.
func WithAssemblyAIMaxPollDuration(
	d time.Duration,
) AssemblyAIOption {
	return func(options *assemblyAIOptions) {
		options.maxPollDuration = d
	}
}

// WithAssemblyAISpeakerLabels enables speaker diarization.
func WithAssemblyAISpeakerLabels(
	enabled bool,
) AssemblyAIOption {
	return func(options *assemblyAIOptions) {
		options.speakerLabels = enabled
	}
}
