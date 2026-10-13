package azure

import (
	"net/http"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// ResponsesClient delegates Responses requests to the OpenAI implementation.
type ResponsesClient struct {
	*llmopenai.ResponsesClient
}

// WithResponsesOptions configures Responses-specific features, including
// built-in tools and reasoning, using the OpenAI Responses options.
// Set authentication, endpoint, deployment and HTTP transport with Azure options.
func WithResponsesOptions(opts ...llmopenai.ResponsesOption) Option {
	return func(o *Options) {
		o.responsesOptions = append(o.responsesOptions, opts...)
	}
}

// WithResponsesStore sets storage for the Responses client. False preserves
// encrypted reasoning in message history; unencrypted items are dropped.
// Azure encrypted reasoning requires api-version v1 (/openai/v1/responses,
// without an api-version query); dated previews through 2025-04-01-preview
// lack support. See [llmopenai.WithResponsesStore] and
// https://learn.microsoft.com/en-us/azure/foundry/openai/api-version-lifecycle.
func WithResponsesStore(store bool) Option {
	return WithResponsesOptions(llmopenai.WithResponsesStore(store))
}

// NewResponsesLLM constructs an Azure Responses client using deployment-based
// model resolution and Azure API-key or DefaultAzureCredential authentication.
// An endpoint ending in /openai/v1 uses v1 without an api-version query.
// A resource endpoint also defaults to v1 when WithAPIVersion is unset or v1;
// an explicit dated version uses /openai/responses?api-version=<version>.
// Responses-specific options can be passed with WithResponsesOptions.
func NewResponsesLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	var responsesOptions llmopenai.ResponsesOptions
	llmopenai.WithResponsesMaxTokens(options.maxTokens)(&responsesOptions)
	if options.temperature != nil {
		llmopenai.WithResponsesTemperature(
			*options.temperature,
		)(
			&responsesOptions,
		)
	}
	if options.topP != nil {
		llmopenai.WithResponsesTopP(*options.topP)(&responsesOptions)
	}
	if options.timeout != nil {
		llmopenai.WithResponsesTimeout(*options.timeout)(&responsesOptions)
	}
	if options.reasoningEffort != nil {
		llmopenai.WithResponsesReasoningEffort(
			*options.reasoningEffort,
		)(
			&responsesOptions,
		)
	}
	for _, opt := range options.responsesOptions {
		opt(&responsesOptions)
	}
	llmopenai.WithResponsesModel(modelFromOptions(options))(&responsesOptions)

	v1 := strings.HasSuffix(
		strings.TrimRight(options.endpoint, "/"),
		"/openai/v1",
	) ||
		options.apiVersion == "" ||
		options.apiVersion == "v1"
	if v1 {
		options.endpoint = strings.TrimSuffix(
			strings.TrimRight(options.endpoint, "/"),
			"/openai/v1",
		)
		options.apiVersion = "v1"
	}
	reqOpts := azureClientOptions(options)
	if v1 {
		reqOpts = append(reqOpts, option.WithMiddleware(azureResponsesV1))
	}
	bare := llmopenai.NewResponsesWithExistingClient(
		responsesOptions,
		openaisdk.NewClient(reqOpts...),
	)
	return llm.WithTracing(
		&ResponsesClient{ResponsesClient: bare},
		llm.TracingAttrs{
			MaxTokens:   options.maxTokens,
			Temperature: options.temperature,
			TopP:        options.topP,
		},
	)
}

// azureResponsesV1 maps the SDK's Azure route onto the versionless v1 endpoint.
func azureResponsesV1(
	req *http.Request,
	next option.MiddlewareNext,
) (*http.Response, error) {
	req.URL.Path = strings.Replace(req.URL.Path, "/openai/", "/openai/v1/", 1)
	query := req.URL.Query()
	query.Del("api-version")
	req.URL.RawQuery = query.Encode()
	return next(req)
}
