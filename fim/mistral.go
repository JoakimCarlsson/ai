package fim

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const mistralFIMBaseURL = "https://api.mistral.ai/v1/fim/completions"

type mistralOptions struct {
	minTokens *int64
}

// MistralOption configures the Mistral FIM client.
type MistralOption func(*mistralOptions)

type mistralClient struct {
	providerOptions fimClientOptions
	options         mistralOptions
	httpClient      *http.Client
}

func newMistralClient(opts fimClientOptions) *mistralClient {
	mistralOpts := mistralOptions{}
	for _, o := range opts.mistralOptions {
		o(&mistralOpts)
	}

	timeout := 60 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &mistralClient{
		providerOptions: opts,
		options:         mistralOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type mistralFIMRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Suffix      string   `json:"suffix,omitempty"`
	MaxTokens   *int64   `json:"max_tokens,omitempty"`
	MinTokens   *int64   `json:"min_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	RandomSeed  *int64   `json:"random_seed,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	Stream      bool     `json:"stream"`
}

type mistralFIMChoice struct {
	Index   int `json:"index"`
	Message struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type mistralFIMUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type mistralFIMResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []mistralFIMChoice `json:"choices"`
	Usage   mistralFIMUsage    `json:"usage"`
}

type mistralFIMStreamDelta struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type mistralFIMStreamChoice struct {
	Index        int                   `json:"index"`
	Delta        mistralFIMStreamDelta `json:"delta"`
	FinishReason *string               `json:"finish_reason"`
}

type mistralFIMStreamResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []mistralFIMStreamChoice `json:"choices"`
	Usage   *mistralFIMUsage         `json:"usage,omitempty"`
}

func (m *mistralClient) buildRequest(req FIMRequest, stream bool) mistralFIMRequest {
	fimReq := mistralFIMRequest{
		Model:  m.providerOptions.model.APIModel,
		Prompt: req.Prompt,
		Suffix: req.Suffix,
		Stream: stream,
	}

	if req.MaxTokens != nil {
		fimReq.MaxTokens = req.MaxTokens
	} else if m.providerOptions.maxTokens > 0 {
		fimReq.MaxTokens = &m.providerOptions.maxTokens
	}

	if req.Temperature != nil {
		fimReq.Temperature = req.Temperature
	} else if m.providerOptions.temperature != nil {
		fimReq.Temperature = m.providerOptions.temperature
	}

	if req.TopP != nil {
		fimReq.TopP = req.TopP
	} else if m.providerOptions.topP != nil {
		fimReq.TopP = m.providerOptions.topP
	}

	if req.RandomSeed != nil {
		fimReq.RandomSeed = req.RandomSeed
	}

	if len(req.Stop) > 0 {
		fimReq.Stop = req.Stop
	}

	if m.options.minTokens != nil {
		fimReq.MinTokens = m.options.minTokens
	}

	return fimReq
}

func (m *mistralClient) finishReason(reason string) FinishReason {
	switch reason {
	case "stop":
		return FinishReasonStop
	case "length":
		return FinishReasonLength
	default:
		return FinishReasonUnknown
	}
}

func (m *mistralClient) complete(
	ctx context.Context,
	req FIMRequest,
) (*FIMResponse, error) {
	fimReq := m.buildRequest(req, false)

	body, err := json.Marshal(fimReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralFIMBaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.providerOptions.apiKey)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mistral fim api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var fimResp mistralFIMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fimResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(fimResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from mistral fim")
	}

	return &FIMResponse{
		Content: fimResp.Choices[0].Message.Content,
		Usage: FIMUsage{
			InputTokens:  fimResp.Usage.PromptTokens,
			OutputTokens: fimResp.Usage.CompletionTokens,
		},
		FinishReason: m.finishReason(fimResp.Choices[0].FinishReason),
	}, nil
}

func (m *mistralClient) stream(
	ctx context.Context,
	req FIMRequest,
) <-chan FIMEvent {
	fimReq := m.buildRequest(req, true)
	eventChan := make(chan FIMEvent)

	go func() {
		defer close(eventChan)

		body, err := json.Marshal(fimReq)
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralFIMBaseURL, bytes.NewReader(body))
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to create request: %w", err)}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+m.providerOptions.apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := m.httpClient.Do(httpReq)
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to send request: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("mistral fim api error (status %d): %s", resp.StatusCode, string(bodyBytes))}
			return
		}

		reader := bufio.NewReader(resp.Body)
		var currentContent strings.Builder
		var finalUsage FIMUsage
		var finalFinishReason FinishReason

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					eventChan <- FIMEvent{
						Type: EventComplete,
						Response: &FIMResponse{
							Content:      currentContent.String(),
							Usage:        finalUsage,
							FinishReason: finalFinishReason,
						},
					}
					return
				}
				eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("error reading stream: %w", err)}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if bytes.HasPrefix(line, []byte("data: ")) {
				data := bytes.TrimPrefix(line, []byte("data: "))
				if bytes.Equal(data, []byte("[DONE]")) {
					eventChan <- FIMEvent{
						Type: EventComplete,
						Response: &FIMResponse{
							Content:      currentContent.String(),
							Usage:        finalUsage,
							FinishReason: finalFinishReason,
						},
					}
					return
				}

				var streamResp mistralFIMStreamResponse
				if err := json.Unmarshal(data, &streamResp); err != nil {
					continue
				}

				for _, choice := range streamResp.Choices {
					if choice.Delta.Content != "" {
						currentContent.WriteString(choice.Delta.Content)
						eventChan <- FIMEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
					}
					if choice.FinishReason != nil {
						finalFinishReason = m.finishReason(*choice.FinishReason)
					}
				}

				if streamResp.Usage != nil {
					finalUsage = FIMUsage{
						InputTokens:  streamResp.Usage.PromptTokens,
						OutputTokens: streamResp.Usage.CompletionTokens,
					}
				}
			}
		}
	}()

	return eventChan
}

// WithMinTokens sets the minimum number of tokens to generate.
func WithMinTokens(minTokens int64) MistralOption {
	return func(options *mistralOptions) {
		options.minTokens = &minTokens
	}
}
