package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type bedrockOptions struct {
}

type BedrockOption func(*bedrockOptions)

type bedrockClient struct {
	providerOptions llmClientOptions
	options         bedrockOptions
	childProvider   LLMClient
}

type BedrockClient LLMClient

func newBedrockClient(opts llmClientOptions) BedrockClient {
	bedrockOpts := bedrockOptions{}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}

	if region == "" {
		region = "us-east-1"
	}
	if len(region) < 2 {
		return &bedrockClient{
			providerOptions: opts,
			options:         bedrockOpts,
			childProvider:   nil,
		}
	}

	regionPrefix := region[:2]
	modelName := opts.model.APIModel
	opts.model.APIModel = fmt.Sprintf("%s.%s", regionPrefix, modelName)

	if strings.Contains(string(opts.model.APIModel), "anthropic") {
		anthropicOpts := opts
		anthropicOpts.anthropicOptions = append(anthropicOpts.anthropicOptions,
			WithAnthropicBedrock(true),
			WithAnthropicDisableCache(),
		)
		return &bedrockClient{
			providerOptions: opts,
			options:         bedrockOpts,
			childProvider:   newAnthropicClient(anthropicOpts),
		}
	}

	return &bedrockClient{
		providerOptions: opts,
		options:         bedrockOpts,
		childProvider:   nil,
	}
}

func (b *bedrockClient) send(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (*LLMResponse, error) {
	if b.childProvider == nil {
		return nil, errors.New("unsupported model for bedrock provider")
	}
	return b.childProvider.send(ctx, messages, tools)
}

func (b *bedrockClient) stream(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent {
	eventChan := make(chan LLMEvent)

	if b.childProvider == nil {
		go func() {
			eventChan <- LLMEvent{
				Type:  types.EventError,
				Error: errors.New("unsupported model for bedrock provider"),
			}
			close(eventChan)
		}()
		return eventChan
	}

	return b.childProvider.stream(ctx, messages, tools)
}
