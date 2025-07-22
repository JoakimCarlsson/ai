package llm

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type azureClient struct {
	*openaiClient
}

type AzureClient LLMClient

func newAzureClient(opts llmClientOptions) AzureClient {

	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")

	if endpoint == "" || apiVersion == "" {
		return &azureClient{openaiClient: newOpenAIClient(opts).(*openaiClient)}
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(endpoint, apiVersion),
	}

	if opts.apiKey != "" || os.Getenv("AZURE_OPENAI_API_KEY") != "" {
		key := opts.apiKey
		if key == "" {
			key = os.Getenv("AZURE_OPENAI_API_KEY")
		}
		reqOpts = append(reqOpts, azure.WithAPIKey(key))
	} else if cred, err := azidentity.NewDefaultAzureCredential(nil); err == nil {
		reqOpts = append(reqOpts, azure.WithTokenCredential(cred))
	}

	base := &openaiClient{
		providerOptions: opts,
		client:          openai.NewClient(reqOpts...),
	}

	return &azureClient{openaiClient: base}
}

func (a *azureClient) supportsStructuredOutput() bool {
	return a.providerOptions.model.SupportsStructuredOut
}

func (a *azureClient) sendWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) (*LLMResponse, error) {
	return a.openaiClient.sendWithStructuredOutput(ctx, messages, tools, outputSchema)
}

func (a *azureClient) streamWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) <-chan LLMEvent {
	return a.openaiClient.streamWithStructuredOutput(ctx, messages, tools, outputSchema)
}
