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
	reasoningEffort       *llmopenai.ReasoningEffort
	supportsStructuredOut *bool
	attachments           *bool
	httpClient            *http.Client
	openaiOptions         []llmopenai.Option
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
// string against [Models] (exact APIModel match first, then a
// substring fallback for custom deployment names like "gpt-4.1-nano-prod").
// When no match is found, a minimal [llm.Model] is used with the deployment
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
// deployment name does not match an entry in [Models].
func WithReasoning(canReason bool) Option {
	return func(o *Options) { o.canReason = &canReason }
}

// WithReasoningEffort sets effort for reasoning deployments. ReasoningEffortNone
// disables reasoning on models that accept it. Unset uses the provider default;
// deployments without CanReason ignore this option.
func WithReasoningEffort(effort llmopenai.ReasoningEffort) Option {
	return func(o *Options) { o.reasoningEffort = &effort }
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
// the deployment name does not match an entry in [Models].
func WithStructuredOutput(supportsStructuredOutput bool) Option {
	return func(o *Options) { o.supportsStructuredOut = &supportsStructuredOutput }
}

// WithAttachments declares whether the deployed model accepts image
// attachments. Azure routes on deployment name, so a deployment whose name
// does not appear in [Models] resolves to a model with every
// capability false; this is how a caller states otherwise.
func WithAttachments(supportsAttachments bool) Option {
	return func(o *Options) { o.attachments = &supportsAttachments }
}

// NewLLM constructs an Azure OpenAI LLM client targeting the Responses API,
// using deployment-based model resolution and Azure API-key or
// DefaultAzureCredential authentication. An endpoint ending in /openai/v1 uses
// v1 without an api-version query. A resource endpoint also defaults to v1 when
// WithAPIVersion is unset or v1; an explicit dated version uses
// /openai/responses?api-version=<version>. Without an endpoint the client talks
// to OpenAI directly. Built-in tools and other OpenAI options can be passed
// with [WithOpenAIOptions].
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	if options.endpoint == "" {
		openaiOpts := append(
			openAIOptionList(options),
			llmopenai.WithAPIKey(options.apiKey),
		)
		return llmopenai.NewLLM(openaiOpts...)
	}
	return newResponsesClient(options)
}

// openAIOptionList converts the Azure options into [llmopenai] options,
// followed by any options passed through [WithOpenAIOptions].
func openAIOptionList(o Options) []llmopenai.Option {
	opts := []llmopenai.Option{
		llmopenai.WithModel(modelFromOptions(o)),
		llmopenai.WithMaxTokens(o.maxTokens),
	}
	if o.reasoningEffort != nil {
		opts = append(opts, llmopenai.WithReasoningEffort(*o.reasoningEffort))
	}
	if o.temperature != nil {
		opts = append(opts, llmopenai.WithTemperature(*o.temperature))
	}
	if o.topP != nil {
		opts = append(opts, llmopenai.WithTopP(*o.topP))
	}
	if o.topK != nil {
		opts = append(opts, llmopenai.WithTopK(*o.topK))
	}
	if len(o.stopSequences) > 0 {
		opts = append(opts, llmopenai.WithStopSequences(o.stopSequences...))
	}
	if o.timeout != nil {
		opts = append(opts, llmopenai.WithTimeout(*o.timeout))
	}
	if o.httpClient != nil {
		opts = append(opts, llmopenai.WithHTTPClient(o.httpClient))
	}
	return append(opts, o.openaiOptions...)
}

// resolveDeployment maps an Azure deployment name to a [llm.Model] carrying
// cost and context metadata. The deployment string always becomes APIModel so
// requests route correctly; metadata is filled in from [Models] by
// exact APIModel match, then by substring fallback (so a deployment named
// "gpt-4.1-nano-prod" inherits metadata from the "gpt-4.1-nano" entry).
//
// Azure Foundry deployment names are opaque — callers whose deployment
// doesn't match any registry entry should declare capabilities explicitly
// via [WithReasoning], [WithStructuredOutput] and [WithAttachments].
func resolveDeployment(deployment string) llm.Model {
	if deployment == "" {
		return llm.Model{}
	}
	if m, ok := findAzureModel(deployment, true); ok {
		m.APIModel = deployment
		return m
	}
	if m, ok := findAzureModel(deployment, false); ok {
		m.APIModel = deployment
		return m
	}
	return llm.Model{
		APIModel: deployment,
		Provider: "azure",
	}
}

// modelFromOptions resolves the deployment and applies any explicit
// capability overrides from [WithReasoning] / [WithStructuredOutput] /
// [WithAttachments].
func modelFromOptions(o Options) llm.Model {
	m := resolveDeployment(o.deployment)
	if o.canReason != nil {
		m.CanReason = *o.canReason
	}
	if o.supportsStructuredOut != nil {
		m.SupportsStructuredOut = *o.supportsStructuredOut
	}
	if o.attachments != nil {
		m.SupportsAttachments = *o.attachments
	}
	return m
}

// findAzureModel selects an exact match or the longest model-name substring.
func findAzureModel(deployment string, exact bool) (llm.Model, bool) {
	var best llm.Model
	var bestLen int
	found := false
	for _, m := range Models {
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

// azureClientOptions configures the shared Azure endpoint and authentication.
func azureClientOptions(options Options) []option.RequestOption {
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

	return reqOpts
}
