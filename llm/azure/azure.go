// Package azure provides an Azure OpenAI implementation of the [llm.LLM] interface.
//
// Azure's request/response semantics are identical to OpenAI's, so this package
// embeds [llm/openai].Client and overrides only the SDK construction (custom auth
// + endpoint).
package azure

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

// Options configures the Azure OpenAI LLM client.
type Options struct {
	apiKey        string
	model         model.Model
	maxTokens     int64
	temperature   *float64
	topP          *float64
	topK          *int64
	stopSequences []string
	timeout       *time.Duration
	endpoint      string
	apiVersion    string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key (optional — Azure also supports DefaultAzureCredential).
func WithAPIKey(apiKey string) Option { return func(o *Options) { o.apiKey = apiKey } }

// WithModel selects the LLM model.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithMaxTokens sets the max generation tokens.
func WithMaxTokens(maxTokens int64) Option { return func(o *Options) { o.maxTokens = maxTokens } }

// WithTemperature controls randomness.
func WithTemperature(t float64) Option { return func(o *Options) { o.temperature = &t } }

// WithTopP sets nucleus sampling probability mass.
func WithTopP(p float64) Option { return func(o *Options) { o.topP = &p } }

// WithTopK limits token selection to the top K candidates.
func WithTopK(k int64) Option { return func(o *Options) { o.topK = &k } }

// WithStopSequences sets text sequences that halt generation.
func WithStopSequences(seqs ...string) Option { return func(o *Options) { o.stopSequences = seqs } }

// WithTimeout sets the maximum duration to wait for API responses.
func WithTimeout(timeout time.Duration) Option { return func(o *Options) { o.timeout = &timeout } }

// WithEndpoint sets the Azure OpenAI endpoint URL.
func WithEndpoint(endpoint string) Option { return func(o *Options) { o.endpoint = endpoint } }

// WithAPIVersion sets the Azure OpenAI API version.
func WithAPIVersion(apiVersion string) Option {
	return func(o *Options) { o.apiVersion = apiVersion }
}

// Client implements [llm.LLM] against Azure OpenAI by delegating request handling
// to [llm/openai].Client constructed with Azure-specific SDK options.
type Client struct {
	*llmopenai.Client
}

// NewLLM constructs an Azure OpenAI LLM client.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	openaiOpts := []llmopenai.Option{
		llmopenai.WithModel(options.model),
		llmopenai.WithMaxTokens(options.maxTokens),
	}
	if options.temperature != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTemperature(*options.temperature))
	}
	if options.topP != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTopP(*options.topP))
	}
	if options.topK != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTopK(*options.topK))
	}
	if len(options.stopSequences) > 0 {
		openaiOpts = append(openaiOpts, llmopenai.WithStopSequences(options.stopSequences...))
	}
	if options.timeout != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTimeout(*options.timeout))
	}

	// If Azure-specific endpoint+apiVersion aren't set, fall through to plain OpenAI.
	if options.endpoint == "" || options.apiVersion == "" {
		if options.apiKey != "" {
			openaiOpts = append(openaiOpts, llmopenai.WithAPIKey(options.apiKey))
		}
		return llmopenai.NewLLM(openaiOpts...)
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(options.endpoint, options.apiVersion),
	}
	if options.apiKey != "" {
		reqOpts = append(reqOpts, azure.WithAPIKey(options.apiKey))
	} else if cred, err := azidentity.NewDefaultAzureCredential(nil); err == nil {
		reqOpts = append(reqOpts, azure.WithTokenCredential(cred))
	}

	// Build a fully-configured openai.Client with Azure auth, then hand it to the
	// openai package's bare Client implementation. Wrap in tracing here.
	bare := llmopenai.NewWithExistingClient(
		buildOpenAIOptions(options),
		openaisdk.NewClient(reqOpts...),
	)
	return llm.WithTracing(&Client{Client: bare}, llm.TracingAttrs{
		MaxTokens:   options.maxTokens,
		Temperature: options.temperature,
		TopP:        options.topP,
	})
}

// buildOpenAIOptions converts our Options to the embedded openai package's
// Options. The openai package keeps its options struct unexported, so we go
// through the option-func ladder.
func buildOpenAIOptions(o Options) llmopenai.Options {
	var dst llmopenai.Options
	llmopenai.WithModel(o.model)(&dst)
	llmopenai.WithMaxTokens(o.maxTokens)(&dst)
	if o.temperature != nil {
		llmopenai.WithTemperature(*o.temperature)(&dst)
	}
	if o.topP != nil {
		llmopenai.WithTopP(*o.topP)(&dst)
	}
	if o.topK != nil {
		llmopenai.WithTopK(*o.topK)(&dst)
	}
	if len(o.stopSequences) > 0 {
		llmopenai.WithStopSequences(o.stopSequences...)(&dst)
	}
	if o.timeout != nil {
		llmopenai.WithTimeout(*o.timeout)(&dst)
	}
	return dst
}
