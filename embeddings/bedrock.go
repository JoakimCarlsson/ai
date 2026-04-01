package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type bedrockOptions struct {
	region  string
	profile string
}

// BedrockOption configures Bedrock-specific embedding behavior.
type BedrockOption func(*bedrockOptions)

type bedrockClient struct {
	providerOptions embeddingClientOptions
	options         bedrockOptions
	client          *bedrockruntime.Client
}

// BedrockClient is the AWS Bedrock implementation of EmbeddingClient.
type BedrockClient EmbeddingClient

type titanEmbedRequest struct {
	InputText  string `json:"inputText"`
	Dimensions int    `json:"dimensions,omitempty"`
	Normalize  bool   `json:"normalize"`
}

type titanEmbedResponse struct {
	Embedding           []float32 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

type cohereBedrockEmbedRequest struct {
	Texts     []string `json:"texts"`
	InputType string   `json:"input_type,omitempty"`
	Truncate  string   `json:"truncate,omitempty"`
}

type cohereBedrockEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func newBedrockClient(
	opts embeddingClientOptions,
) BedrockClient {
	brOpts := bedrockOptions{
		region: "us-east-1",
	}
	for _, o := range opts.bedrockOptions {
		o(&brOpts)
	}

	cfgOpts := []func(*config.LoadOptions) error{
		config.WithRegion(brOpts.region),
	}
	if brOpts.profile != "" {
		cfgOpts = append(
			cfgOpts,
			config.WithSharedConfigProfile(brOpts.profile),
		)
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		cfgOpts...,
	)
	if err != nil {
		return nil
	}

	return &bedrockClient{
		providerOptions: opts,
		options:         brOpts,
		client:          bedrockruntime.NewFromConfig(cfg),
	}
}

func (b *bedrockClient) embed(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      b.providerOptions.model.APIModel,
		}, nil
	}

	modelID := b.providerOptions.model.APIModel

	if strings.HasPrefix(modelID, "cohere.") {
		return b.embedCohere(ctx, texts, inputType...)
	}

	return b.embedTitan(ctx, texts)
}

func (b *bedrockClient) embedTitan(
	ctx context.Context,
	texts []string,
) (*EmbeddingResponse, error) {
	var allEmbeddings [][]float32
	var totalTokens int64

	for _, text := range texts {
		reqBody := titanEmbedRequest{
			InputText: text,
			Normalize: true,
		}
		if b.providerOptions.dimensions != nil {
			reqBody.Dimensions = *b.providerOptions.dimensions
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal Titan embed request: %w",
				err,
			)
		}

		resp, err := b.client.InvokeModel(
			ctx,
			&bedrockruntime.InvokeModelInput{
				ModelId:     &b.providerOptions.model.APIModel,
				Body:        jsonBody,
				ContentType: strPtr("application/json"),
				Accept:      strPtr("application/json"),
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to invoke Titan embed model: %w",
				err,
			)
		}

		var titanResp titanEmbedResponse
		if err := json.Unmarshal(resp.Body, &titanResp); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal Titan response: %w",
				err,
			)
		}

		allEmbeddings = append(
			allEmbeddings,
			titanResp.Embedding,
		)
		totalTokens += int64(titanResp.InputTextTokenCount)
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{TotalTokens: totalTokens},
		Model:      b.providerOptions.model.APIModel,
	}, nil
}

func (b *bedrockClient) embedCohere(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*EmbeddingResponse, error) {
	batchSize := b.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 96
	}

	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		reqBody := cohereBedrockEmbedRequest{
			Texts: texts[i:end],
		}
		if len(inputType) > 0 && inputType[0] != "" {
			reqBody.InputType = inputType[0]
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal Cohere embed request: %w",
				err,
			)
		}

		resp, err := b.client.InvokeModel(
			ctx,
			&bedrockruntime.InvokeModelInput{
				ModelId:     &b.providerOptions.model.APIModel,
				Body:        jsonBody,
				ContentType: strPtr("application/json"),
				Accept:      strPtr("application/json"),
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to invoke Cohere embed model: %w",
				err,
			)
		}

		var cohereResp cohereBedrockEmbedResponse
		if err := json.Unmarshal(resp.Body, &cohereResp); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal Cohere response: %w",
				err,
			)
		}

		allEmbeddings = append(
			allEmbeddings,
			cohereResp.Embeddings...,
		)
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{},
		Model:      b.providerOptions.model.APIModel,
	}, nil
}

func (b *bedrockClient) embedMultimodal(
	_ context.Context,
	_ []MultimodalInput,
	_ ...string,
) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"bedrock does not support multimodal embeddings",
	)
}

func (b *bedrockClient) embedContextualized(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"bedrock does not support contextualized embeddings",
	)
}

func strPtr(s string) *string {
	return &s
}

// WithBedrockRegion sets the AWS region for the Bedrock endpoint.
func WithBedrockRegion(
	region string,
) BedrockOption {
	return func(options *bedrockOptions) {
		options.region = region
	}
}

// WithBedrockProfile sets the AWS shared config profile to use for credentials.
func WithBedrockProfile(
	profile string,
) BedrockOption {
	return func(options *bedrockOptions) {
		options.profile = profile
	}
}
