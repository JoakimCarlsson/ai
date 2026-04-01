package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type mistralOptions struct {
	outputDimension *int
	outputDtype     string
}

// MistralOption configures Mistral-specific embedding behavior.
type MistralOption func(*mistralOptions)

type mistralClient struct {
	providerOptions embeddingClientOptions
	options         mistralOptions
	httpClient      *http.Client
	baseURL         string
}

// MistralClient is the Mistral implementation of EmbeddingClient.
type MistralClient EmbeddingClient

type mistralEmbedRequest struct {
	Model           string   `json:"model"`
	Input           []string `json:"input"`
	OutputDimension *int     `json:"output_dimension,omitempty"`
	OutputDtype     string   `json:"output_dtype,omitempty"`
}

type mistralEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int64 `json:"prompt_tokens"`
		TotalTokens  int64 `json:"total_tokens"`
	} `json:"usage"`
}

func newMistralClient(
	opts embeddingClientOptions,
) MistralClient {
	mOpts := mistralOptions{}
	for _, o := range opts.mistralOptions {
		o(&mOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &mistralClient{
		providerOptions: opts,
		options:         mOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.mistral.ai/v1",
	}
}

func (m *mistralClient) embed(
	ctx context.Context,
	texts []string,
	_ ...string,
) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      m.providerOptions.model.APIModel,
		}, nil
	}

	batchSize := m.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 512
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := m.embedBatch(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(
			allEmbeddings,
			resp.Embeddings...,
		)
		totalTokens += resp.Usage.TotalTokens
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{TotalTokens: totalTokens},
		Model:      m.providerOptions.model.APIModel,
	}, nil
}

func (m *mistralClient) embedBatch(
	ctx context.Context,
	texts []string,
) (*EmbeddingResponse, error) {
	reqBody := mistralEmbedRequest{
		Model: m.providerOptions.model.APIModel,
		Input: texts,
	}

	if m.providerOptions.dimensions != nil {
		reqBody.OutputDimension = m.providerOptions.dimensions
	} else if m.options.outputDimension != nil {
		reqBody.OutputDimension = m.options.outputDimension
	}
	if m.options.outputDtype != "" {
		reqBody.OutputDtype = m.options.outputDtype
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal embed request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		m.baseURL+"/embeddings",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create embed request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(
		"Authorization",
		"Bearer "+m.providerOptions.apiKey,
	)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to make embed request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read embed response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embed API failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var mResp mistralEmbedResponse
	if err := json.Unmarshal(body, &mResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal embed response: %w",
			err,
		)
	}

	embeddings := make([][]float32, len(mResp.Data))
	for _, d := range mResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage: EmbeddingUsage{
			TotalTokens: mResp.Usage.TotalTokens,
		},
		Model: mResp.Model,
	}, nil
}

func (m *mistralClient) embedMultimodal(
	_ context.Context,
	_ []MultimodalInput,
	_ ...string,
) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"mistral does not support multimodal embeddings",
	)
}

func (m *mistralClient) embedContextualized(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf(
		"mistral does not support contextualized embeddings",
	)
}

// WithMistralOutputDimension sets the output embedding dimensionality (codestral-embed only).
func WithMistralOutputDimension(
	dim int,
) MistralOption {
	return func(options *mistralOptions) {
		options.outputDimension = &dim
	}
}

// WithMistralOutputDtype sets the output data type ("float", "int8", "uint8", "binary", "ubinary").
func WithMistralOutputDtype(
	dtype string,
) MistralOption {
	return func(options *mistralOptions) {
		options.outputDtype = dtype
	}
}
