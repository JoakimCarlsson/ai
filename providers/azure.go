package llm

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type AzureOption func(*azureOptions)

type azureOptions struct {
	endpoint   string
	apiVersion string
}

type azureClient struct {
	*openaiClient
}

type AzureClient LLMClient

func newAzureClient(opts llmClientOptions) AzureClient {
	azureOpts := &azureOptions{}
	for _, opt := range opts.azureOptions {
		opt(azureOpts)
	}

	if azureOpts.endpoint == "" || azureOpts.apiVersion == "" {
		return &azureClient{openaiClient: newOpenAIClient(opts).(*openaiClient)}
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(azureOpts.endpoint, azureOpts.apiVersion),
	}

	if opts.apiKey != "" {
		reqOpts = append(reqOpts, azure.WithAPIKey(opts.apiKey))
	} else if cred, err := azidentity.NewDefaultAzureCredential(nil); err == nil {
		reqOpts = append(reqOpts, azure.WithTokenCredential(cred))
	}

	base := &openaiClient{
		providerOptions: opts,
		client:          openai.NewClient(reqOpts...),
	}

	return &azureClient{openaiClient: base}
}

// WithAzureEndpoint sets the Azure OpenAI endpoint URL
func WithAzureEndpoint(endpoint string) AzureOption {
	return func(opts *azureOptions) {
		opts.endpoint = endpoint
	}
}

// WithAzureAPIVersion sets the Azure OpenAI API version
func WithAzureAPIVersion(apiVersion string) AzureOption {
	return func(opts *azureOptions) {
		opts.apiVersion = apiVersion
	}
}

// supportsStructuredOutput checks if the Azure client supports structured output
func (a *azureClient) supportsStructuredOutput() bool {
	return a.providerOptions.model.SupportsStructuredOut
}

// sendWithStructuredOutput sends a request with structured output to the Azure OpenAI client
func (a *azureClient) sendWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*LLMResponse, error) {
	return a.openaiClient.sendWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
}

// stream streams responses with structured output from the Azure OpenAI client
func (a *azureClient) streamWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan LLMEvent {
	return a.openaiClient.streamWithStructuredOutput(
		ctx,
		messages,
		tools,
		outputSchema,
	)
}
