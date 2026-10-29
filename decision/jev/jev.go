// Package jev provides a TypeSafe AI Jev (System One) implementation of the
// [decision.Decider] interface.
package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joakimcarlsson/ai/decision"
)

const (
	defaultBaseURL = "https://api.typesafe.ai"
	apiKeyEnv      = "TYPESAFE_API_KEY"
)

// Options configures the Jev decider client.
type Options struct {
	apiKey     string
	model      decision.DecisionModel
	baseURL    string
	httpClient *http.Client
	timeout    *time.Duration
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with TypeSafe. It defaults
// to the TYPESAFE_API_KEY environment variable.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.apiKey = apiKey
	}
}

// WithModel selects the decision model. It defaults to
// Models[JevLatest].
func WithModel(m decision.DecisionModel) Option {
	return func(o *Options) {
		o.model = m
	}
}

// WithBaseURL overrides the API base URL (without the /v1/systemone path).
func WithBaseURL(baseURL string) Option {
	return func(o *Options) {
		o.baseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client used for requests. It takes precedence
// over WithTimeout.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *Options) {
		o.httpClient = httpClient
	}
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.timeout = &timeout
	}
}

// Client implements [decision.Decider] against the TypeSafe System One API.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewDecider constructs a Jev decider client. The returned [decision.Decider]
// is wrapped with [decision.WithTracing], so callers always get tracing spans
// and metrics.
func NewDecider(opts ...Option) decision.Decider {
	options := Options{
		apiKey:  os.Getenv(apiKeyEnv),
		model:   Models[JevLatest],
		baseURL: defaultBaseURL,
	}
	for _, o := range opts {
		o(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		timeout := 30 * time.Second
		if options.timeout != nil {
			timeout = *options.timeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return decision.WithTracing(&Client{
		options:    options,
		httpClient: httpClient,
	})
}

// Model returns the configured decision model.
func (c *Client) Model() decision.DecisionModel {
	return c.options.model
}

type request struct {
	Model     string                       `json:"model"`
	State     any                          `json:"state"`
	Questions map[string]decision.Question `json:"questions"`
}

type response struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   decision.Usage             `json:"usage"`
}

// Decide answers every question over the same state in a single call.
func (c *Client) Decide(
	ctx context.Context,
	state any,
	questions map[string]decision.Question,
) (*decision.Response, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}

	var jevResp response
	if err := decision.PostJSON(
		ctx,
		c.httpClient,
		c.options.baseURL+"/v1/systemone",
		c.options.apiKey,
		request{
			Model:     c.options.model.APIModel,
			State:     state,
			Questions: questions,
		},
		&jevResp,
	); err != nil {
		return nil, err
	}

	answers := make(map[string]decision.Answer, len(jevResp.Answers))
	for name, raw := range jevResp.Answers {
		answer, err := decision.UnmarshalAnswer(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to decode answer %q: %w", name, err)
		}
		answers[name] = answer
	}

	return &decision.Response{
		Answers: answers,
		Usage:   jevResp.Usage,
		Model:   jevResp.Model,
	}, nil
}
