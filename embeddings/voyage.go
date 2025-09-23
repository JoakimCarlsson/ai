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

type voyageOptions struct {
	inputType       string
	truncation      *bool
	outputDimension *int
	outputDtype     string
	encodingFormat  string
}

type VoyageOption func(*voyageOptions)

type voyageClient struct {
	providerOptions embeddingClientOptions
	options         voyageOptions
	httpClient      *http.Client
	baseURL         string
}

type VoyageClient EmbeddingClient

type voyageEmbeddingRequest struct {
	Input           []string `json:"input"`
	Model           string   `json:"model"`
	InputType       string   `json:"input_type,omitempty"`
	Truncation      *bool    `json:"truncation,omitempty"`
	OutputDimension *int     `json:"output_dimension,omitempty"`
	OutputDtype     string   `json:"output_dtype,omitempty"`
	EncodingFormat  string   `json:"encoding_format,omitempty"`
}

type voyageEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
		TextTokens  int64 `json:"text_tokens,omitempty"`
		ImagePixels int64 `json:"image_pixels,omitempty"`
	} `json:"usage"`
}

type voyageMultimodalRequest struct {
	Inputs         []MultimodalInput `json:"inputs"`
	Model          string            `json:"model"`
	InputType      string            `json:"input_type,omitempty"`
	Truncation     *bool             `json:"truncation,omitempty"`
	OutputEncoding string            `json:"output_encoding,omitempty"`
}

type voyageContextualizedRequest struct {
	Inputs    [][]string `json:"inputs"`
	Model     string     `json:"model"`
	InputType string     `json:"input_type,omitempty"`
}

type voyageContextualizedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object string `json:"object"`
		Data   []struct {
			Object    string    `json:"object"`
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Index int `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
	} `json:"usage"`
}

func newVoyageClient(opts embeddingClientOptions) VoyageClient {
	voyageOpts := voyageOptions{
		inputType:      "document",
		outputDtype:    "float",
		encodingFormat: "",
	}
	for _, o := range opts.voyageOptions {
		o(&voyageOpts)
	}

	timeout := 30 * time.Second
	if opts.timeout != nil {
		timeout = *opts.timeout
	}

	return &voyageClient{
		providerOptions: opts,
		options:         voyageOpts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.voyageai.com/v1",
	}
}

func (v *voyageClient) embed(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      v.providerOptions.model.APIModel,
		}, nil
	}

	batchSize := v.providerOptions.batchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	var allEmbeddings [][]float32
	var totalTokens int64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		response, err := v.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch: %w", err)
		}

		allEmbeddings = append(allEmbeddings, response.Embeddings...)
		totalTokens += response.Usage.TotalTokens
	}

	return &EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      EmbeddingUsage{TotalTokens: totalTokens},
		Model:      v.providerOptions.model.APIModel,
	}, nil
}

func (v *voyageClient) embedBatch(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	reqBody := voyageEmbeddingRequest{
		Input:           texts,
		Model:           v.providerOptions.model.APIModel,
		InputType:       v.options.inputType,
		Truncation:      v.options.truncation,
		OutputDimension: v.options.outputDimension,
		OutputDtype:     v.options.outputDtype,
		EncodingFormat:  v.options.encodingFormat,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", v.baseURL+"/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.providerOptions.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var voyageResp voyageEmbeddingResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	embeddings := make([][]float32, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		embeddings[i] = data.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage: EmbeddingUsage{
			TotalTokens: voyageResp.Usage.TotalTokens,
			TextTokens:  voyageResp.Usage.TextTokens,
			ImagePixels: voyageResp.Usage.ImagePixels,
		},
		Model: voyageResp.Model,
	}, nil
}

func (v *voyageClient) embedMultimodal(ctx context.Context, inputs []MultimodalInput) (*EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      EmbeddingUsage{TotalTokens: 0},
			Model:      v.providerOptions.model.APIModel,
		}, nil
	}

	reqBody := voyageMultimodalRequest{
		Inputs:         inputs,
		Model:          v.providerOptions.model.APIModel,
		InputType:      v.options.inputType,
		Truncation:     v.options.truncation,
		OutputEncoding: v.options.encodingFormat,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal multimodal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", v.baseURL+"/multimodalembeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create multimodal request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.providerOptions.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make multimodal request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read multimodal response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("multimodal API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var voyageResp voyageEmbeddingResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multimodal response: %w", err)
	}

	embeddings := make([][]float32, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		embeddings[i] = data.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage: EmbeddingUsage{
			TotalTokens: voyageResp.Usage.TotalTokens,
			TextTokens:  voyageResp.Usage.TextTokens,
			ImagePixels: voyageResp.Usage.ImagePixels,
		},
		Model: voyageResp.Model,
	}, nil
}

func (v *voyageClient) embedContextualized(ctx context.Context, documentChunks [][]string) (*ContextualizedEmbeddingResponse, error) {
	if len(documentChunks) == 0 {
		return &ContextualizedEmbeddingResponse{
			DocumentEmbeddings: [][][]float32{},
			Usage:              EmbeddingUsage{TotalTokens: 0},
			Model:              v.providerOptions.model.APIModel,
		}, nil
	}

	reqBody := voyageContextualizedRequest{
		Inputs:    documentChunks,
		Model:     v.providerOptions.model.APIModel,
		InputType: v.options.inputType,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contextualized request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", v.baseURL+"/contextualizedembeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create contextualized request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.providerOptions.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make contextualized request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read contextualized response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contextualized API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var voyageResp voyageContextualizedResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contextualized response: %w", err)
	}

	documentEmbeddings := make([][][]float32, len(voyageResp.Data))
	for docIndex, docData := range voyageResp.Data {
		chunkEmbeddings := make([][]float32, len(docData.Data))
		for chunkIndex, chunkData := range docData.Data {
			chunkEmbeddings[chunkIndex] = chunkData.Embedding
		}
		documentEmbeddings[docIndex] = chunkEmbeddings
	}

	return &ContextualizedEmbeddingResponse{
		DocumentEmbeddings: documentEmbeddings,
		Usage:              EmbeddingUsage{TotalTokens: voyageResp.Usage.TotalTokens},
		Model:              voyageResp.Model,
	}, nil
}

func WithInputType(inputType string) VoyageOption {
	return func(options *voyageOptions) {
		options.inputType = inputType
	}
}

func WithTruncation(truncation bool) VoyageOption {
	return func(options *voyageOptions) {
		options.truncation = &truncation
	}
}

func WithEncodingFormat(format string) VoyageOption {
	return func(options *voyageOptions) {
		options.encodingFormat = format
	}
}

func WithOutputDimension(dimension int) VoyageOption {
	return func(options *voyageOptions) {
		options.outputDimension = &dimension
	}
}

func WithOutputDtype(dtype string) VoyageOption {
	return func(options *voyageOptions) {
		options.outputDtype = dtype
	}
}
