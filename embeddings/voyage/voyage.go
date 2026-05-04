// Package voyage provides a Voyage AI implementation of the [embeddings.Embedding] interface,
// supporting standard, multimodal, and contextualized embeddings.
package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

const defaultBaseURL = "https://api.voyageai.com/v1"

// EmbeddingVector holds a Voyage API embedding in one of several numeric or base64 encodings.
type EmbeddingVector struct {
	Float32  []float32 `json:"-"`
	Int8     []int8    `json:"-"`
	Uint8    []uint8   `json:"-"`
	Binary   []int8    `json:"-"`
	UBinary  []uint8   `json:"-"`
	Base64   string    `json:"-"`
	DataType string    `json:"-"`
}

// UnmarshalJSON decodes a Voyage embedding from JSON (array of numbers, string base64, etc.).
func (ev *EmbeddingVector) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case string:
		ev.Base64 = v
		ev.DataType = "base64"
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("empty embedding vector")
		}

		switch first := v[0].(type) {
		case float64:
			ev.Float32 = make([]float32, len(v))
			for i, val := range v {
				if f, ok := val.(float64); ok {
					ev.Float32[i] = float32(f)
				} else {
					return fmt.Errorf("mixed types in float embedding at index %d", i)
				}
			}
			ev.DataType = "float32"
		case float32:
			ev.Float32 = make([]float32, len(v))
			for i, val := range v {
				if f, ok := val.(float32); ok {
					ev.Float32[i] = f
				} else {
					return fmt.Errorf("mixed types in float32 embedding at index %d", i)
				}
			}
			ev.DataType = "float32"
		default:
			intVal, ok := first.(float64)
			if !ok || intVal != float64(int(intVal)) {
				return fmt.Errorf("unsupported embedding value type: %T", first)
			}
			switch {
			case intVal >= -128 && intVal <= 127:
				ev.Int8 = make([]int8, len(v))
				for i, val := range v {
					if f, ok := val.(float64); ok && f == float64(int(f)) {
						ev.Int8[i] = int8(f)
					} else {
						return fmt.Errorf("invalid int8 value at index %d", i)
					}
				}
				ev.DataType = "int8"
			case intVal >= 0 && intVal <= 255:
				ev.Uint8 = make([]uint8, len(v))
				for i, val := range v {
					if f, ok := val.(float64); ok && f == float64(int(f)) {
						ev.Uint8[i] = uint8(f)
					} else {
						return fmt.Errorf("invalid uint8 value at index %d", i)
					}
				}
				ev.DataType = "uint8"
			default:
				return fmt.Errorf("integer value out of range: %v", intVal)
			}
		}
	default:
		return fmt.Errorf("unsupported embedding type: %T", v)
	}

	return nil
}

// ToFloat32 returns the embedding as float32 values when the stored type is convertible.
func (ev *EmbeddingVector) ToFloat32() []float32 {
	switch ev.DataType {
	case "float32":
		return ev.Float32
	case "int8":
		result := make([]float32, len(ev.Int8))
		for i, v := range ev.Int8 {
			result[i] = float32(v)
		}
		return result
	case "uint8":
		result := make([]float32, len(ev.Uint8))
		for i, v := range ev.Uint8 {
			result[i] = float32(v)
		}
		return result
	case "binary":
		result := make([]float32, len(ev.Binary))
		for i, v := range ev.Binary {
			result[i] = float32(v)
		}
		return result
	case "ubinary":
		result := make([]float32, len(ev.UBinary))
		for i, v := range ev.UBinary {
			result[i] = float32(v)
		}
		return result
	case "base64":
		return nil
	default:
		return nil
	}
}

// Len returns the logical length of the embedding for the active data type.
func (ev *EmbeddingVector) Len() int {
	switch ev.DataType {
	case "float32":
		return len(ev.Float32)
	case "int8", "binary":
		return len(ev.Int8)
	case "uint8", "ubinary":
		return len(ev.Uint8)
	case "base64":
		return 0
	default:
		return 0
	}
}

// GetDataType returns the detected embedding encoding label.
func (ev *EmbeddingVector) GetDataType() string { return ev.DataType }

// IsBase64 reports whether the embedding was parsed as a base64 string payload.
func (ev *EmbeddingVector) IsBase64() bool { return ev.DataType == "base64" }

// Options configures the Voyage embeddings client.
type Options struct {
	apiKey          string
	model           model.EmbeddingModel
	timeout         *time.Duration
	batchSize       int
	dimensions      *int
	inputType       string
	truncation      *bool
	outputDimension *int
	outputDtype     string
	encodingFormat  string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Voyage.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the embedding model.
func WithModel(
	m model.EmbeddingModel,
) Option {
	return func(o *Options) { o.model = m }
}

// WithTimeout sets the maximum duration to wait for a single request.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(
	batchSize int,
) Option {
	return func(o *Options) { o.batchSize = batchSize }
}

// WithDimensions specifies the output dimensionality for embedding vectors.
// Equivalent to [WithOutputDimensions]; Voyage accepts either.
func WithDimensions(
	dimensions int,
) Option {
	return func(o *Options) { o.dimensions = &dimensions }
}

// WithInputType specifies the type of input for optimized embedding generation.
// Common values: "query" for search queries, "document" for documents to be searched.
func WithInputType(
	inputType string,
) Option {
	return func(o *Options) { o.inputType = inputType }
}

// WithTruncation enables automatic truncation of inputs exceeding the model's token limit.
func WithTruncation(
	truncation bool,
) Option {
	return func(o *Options) { o.truncation = &truncation }
}

// WithEncodingFormat specifies the format for encoded embeddings.
func WithEncodingFormat(format string) Option {
	return func(o *Options) { o.encodingFormat = format }
}

// WithOutputDimensions sets the dimensionality of the output embedding vectors.
func WithOutputDimensions(dimensions int) Option {
	return func(o *Options) { o.outputDimension = &dimensions }
}

// WithOutputDtype specifies the data type for embedding outputs.
// Common values: "float" (default), "int8", "uint8", "binary", "ubinary".
func WithOutputDtype(
	dtype string,
) Option {
	return func(o *Options) { o.outputDtype = dtype }
}

// Client implements [embeddings.Embedding] against the Voyage AI API.
type Client struct {
	options    Options
	httpClient *http.Client
	baseURL    string
}

// NewEmbedding constructs a Voyage embeddings client. The returned [embeddings.Embedding]
// is wrapped with [embeddings.WithTracing], so callers always get tracing spans and metrics.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{
		batchSize:   100,
		outputDtype: "float",
	}
	for _, o := range opts {
		o(&options)
	}

	timeout := 30 * time.Second
	if options.timeout != nil {
		timeout = *options.timeout
	}
	dimensions := options.dimensions
	if options.outputDimension != nil {
		dimensions = options.outputDimension
	}

	return embeddings.WithTracing(&Client{
		options:    options,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	}, embeddings.TracingAttrs{
		Dimensions: dimensions,
	})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

type embedRequest struct {
	Input           []string `json:"input"`
	Model           string   `json:"model"`
	InputType       string   `json:"input_type,omitempty"`
	Truncation      *bool    `json:"truncation,omitempty"`
	OutputDimension *int     `json:"output_dimension,omitempty"`
	OutputDtype     string   `json:"output_dtype,omitempty"`
	EncodingFormat  string   `json:"encoding_format,omitempty"`
}

type embedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string          `json:"object"`
		Embedding EmbeddingVector `json:"embedding"`
		Index     int             `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
		TextTokens  int64 `json:"text_tokens,omitempty"`
		ImagePixels int64 `json:"image_pixels,omitempty"`
	} `json:"usage"`
}

type multimodalRequest struct {
	Inputs         []embeddings.MultimodalInput `json:"inputs"`
	Model          string                       `json:"model"`
	InputType      string                       `json:"input_type,omitempty"`
	Truncation     *bool                        `json:"truncation,omitempty"`
	OutputEncoding string                       `json:"output_encoding,omitempty"`
}

type contextualizedRequest struct {
	Inputs    [][]string `json:"inputs"`
	Model     string     `json:"model"`
	InputType string     `json:"input_type,omitempty"`
}

type contextualizedResponse struct {
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

	batchSize := c.options.batchSize
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

		response, err := c.embedBatch(ctx, texts[i:end], inputType...)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch: %w", err)
		}

		allEmbeddings = append(allEmbeddings, response.Embeddings...)
		totalTokens += response.Usage.TotalTokens
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{TotalTokens: totalTokens},
		Model:      c.options.model.APIModel,
	}, nil
}

func (c *Client) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	reqBody := embedRequest{
		Input: texts,
		Model: c.options.model.APIModel,
	}

	if len(inputType) > 0 && inputType[0] != "" {
		reqBody.InputType = inputType[0]
	} else if c.options.inputType != "" {
		reqBody.InputType = c.options.inputType
	}
	if c.options.truncation != nil {
		reqBody.Truncation = c.options.truncation
	}
	if c.options.dimensions != nil {
		reqBody.OutputDimension = c.options.dimensions
	} else if c.options.outputDimension != nil {
		reqBody.OutputDimension = c.options.outputDimension
	}
	if c.options.outputDtype != "float" {
		reqBody.OutputDtype = c.options.outputDtype
	}
	if c.options.encodingFormat != "" {
		reqBody.EncodingFormat = c.options.encodingFormat
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, "POST", c.baseURL+"/embeddings", bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var voyageResp embedResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	out := make([][]float32, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		embedding := data.Embedding.ToFloat32()
		if embedding == nil {
			return nil, fmt.Errorf(
				"failed to convert embedding at index %d: unsupported data type %s",
				i,
				data.Embedding.DataType,
			)
		}
		out[i] = embedding
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage: embeddings.EmbeddingUsage{
			TotalTokens: voyageResp.Usage.TotalTokens,
			TextTokens:  voyageResp.Usage.TextTokens,
			ImagePixels: voyageResp.Usage.ImagePixels,
		},
		Model: voyageResp.Model,
	}, nil
}

// GenerateMultimodalEmbeddings creates embeddings from mixed text and image content.
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	inputs []embeddings.MultimodalInput,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	if len(inputs) == 0 {
		return &embeddings.EmbeddingResponse{
			Embeddings: [][]float32{},
			Usage:      embeddings.EmbeddingUsage{TotalTokens: 0},
			Model:      c.options.model.APIModel,
		}, nil
	}

	reqBody := multimodalRequest{
		Inputs: inputs,
		Model:  c.options.model.APIModel,
	}

	if len(inputType) > 0 && inputType[0] != "" {
		reqBody.InputType = inputType[0]
	} else if c.options.inputType != "" {
		reqBody.InputType = c.options.inputType
	}

	if c.options.truncation != nil {
		reqBody.Truncation = c.options.truncation
	}

	if c.options.encodingFormat != "" {
		reqBody.OutputEncoding = c.options.encodingFormat
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal multimodal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/multimodalembeddings",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create multimodal request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make multimodal request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read multimodal response body: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"multimodal API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var voyageResp embedResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal multimodal response: %w",
			err,
		)
	}

	out := make([][]float32, len(voyageResp.Data))
	for i, data := range voyageResp.Data {
		embedding := data.Embedding.ToFloat32()
		if embedding == nil {
			return nil, fmt.Errorf(
				"failed to convert multimodal embedding at index %d: unsupported data type %s",
				i,
				data.Embedding.DataType,
			)
		}
		out[i] = embedding
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage: embeddings.EmbeddingUsage{
			TotalTokens: voyageResp.Usage.TotalTokens,
			TextTokens:  voyageResp.Usage.TextTokens,
			ImagePixels: voyageResp.Usage.ImagePixels,
		},
		Model: voyageResp.Model,
	}, nil
}

// GenerateContextualizedEmbeddings creates embeddings where each chunk is aware of its document context.
func (c *Client) GenerateContextualizedEmbeddings(
	ctx context.Context,
	documentChunks [][]string,
	inputType ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	if len(documentChunks) == 0 {
		return &embeddings.ContextualizedEmbeddingResponse{
			DocumentEmbeddings: [][][]float32{},
			Usage:              embeddings.EmbeddingUsage{TotalTokens: 0},
			Model:              c.options.model.APIModel,
		}, nil
	}

	reqBody := contextualizedRequest{
		Inputs: documentChunks,
		Model:  c.options.model.APIModel,
	}

	if len(inputType) > 0 && inputType[0] != "" {
		reqBody.InputType = inputType[0]
	} else if c.options.inputType != "" {
		reqBody.InputType = c.options.inputType
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal contextualized request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/contextualizedembeddings",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create contextualized request: %w",
			err,
		)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.options.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make contextualized request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read contextualized response body: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"contextualized API request failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var voyageResp contextualizedResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal contextualized response: %w",
			err,
		)
	}

	documentEmbeddings := make([][][]float32, len(voyageResp.Data))
	for docIndex, docData := range voyageResp.Data {
		chunkEmbeddings := make([][]float32, len(docData.Data))
		for chunkIndex, chunkData := range docData.Data {
			chunkEmbeddings[chunkIndex] = chunkData.Embedding
		}
		documentEmbeddings[docIndex] = chunkEmbeddings
	}

	return &embeddings.ContextualizedEmbeddingResponse{
		DocumentEmbeddings: documentEmbeddings,
		Usage: embeddings.EmbeddingUsage{
			TotalTokens: voyageResp.Usage.TotalTokens,
		},
		Model: voyageResp.Model,
	}, nil
}
