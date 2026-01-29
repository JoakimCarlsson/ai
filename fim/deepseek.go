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

const deepseekFIMBaseURL = "https://api.deepseek.com/beta/completions"

type deepseekOptions struct {
	frequencyPenalty *float64
	presencePenalty  *float64
	echo             *bool
}

// DeepSeekOption configures the DeepSeek FIM client.
type DeepSeekOption func(*deepseekOptions)

type deepseekClient struct {
	providerOptions fimClientOptions
	options         deepseekOptions
	httpClient      *http.Client
}

func newDeepSeekClient(opts fimClientOptions) *deepseekClient {
	deepseekOpts := deepseekOptions{}
	for _, o := range opts.deepseekOptions {
		o(&deepseekOpts)
	}

	timeout := 60 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &deepseekClient{
		providerOptions: opts,
		options:         deepseekOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type deepseekFIMRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	Suffix           string   `json:"suffix,omitempty"`
	MaxTokens        *int64   `json:"max_tokens,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	Stream           bool     `json:"stream"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	Echo             *bool    `json:"echo,omitempty"`
}

type deepseekFIMChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type deepseekFIMUsage struct {
	PromptTokens          int64 `json:"prompt_tokens"`
	CompletionTokens      int64 `json:"completion_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
	PromptCacheHitTokens  int64 `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int64 `json:"prompt_cache_miss_tokens"`
}

type deepseekFIMResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []deepseekFIMChoice `json:"choices"`
	Usage   deepseekFIMUsage    `json:"usage"`
}

type deepseekFIMStreamChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type deepseekFIMStreamResponse struct {
	ID      string                    `json:"id"`
	Object  string                    `json:"object"`
	Created int64                     `json:"created"`
	Model   string                    `json:"model"`
	Choices []deepseekFIMStreamChoice `json:"choices"`
	Usage   *deepseekFIMUsage         `json:"usage,omitempty"`
}

func (d *deepseekClient) buildRequest(req FIMRequest, stream bool) deepseekFIMRequest {
	fimReq := deepseekFIMRequest{
		Model:  d.providerOptions.model.APIModel,
		Prompt: req.Prompt,
		Suffix: req.Suffix,
		Stream: stream,
	}

	if req.MaxTokens != nil {
		fimReq.MaxTokens = req.MaxTokens
	} else if d.providerOptions.maxTokens > 0 {
		fimReq.MaxTokens = &d.providerOptions.maxTokens
	}

	if req.Temperature != nil {
		fimReq.Temperature = req.Temperature
	} else if d.providerOptions.temperature != nil {
		fimReq.Temperature = d.providerOptions.temperature
	}

	if req.TopP != nil {
		fimReq.TopP = req.TopP
	} else if d.providerOptions.topP != nil {
		fimReq.TopP = d.providerOptions.topP
	}

	if len(req.Stop) > 0 {
		fimReq.Stop = req.Stop
	}

	if d.options.frequencyPenalty != nil {
		fimReq.FrequencyPenalty = d.options.frequencyPenalty
	}

	if d.options.presencePenalty != nil {
		fimReq.PresencePenalty = d.options.presencePenalty
	}

	if d.options.echo != nil {
		fimReq.Echo = d.options.echo
	}

	return fimReq
}

func (d *deepseekClient) finishReason(reason string) FinishReason {
	switch reason {
	case "stop":
		return FinishReasonStop
	case "length":
		return FinishReasonLength
	default:
		return FinishReasonUnknown
	}
}

func (d *deepseekClient) complete(
	ctx context.Context,
	req FIMRequest,
) (*FIMResponse, error) {
	fimReq := d.buildRequest(req, false)

	body, err := json.Marshal(fimReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, deepseekFIMBaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.providerOptions.apiKey)

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepseek fim api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var fimResp deepseekFIMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fimResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(fimResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from deepseek fim")
	}

	return &FIMResponse{
		Content: fimResp.Choices[0].Text,
		Usage: FIMUsage{
			InputTokens:  fimResp.Usage.PromptTokens,
			OutputTokens: fimResp.Usage.CompletionTokens,
		},
		FinishReason: d.finishReason(fimResp.Choices[0].FinishReason),
	}, nil
}

func (d *deepseekClient) stream(
	ctx context.Context,
	req FIMRequest,
) <-chan FIMEvent {
	fimReq := d.buildRequest(req, true)
	eventChan := make(chan FIMEvent)

	go func() {
		defer close(eventChan)

		body, err := json.Marshal(fimReq)
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, deepseekFIMBaseURL, bytes.NewReader(body))
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to create request: %w", err)}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+d.providerOptions.apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := d.httpClient.Do(httpReq)
		if err != nil {
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("failed to send request: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			eventChan <- FIMEvent{Type: EventError, Error: fmt.Errorf("deepseek fim api error (status %d): %s", resp.StatusCode, string(bodyBytes))}
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

				var streamResp deepseekFIMStreamResponse
				if err := json.Unmarshal(data, &streamResp); err != nil {
					continue
				}

				for _, choice := range streamResp.Choices {
					if choice.Text != "" {
						currentContent.WriteString(choice.Text)
						eventChan <- FIMEvent{
							Type:    EventContentDelta,
							Content: choice.Text,
						}
					}
					if choice.FinishReason != "" {
						finalFinishReason = d.finishReason(choice.FinishReason)
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

// WithFrequencyPenalty sets the frequency penalty to reduce repetition (-2.0 to 2.0).
func WithFrequencyPenalty(frequencyPenalty float64) DeepSeekOption {
	return func(options *deepseekOptions) {
		options.frequencyPenalty = &frequencyPenalty
	}
}

// WithPresencePenalty sets the presence penalty to encourage topic diversity (-2.0 to 2.0).
func WithPresencePenalty(presencePenalty float64) DeepSeekOption {
	return func(options *deepseekOptions) {
		options.presencePenalty = &presencePenalty
	}
}

// WithEcho enables echoing the prompt back in addition to the completion.
func WithEcho(echo bool) DeepSeekOption {
	return func(options *deepseekOptions) {
		options.echo = &echo
	}
}
