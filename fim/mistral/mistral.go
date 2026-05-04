// Package mistral provides a Mistral Codestral implementation of the [fim.FIM] interface.
package mistral

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

	"github.com/joakimcarlsson/ai/fim"
	"github.com/joakimcarlsson/ai/model"
)

const defaultBaseURL = "https://api.mistral.ai/v1/fim/completions"

// Options configures the Mistral FIM client.
type Options struct {
	apiKey      string
	model       model.Model
	maxTokens   int64
	temperature *float64
	topP        *float64
	timeout     *time.Duration
	minTokens   *int64
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Mistral.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.apiKey = apiKey
	}
}

// WithModel selects the FIM model.
func WithModel(m model.Model) Option {
	return func(o *Options) {
		o.model = m
	}
}

// WithMaxTokens sets the default maximum number of tokens to generate.
func WithMaxTokens(maxTokens int64) Option {
	return func(o *Options) {
		o.maxTokens = maxTokens
	}
}

// WithTemperature sets the default sampling temperature (0.0 to 1.0).
func WithTemperature(temperature float64) Option {
	return func(o *Options) {
		o.temperature = &temperature
	}
}

// WithTopP sets the default nucleus sampling probability.
func WithTopP(topP float64) Option {
	return func(o *Options) {
		o.topP = &topP
	}
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.timeout = &timeout
	}
}

// WithMinTokens sets the minimum number of tokens to generate.
func WithMinTokens(minTokens int64) Option {
	return func(o *Options) {
		o.minTokens = &minTokens
	}
}

// Client implements [fim.FIM] against the Mistral Codestral FIM API.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewFIM constructs a Mistral FIM client. The returned [fim.FIM] is wrapped with
// [fim.WithTracing], so callers always get tracing spans and metrics.
func NewFIM(opts ...Option) fim.FIM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	timeout := 60 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}

	return fim.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
	}, fim.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

// Model returns the configured FIM model.
func (c *Client) Model() model.Model {
	return c.options.model
}

type request struct {
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

type choice struct {
	Index   int `json:"index"`
	Message struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type streamDelta struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type streamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
	Usage   *usage         `json:"usage,omitempty"`
}

func (c *Client) buildRequest(req fim.Request, stream bool) request {
	out := request{
		Model:  c.options.model.APIModel,
		Prompt: req.Prompt,
		Suffix: req.Suffix,
		Stream: stream,
	}

	if req.MaxTokens != nil {
		out.MaxTokens = req.MaxTokens
	} else if c.options.maxTokens > 0 {
		out.MaxTokens = &c.options.maxTokens
	}

	if req.Temperature != nil {
		out.Temperature = req.Temperature
	} else if c.options.temperature != nil {
		out.Temperature = c.options.temperature
	}

	if req.TopP != nil {
		out.TopP = req.TopP
	} else if c.options.topP != nil {
		out.TopP = c.options.topP
	}

	if req.RandomSeed != nil {
		out.RandomSeed = req.RandomSeed
	}

	if len(req.Stop) > 0 {
		out.Stop = req.Stop
	}

	if c.options.minTokens != nil {
		out.MinTokens = c.options.minTokens
	}

	return out
}

func mapFinishReason(reason string) fim.FinishReason {
	switch reason {
	case "stop":
		return fim.FinishReasonStop
	case "length":
		return fim.FinishReasonLength
	default:
		return fim.FinishReasonUnknown
	}
}

// Complete performs a non-streaming FIM completion.
func (c *Client) Complete(
	ctx context.Context,
	req fim.Request,
) (*fim.Response, error) {
	body, err := json.Marshal(c.buildRequest(req, false))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, defaultBaseURL, bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"mistral fim api error (status %d): %s",
			resp.StatusCode, string(bodyBytes),
		)
	}

	var fimResp response
	if err := json.NewDecoder(resp.Body).Decode(&fimResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(fimResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from mistral fim")
	}

	return &fim.Response{
		Content: fimResp.Choices[0].Message.Content,
		Usage: fim.Usage{
			InputTokens:  fimResp.Usage.PromptTokens,
			OutputTokens: fimResp.Usage.CompletionTokens,
		},
		FinishReason: mapFinishReason(fimResp.Choices[0].FinishReason),
	}, nil
}

// CompleteStream performs a streaming FIM completion via Server-Sent Events.
func (c *Client) CompleteStream(
	ctx context.Context,
	req fim.Request,
) <-chan fim.Event {
	eventChan := make(chan fim.Event)

	go func() {
		defer close(eventChan)

		body, err := json.Marshal(c.buildRequest(req, true))
		if err != nil {
			eventChan <- fim.Event{Type: fim.EventError, Error: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}

		httpReq, err := http.NewRequestWithContext(
			ctx, http.MethodPost, defaultBaseURL, bytes.NewReader(body),
		)
		if err != nil {
			eventChan <- fim.Event{Type: fim.EventError, Error: fmt.Errorf("failed to create request: %w", err)}
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.options.apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			eventChan <- fim.Event{Type: fim.EventError, Error: fmt.Errorf("failed to send request: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			eventChan <- fim.Event{Type: fim.EventError, Error: fmt.Errorf("mistral fim api error (status %d): %s", resp.StatusCode, string(bodyBytes))}
			return
		}

		reader := bufio.NewReader(resp.Body)
		var currentContent strings.Builder
		var finalUsage fim.Usage
		var finalFinishReason fim.FinishReason

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					eventChan <- fim.Event{
						Type: fim.EventComplete,
						Response: &fim.Response{
							Content:      currentContent.String(),
							Usage:        finalUsage,
							FinishReason: finalFinishReason,
						},
					}
					return
				}
				eventChan <- fim.Event{Type: fim.EventError, Error: fmt.Errorf("error reading stream: %w", err)}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			data := bytes.TrimPrefix(line, []byte("data: "))
			if bytes.Equal(data, []byte("[DONE]")) {
				eventChan <- fim.Event{
					Type: fim.EventComplete,
					Response: &fim.Response{
						Content:      currentContent.String(),
						Usage:        finalUsage,
						FinishReason: finalFinishReason,
					},
				}
				return
			}

			var sr streamResponse
			if err := json.Unmarshal(data, &sr); err != nil {
				continue
			}

			for _, ch := range sr.Choices {
				if ch.Delta.Content != "" {
					currentContent.WriteString(ch.Delta.Content)
					eventChan <- fim.Event{
						Type:    fim.EventContentDelta,
						Content: ch.Delta.Content,
					}
				}
				if ch.FinishReason != nil {
					finalFinishReason = mapFinishReason(*ch.FinishReason)
				}
			}

			if sr.Usage != nil {
				finalUsage = fim.Usage{
					InputTokens:  sr.Usage.PromptTokens,
					OutputTokens: sr.Usage.CompletionTokens,
				}
			}
		}
	}()

	return eventChan
}
