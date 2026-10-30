// Package jev provides a TypeSafe AI Jev (System One) implementation of the
// [decision.Decider] interface.
package jev

import (
	"context"
	"encoding/base64"
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
	model      decision.Model
	baseURL    string
	httpClient *http.Client
	timeout    *time.Duration

	longContext  bool
	modelsPath   string
	modelsDecode func([]byte) ([]decision.ModelInfo, error)
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
func WithModel(m decision.Model) Option {
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

// WithLongContext opts every request into the provider's long-context mode,
// which accepts larger states at some cost in speed and accuracy. The provider
// switches to it on its own for very large requests and for images.
func WithLongContext() Option {
	return func(o *Options) {
		o.longContext = true
	}
}

// WithModelsEndpoint overrides how [Client.ListModels] works: path is
// appended to the base URL and decode turns the response body into model
// infos. It exists for gateways that serve System One but list models in their
// own format.
func WithModelsEndpoint(
	path string,
	decode func([]byte) ([]decision.ModelInfo, error),
) Option {
	return func(o *Options) {
		o.modelsPath = path
		o.modelsDecode = decode
	}
}

// Client implements [decision.Decider] against the TypeSafe System One API.
type Client struct {
	options    Options
	httpClient *http.Client
}

// NewDecider constructs a Jev decider client. The returned [decision.Decider]
// is wrapped with [decision.WithTracing], so callers always get tracing spans
// and metrics, and it is also a [decision.ModelLister].
func NewDecider(opts ...Option) decision.Decider {
	return decision.WithTracing(NewClient(opts...))
}

// NewClient constructs a Jev client without the tracing wrapper.
func NewClient(opts ...Option) *Client {
	options := Options{
		apiKey:  os.Getenv(apiKeyEnv),
		model:   Models[JevLatest],
		baseURL: defaultBaseURL,

		modelsPath:   "/v1/models",
		modelsDecode: decodeModels,
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

	return &Client{options: options, httpClient: httpClient}
}

// Model returns the configured decision model.
func (c *Client) Model() decision.Model {
	return c.options.model
}

type request struct {
	Model     string                       `json:"model"`
	State     any                          `json:"state"`
	Questions map[string]decision.Question `json:"questions"`
	Assets    map[string]asset             `json:"assets,omitempty"`
}

type response struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   decision.Usage             `json:"usage"`
}

type asset struct {
	Type       string `json:"type"`
	Format     string `json:"format"`
	DataBase64 string `json:"data_base64"`
}

// Decide answers every question over the same state in a single call. Images
// passed with [decision.WithImages] are sent as base64 assets; Box questions
// and images need a provider and model that support them.
func (c *Client) Decide(
	ctx context.Context,
	state any,
	questions map[string]decision.Question,
	opts ...decision.CallOption,
) (*decision.Response, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}
	call := decision.ApplyOptions(opts)

	req := request{
		Model:     c.options.model.APIModel,
		State:     state,
		Questions: questions,
	}
	headers := map[string]string{}
	if len(call.Images) > 0 {
		req.Assets = make(map[string]asset, len(call.Images))
		for _, img := range call.Images {
			switch img.Format {
			case "png", "jpg", "jpeg":
			default:
				return nil, fmt.Errorf(
					"unsupported image format %q: want png, jpg or jpeg",
					img.Format,
				)
			}
			req.Assets[img.Ref] = asset{
				Type:       "image",
				Format:     img.Format,
				DataBase64: base64.StdEncoding.EncodeToString(img.Data),
			}
		}
	}
	if len(req.Assets) > 0 || hasBox(questions) {
		headers["TypeSafe-Danger-Zone-Images"] = "true"
	}
	if c.options.longContext {
		headers["TypeSafe-Danger-Zone-Long-Context"] = "true"
	}

	var jevResp response
	if err := decision.PostJSON(
		ctx,
		c.httpClient,
		c.options.baseURL+"/v1/systemone",
		c.options.apiKey,
		headers,
		req,
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

func hasBox(questions map[string]decision.Question) bool {
	for _, q := range questions {
		if q.Type() == "box" {
			return true
		}
	}
	return false
}

// ListModels returns the models and aliases available to the account.
func (c *Client) ListModels(ctx context.Context) ([]decision.ModelInfo, error) {
	var raw json.RawMessage
	if err := decision.GetJSON(
		ctx,
		c.httpClient,
		c.options.baseURL+c.options.modelsPath,
		c.options.apiKey,
		&raw,
	); err != nil {
		return nil, err
	}
	return c.options.modelsDecode(raw)
}

func decodeModels(body []byte) ([]decision.ModelInfo, error) {
	var r struct {
		Models []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ReleaseDate string `json:"release_date"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("failed to decode models: %w", err)
	}
	out := make([]decision.ModelInfo, 0, len(r.Models))
	for _, m := range r.Models {
		info := decision.ModelInfo{Name: m.Name, Description: m.Description}
		for _, layout := range []string{time.RFC3339, time.DateOnly} {
			if t, err := time.Parse(layout, m.ReleaseDate); err == nil {
				info.ReleaseDate = t
				break
			}
		}
		out = append(out, info)
	}
	return out, nil
}
