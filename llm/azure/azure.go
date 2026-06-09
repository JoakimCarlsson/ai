// Package azure provides an Azure OpenAI implementation of the [llm.LLM] interface.
//
// Azure's request/response semantics are identical to OpenAI's, so this package
// embeds [llm/openai].Client and overrides only the SDK construction (custom auth
// + endpoint).
package azure

import (
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
)

// Options configures the Azure OpenAI LLM client.
type Options struct {
	apiKey                string
	deployment            string
	maxTokens             int64
	temperature           *float64
	topP                  *float64
	topK                  *int64
	stopSequences         []string
	timeout               *time.Duration
	endpoint              string
	apiVersion            string
	canReason             *bool
	supportsStructuredOut *bool
	httpClient            *http.Client
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key (optional — Azure also supports DefaultAzureCredential).
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithDeployment sets the Azure OpenAI deployment name. Azure routes requests
// purely on deployment, so this is the only model selector callers need to
// provide. Cost and context metadata are resolved by matching the deployment
// string against [model.AzureModels] (exact APIModel match first, then a
// substring fallback for custom deployment names like "gpt-4.1-nano-prod").
// When no match is found, a minimal [model.Model] is used with the deployment
// as APIModel.
func WithDeployment(deployment string) Option {
	return func(o *Options) { o.deployment = deployment }
}

// WithMaxTokens sets the max generation tokens.
func WithMaxTokens(
	maxTokens int64,
) Option {
	return func(o *Options) { o.maxTokens = maxTokens }
}

// WithTemperature controls randomness.
func WithTemperature(
	t float64,
) Option {
	return func(o *Options) { o.temperature = &t }
}

// WithTopP sets nucleus sampling probability mass.
func WithTopP(p float64) Option { return func(o *Options) { o.topP = &p } }

// WithTopK limits token selection to the top K candidates.
func WithTopK(k int64) Option { return func(o *Options) { o.topK = &k } }

// WithStopSequences sets text sequences that halt generation.
func WithStopSequences(
	seqs ...string,
) Option {
	return func(o *Options) { o.stopSequences = seqs }
}

// WithTimeout sets the maximum duration to wait for API responses.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithEndpoint sets the Azure OpenAI endpoint URL.
func WithEndpoint(
	endpoint string,
) Option {
	return func(o *Options) { o.endpoint = endpoint }
}

// WithAPIVersion sets the Azure OpenAI API version.
func WithAPIVersion(apiVersion string) Option {
	return func(o *Options) { o.apiVersion = apiVersion }
}

// WithReasoning declares whether the deployed model supports reasoning
// semantics. When true the chat-completions request emits
// max_completion_tokens instead of max_tokens — required by gpt-5.x,
// o-series, and most newer Azure Foundry deployments. Use this when the
// deployment name does not match an entry in [model.AzureModels].
func WithReasoning(canReason bool) Option {
	return func(o *Options) { o.canReason = &canReason }
}

// WithHTTPClient injects a custom *http.Client, threaded into the OpenAI SDK
// (and the Azure-auth SDK path) via option.WithHTTPClient. Use it for outbound
// proxies, custom TLS (private CAs, mTLS), connection-pool tuning, or
// transport-level instrumentation. A nil client is a no-op, leaving the SDK
// default client in place. The per-request context timeout from WithTimeout
// still applies on top of the injected client's transport: the two compose and
// the shorter deadline wins.
func WithHTTPClient(c *http.Client) Option {
	return func(o *Options) { o.httpClient = c }
}

// WithStructuredOutput declares whether the deployed model supports the
// chat-completions response_format JSON-schema constraint. Use this when
// the deployment name does not match an entry in [model.AzureModels].
func WithStructuredOutput(supportsStructuredOutput bool) Option {
	return func(o *Options) { o.supportsStructuredOut = &supportsStructuredOutput }
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

	resolvedModel := modelFromOptions(options)

	openaiOpts := []llmopenai.Option{
		llmopenai.WithModel(resolvedModel),
		llmopenai.WithMaxTokens(options.maxTokens),
	}
	if options.temperature != nil {
		openaiOpts = append(
			openaiOpts,
			llmopenai.WithTemperature(*options.temperature),
		)
	}
	if options.topP != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTopP(*options.topP))
	}
	if options.topK != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTopK(*options.topK))
	}
	if len(options.stopSequences) > 0 {
		openaiOpts = append(
			openaiOpts,
			llmopenai.WithStopSequences(options.stopSequences...),
		)
	}
	if options.timeout != nil {
		openaiOpts = append(openaiOpts, llmopenai.WithTimeout(*options.timeout))
	}
	if options.httpClient != nil {
		openaiOpts = append(
			openaiOpts,
			llmopenai.WithHTTPClient(options.httpClient),
		)
	}

	// If Azure-specific endpoint+apiVersion aren't set, fall through to plain OpenAI.
	if options.endpoint == "" || options.apiVersion == "" {
		if options.apiKey != "" {
			openaiOpts = append(
				openaiOpts,
				llmopenai.WithAPIKey(options.apiKey),
			)
		}
		return llmopenai.NewLLM(openaiOpts...)
	}

	// Azure's v1 endpoint (".../openai/v1") is OpenAI-compatible: it does not
	// accept the api-version query param and uses /chat/completions directly,
	// not /openai/deployments/<name>/chat/completions. Route via the plain
	// OpenAI client with WithBaseURL.
	if strings.Contains(options.endpoint, "/openai/v1") {
		openaiOpts = append(
			openaiOpts,
			llmopenai.WithBaseURL(strings.TrimRight(options.endpoint, "/")),
		)
		if options.apiKey != "" {
			openaiOpts = append(
				openaiOpts,
				llmopenai.WithAPIKey(options.apiKey),
			)
		}
		return llmopenai.NewLLM(openaiOpts...)
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(options.endpoint, options.apiVersion),
	}
	if options.httpClient != nil {
		reqOpts = append(reqOpts, option.WithHTTPClient(options.httpClient))
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

// resolveDeployment maps an Azure deployment name to a [model.Model] carrying
// cost and context metadata. The deployment string always becomes APIModel so
// requests route correctly; metadata is filled in from [model.AzureModels] by
// exact APIModel match, then by substring fallback (so a deployment named
// "gpt-4.1-nano-prod" inherits metadata from the "gpt-4.1-nano" entry).
//
// Azure Foundry deployment names are opaque — callers whose deployment
// doesn't match any registry entry should declare capabilities explicitly
// via [WithReasoning] and [WithStructuredOutput].
func resolveDeployment(deployment string) model.Model {
	if deployment == "" {
		return model.Model{}
	}
	if m, ok := findAzureModel(deployment, true); ok {
		m.APIModel = deployment
		return m
	}
	if m, ok := findAzureModel(deployment, false); ok {
		m.APIModel = deployment
		return m
	}
	return model.Model{
		APIModel: deployment,
		Provider: model.ProviderAzure,
	}
}

// modelFromOptions resolves the deployment and applies any explicit
// capability overrides from [WithReasoning] / [WithStructuredOutput].
func modelFromOptions(o Options) model.Model {
	m := resolveDeployment(o.deployment)
	if o.canReason != nil {
		m.CanReason = *o.canReason
	}
	if o.supportsStructuredOut != nil {
		m.SupportsStructuredOut = *o.supportsStructuredOut
	}
	return m
}

func findAzureModel(deployment string, exact bool) (model.Model, bool) {
	var best model.Model
	var bestLen int
	found := false
	for _, m := range model.AzureModels {
		if exact {
			if m.APIModel == deployment {
				return m, true
			}
			continue
		}
		if m.APIModel != "" && strings.Contains(deployment, m.APIModel) {
			if len(m.APIModel) > bestLen {
				best = m
				bestLen = len(m.APIModel)
				found = true
			}
		}
	}
	return best, found
}

// buildOpenAIOptions converts our Options to the embedded openai package's
// Options. The openai package keeps its options struct unexported, so we go
// through the option-func ladder.
func buildOpenAIOptions(o Options) llmopenai.Options {
	var dst llmopenai.Options
	llmopenai.WithModel(modelFromOptions(o))(&dst)
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
