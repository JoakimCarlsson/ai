// Package gemini provides a Google Gemini implementation of the [embeddings.Embedding] interface.
package gemini

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
	"google.golang.org/genai"
)

// Options configures the Gemini embeddings client.
type Options struct {
	apiKey     string
	model      model.EmbeddingModel
	batchSize  int
	dimensions *int
	taskType   string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key used to authenticate with Gemini.
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

// WithBatchSize sets the number of texts to process in each batch request.
func WithBatchSize(
	batchSize int,
) Option {
	return func(o *Options) { o.batchSize = batchSize }
}

// WithDimensions specifies the output dimensionality for embedding vectors.
func WithDimensions(
	dimensions int,
) Option {
	return func(o *Options) { o.dimensions = &dimensions }
}

// WithTaskType sets the task type for embeddings (e.g., "RETRIEVAL_DOCUMENT", "RETRIEVAL_QUERY").
func WithTaskType(
	taskType string,
) Option {
	return func(o *Options) { o.taskType = taskType }
}

// Client implements [embeddings.Embedding] against the Google Gemini API.
type Client struct {
	options Options
	client  *genai.Client
}

// NewEmbedding constructs a Gemini embeddings client.
func NewEmbedding(opts ...Option) embeddings.Embedding {
	options := Options{batchSize: 100}
	for _, o := range opts {
		o(&options)
	}

	client, _ := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  options.apiKey,
			Backend: genai.BackendGeminiAPI,
		},
	)

	return embeddings.WithTracing(&Client{
		options: options,
		client:  client,
	}, embeddings.TracingAttrs{
		Dimensions: options.dimensions,
	})
}

// Model returns the configured embedding model.
func (c *Client) Model() model.EmbeddingModel { return c.options.model }

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

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := c.embedBatch(ctx, texts[i:end], inputType...)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(allEmbeddings, resp.Embeddings...)
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: allEmbeddings,
		Usage:      embeddings.EmbeddingUsage{},
		Model:      c.options.model.APIModel,
	}, nil
}

func (c *Client) embedBatch(
	ctx context.Context,
	texts []string,
	inputType ...string,
) (*embeddings.EmbeddingResponse, error) {
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = genai.NewContentFromText(text, "user")
	}

	config := &genai.EmbedContentConfig{}
	taskType := c.options.taskType
	if len(inputType) > 0 && inputType[0] != "" {
		taskType = inputType[0]
	}
	if taskType != "" {
		config.TaskType = taskType
	}
	if c.options.dimensions != nil {
		dim := int32(*c.options.dimensions)
		config.OutputDimensionality = &dim
	}

	modelName := c.options.model.APIModel
	resp, err := c.client.Models.EmbedContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	out := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		out[i] = emb.Values
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: out,
		Usage:      embeddings.EmbeddingUsage{},
		Model:      modelName,
	}, nil
}
func taskPrefixForEmbedding2(taskType string) string {
	switch strings.ToUpper(taskType) {
	case "RETRIEVAL_QUERY":
		return "task: search result | query: "
	case "QUESTION_ANSWERING":
		return "task: question answering | query: "
	case "FACT_VERIFICATION":
		return "task: fact checking | query: "
	case "CODE_RETRIEVAL_QUERY":
		return "task: code retrieval | query: "
	case "CLASSIFICATION":
		return "task: classification | query: "
	case "CLUSTERING":
		return "task: clustering | query: "
	case "SEMANTIC_SIMILARITY":
		return "task: sentence similarity | query: "
	// RETRIEVAL_DOCUMENT uses a different shape ("title: … | text: …") and is
	// applied at indexing time, not at query time. The caller should format
	// document text themselves, or pass an empty taskType for documents.
	default:
		return ""
	}
}

func parseDataURI(raw string) ([]byte, string, error) {
	mimeType := "" // caller must use mc.MimeType when no data URI prefix is present

	if strings.HasPrefix(raw, "data:") {
		rest := raw[len("data:"):]

		mType, remainder, found := strings.Cut(rest, ";")
		if !found {
			return nil, "", fmt.Errorf("malformed data URI: missing semicolon")
		}
		mimeType = mType

		_, after, found := strings.Cut(remainder, ",")
		if !found {
			return nil, "", fmt.Errorf("malformed data URI: missing comma after encoding")
		}
		raw = after
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(raw)
		if err != nil {
			return nil, "", fmt.Errorf("base64 decode failed: %w", err)
		}
	}

	return decoded, mimeType, nil
}

// GenerateMultimodalEmbeddings create mulit model embeddings
func (c *Client) GenerateMultimodalEmbeddings(
	ctx context.Context,
	input []embeddings.MultimodalInput,
	taskType ...string,
) (*embeddings.EmbeddingResponse, error) {
	switch c.options.model.ID {
	case model.GeminiEmbedding2:
		if len(input) == 0 {
			return &embeddings.EmbeddingResponse{
				Embeddings: [][]float32{},
				Usage:      embeddings.EmbeddingUsage{},
				Model:      c.options.model.APIModel,
			}, nil
		}

		resolvedTaskType := c.options.taskType
		if len(taskType) > 0 && taskType[0] != "" {
			resolvedTaskType = taskType[0]
		}
		textPrefix := taskPrefixForEmbedding2(resolvedTaskType)

		// Each multimodalInput becomes its own genai.Content, producing one
		// embedding per input.
		contents := make([]*genai.Content, 0, len(input))

		for _, mi := range input {
			parts := make([]*genai.Part, 0, len(mi.Content))

			for _, mc := range mi.Content {
				// priority 1: raw bytes are present — this covers all binary modalities
				// (image/png, image/jpeg, audio/mpeg, video/mp4, application/pdf, …).
				// MimeType is the only thing that distinguishes them, exactly as in
				// the official Python: types.Part.from_bytes(data=..., mime_type=...).
				if len(mc.ContentData) > 0 {
					if mc.MimeType == "" {
						return nil, fmt.Errorf(
							"gemini multimodal embeddings: MimeType required when ContentData is set",
						)
					}
					parts = append(parts, &genai.Part{
						InlineData: &genai.Blob{
							MIMEType: mc.MimeType,
							Data:     mc.ContentData,
						},
					})
					continue
				}

				// Priority 2: type-based dispatch for the remaining kinds.
				switch mc.Type {
				case "text":
					text := mc.Text
					if textPrefix != "" {
						text = textPrefix + text
					}
					parts = append(parts, genai.NewPartFromText(text))

				case "image_base64":
					// Legacy path: base64 string or data URI. Prefer ContentData above.
					if mc.ImageBase64 == "" {
						return nil, fmt.Errorf(
							"gemini multimodal embeddings: image_base64 part has no ContentData or ImageBase64",
						)
					}
					data, parsedMime, err := parseDataURI(mc.ImageBase64)
					if err != nil {
						return nil, fmt.Errorf("gemini multimodal embeddings: decode image_base64: %w", err)
					}
					mimeType := mc.MimeType
					if mimeType == "" {
						mimeType = parsedMime
					}
					if mimeType == "" {
						return nil, fmt.Errorf(
							"gemini multimodal embeddings: MimeType is required for image_base64 content",
						)
					}
					parts = append(parts, &genai.Part{
						InlineData: &genai.Blob{MIMEType: mimeType, Data: data},
					})

				case "image_url":
					// For gs:// or Files API URIs only.
					// Plain HTTPS URLs are not fetched by the API — callers must
					// pre-fetch and supply bytes via ContentData + MimeType.
					if mc.ImageURL == "" {
						return nil, fmt.Errorf(
							"gemini multimodal embeddings: image_url part has empty ImageURL",
						)
					}
					parts = append(parts, &genai.Part{
						FileData: &genai.FileData{FileURI: mc.ImageURL},
					})

				default:
					return nil, fmt.Errorf(
						"gemini multimodal embeddings: unsupported content type %q (valid: text, image_base64, image_url; or set ContentData+MimeType for any binary modality)",
						mc.Type,
					)
				}
			}

			contents = append(contents, &genai.Content{Parts: parts})
		}

		var config *genai.EmbedContentConfig
		if c.options.dimensions != nil {
			dim := int32(*c.options.dimensions)
			config = &genai.EmbedContentConfig{OutputDimensionality: &dim}
		}

		result, err := c.client.Models.EmbedContent(ctx, c.options.model.APIModel, contents, config)
		if err != nil {
			return nil, fmt.Errorf("gemini multimodal embeddings: %w", err)
		}

		embeds := make([][]float32, len(result.Embeddings))
		for i, e := range result.Embeddings {
			embeds[i] = e.Values
		}

		return &embeddings.EmbeddingResponse{
			Embeddings: embeds,
			Model:      c.options.model.APIModel,
		}, nil

	default:
		return nil, fmt.Errorf(
			"%s (%s) does not support multimodal embeddings; use gemini-embedding-2",
			c.options.model.Name, c.options.model.ID,
		)
	}
}

// GenerateContextualizedEmbeddings is not supported by Gemini.
func (c *Client) GenerateContextualizedEmbeddings(
	_ context.Context,
	_ [][]string,
	_ ...string,
) (*embeddings.ContextualizedEmbeddingResponse, error) {
	return nil, fmt.Errorf("gemini does not support contextualized embeddings")
}
