package azure

import (
	"net/http"
	"strings"

	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Client delegates Responses requests to the OpenAI implementation.
type Client struct {
	*llmopenai.ResponsesClient
}

// WithOpenAIOptions passes [llmopenai] options through to the underlying
// client, for example built-in tools such as [llmopenai.WithWebSearch].
// Set authentication, endpoint, deployment and HTTP transport with Azure options.
func WithOpenAIOptions(opts ...llmopenai.Option) Option {
	return func(o *Options) {
		o.openaiOptions = append(o.openaiOptions, opts...)
	}
}

// WithStore sets storage for the Responses client. False preserves
// encrypted reasoning in message history; unencrypted items are dropped.
// Azure encrypted reasoning requires api-version v1 (/openai/v1/responses,
// without an api-version query); dated previews through 2025-04-01-preview
// lack support. See [llmopenai.WithStore] and
// https://learn.microsoft.com/en-us/azure/foundry/openai/api-version-lifecycle.
func WithStore(store bool) Option {
	return WithOpenAIOptions(llmopenai.WithStore(store))
}

// newResponsesClient builds the Azure Responses client from resolved options.
func newResponsesClient(options Options) llm.LLM {
	var responsesOptions llmopenai.Options
	for _, opt := range openAIOptionList(options) {
		opt(&responsesOptions)
	}

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
		&Client{ResponsesClient: bare},
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

// WithToolChoice controls whether and which tool the model may call. See
// [llmopenai.WithToolChoice].
func WithToolChoice(choice llm.ToolChoice) Option {
	return WithOpenAIOptions(llmopenai.WithToolChoice(choice))
}

// WithParallelToolCalls toggles whether the model may return several tool calls
// in one response. See [llmopenai.WithParallelToolCalls].
func WithParallelToolCalls(enabled bool) Option {
	return WithOpenAIOptions(llmopenai.WithParallelToolCalls(enabled))
}
