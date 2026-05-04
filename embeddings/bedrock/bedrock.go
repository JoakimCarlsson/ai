// Package bedrock provides an AWS Bedrock implementation of the [embeddings.Embedding]
// interface, supporting both Titan and Cohere embedding models hosted on Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

// Options configures the Bedrock embeddings client.
type Options struct {
	model      model.EmbeddingModel
	batchSize  int
	dimensions *int
	region     string
	profile    string
}

// Option configures Options.
type Option func(*Options)

// WithModel selects the embedding model.
func WithModel(
	m model.EmbeddingModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithBatchSize sets the number of texts to process in each batch request (Cohere models only).
func WithBatchSize(
	batchSize int,
) Option {
	return func(o *Options) { o.batchSize = batchSize }
}

// WithDimensions specifies the output dimensionality for Titan embedding vectors.
func WithDimensions(
	dimensions int,
) Option {
	return func(o *Options) { o.dimensions = &dimensions }
}

// WithRegion sets the AWS region for the Bedrock endpoint.
func WithRegion(
	region string,
) Option {
	return func(o *Options) { o.region = region }
}

// WithProfile sets the AWS shared config profile to use for credentials.
func WithProfile(
	profile string,
) Option {
	return func(o *Options) { o.profile = profile }
}

// Client implements [embeddings.Embedding] against AWS Bedrock.
type Client struct {
	options Options
	client  *bedrockruntime.Client
}

// NewEmbedding constructs a Bedrock embeddings client.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{
		batchSize: 96,
		region:    "us-east-1",
	}
	for _, o := range opts {
		o(&options)
	}

	cfgOpts := []func(*config.LoadOptions) error{
		config.WithRegion(options.region),
	}
	if options.profile != "" {
		cfgOpts = append(
			cfgOpts,
			config.WithSharedConfigProfile(options.profile),
		)
	}

	cfg, _ := config.LoadDefaultConfig(context.Background(), cfgOpts...)

	return embeddings.WithTracing(&Client{
		options: options,
		client:  bedrockruntime.NewFromConfig(cfg),
	}, embeddings.TracingAttrs{
		Dimensions: options.dimensions,
	})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

type titanRequest struct {
	InputText  string `json:"inputText"`
	Dimensions int    `json:"dimensions,omitempty"`
	Normalize  bool   `json:"normalize"`
}

type titanResponse struct {
	Embedding           []float32 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

type cohereRequest struct {
	Texts     []string `json:"texts"`
	InputType string   `json:"input_type,omitempty"`
	Truncate  string   `json:"truncate,omitempty"`
}

type cohereResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// GenerateEmbeddings creates vector embeddings from text strings.
func (c *Client) GenerateEmbeddings(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &embeddings.EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      embeddings.EmbeddingUsage{TotalTokens: 0},
			Model:      c.options.model.APIModel,
		}, nil
	}

	if strings.HasPrefix(c.options.model.APIModel, "cohere.") {
		return c.embedCohere(ctx, texts, inputType...)
	}
	return c.embedTitan(ctx, texts)
}

func (c *Client) embedTitan(
	ctx context.Context,
	texts []string,
) (*embeddings.EmbeddingResponse, error) {
	var allEmbeddings [][]float32
	var totalTokens int64

	for _, text := range texts {
		reqBody := titanRequest{
			InputText: text,
			Normalize: true,
		}
		if c.options.dimensions != nil {
			reqBody.Dimensions = *c.options.dimensions
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal Titan embed request: %w",
				err,
			)
		}

		modelID := c.options.model.APIModel
		ct := "application/json"
		acc := "application/json"
		resp, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
			ModelId:     &modelID,
			Body:        jsonBody,
			ContentType: &ct,
			Accept:      &acc,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"failed to invoke Titan embed model: %w",
				err,
			)
		}

		var titanResp titanResponse
		if err := json.Unmarshal(resp.Body, &titanResp); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal Titan response: %w",
				err,
			)
		}

		allEmbeddings = append(allEmbeddings, titanResp.Embedding)
		totalTokens += int64(titanResp.InputTextTokenCount)
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{TotalTokens: totalTokens},
		Model:      c.options.model.APIModel,
	}, nil
}

func (c *Client) embedCohere(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	batchSize := c.options.batchSize
	if batchSize <= 0 {
		batchSize = 96
	}

	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		reqBody := cohereRequest{Texts: texts[i:end]}
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

		modelID := c.options.model.APIModel
		ct := "application/json"
		acc := "application/json"
		resp, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
			ModelId:     &modelID,
			Body:        jsonBody,
			ContentType: &ct,
			Accept:      &acc,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"failed to invoke Cohere embed model: %w",
				err,
			)
		}

		var cohereResp cohereResponse
		if err := json.Unmarshal(resp.Body, &cohereResp); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal Cohere response: %w",
				err,
			)
		}

		allEmbeddings = append(allEmbeddings, cohereResp.Embeddings...)
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{},
		Model:      c.options.model.APIModel,
	}, nil
}

// GenerateMultimodalEmbeddings is not supported by Bedrock.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	return nil, fmt.Errorf("bedrock does not support multimodal embeddings")
}

// GenerateContextualizedEmbeddings is not supported by Bedrock.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("bedrock does not support contextualized embeddings")
}
