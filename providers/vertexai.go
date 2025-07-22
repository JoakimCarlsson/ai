package llm

import (
	"context"
	"os"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

type vertexAIClient struct {
	*geminiClient
}

type VertexAIClient LLMClient

func newVertexAIClient(opts llmClientOptions) VertexAIClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  os.Getenv("VERTEXAI_PROJECT"),
		Location: os.Getenv("VERTEXAI_LOCATION"),
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil
	}

	base := &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}

	return &vertexAIClient{geminiClient: base}
}

func (v *vertexAIClient) supportsStructuredOutput() bool {
	return v.providerOptions.model.SupportsStructuredOut
}

func (v *vertexAIClient) sendWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) (*LLMResponse, error) {
	return v.geminiClient.sendWithStructuredOutput(ctx, messages, tools, outputSchema)
}

func (v *vertexAIClient) streamWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) <-chan LLMEvent {
	return v.geminiClient.streamWithStructuredOutput(ctx, messages, tools, outputSchema)
}
