package gemini

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/model"
)

func TestParseDataURI(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantData   []byte
		wantMIME   string
		wantErr    bool
		errMessage string
	}{
		{
			name:     "valid data URI with padding",
			input:    "data:image/png;base64,aGVsbG8=",
			wantData: []byte("hello"),
			wantMIME: "image/png",
			wantErr:  false,
		},
		{
			name:     "valid data URI without padding",
			input:    "data:image/png;base64,aGVsbG8",
			wantData: []byte("hello"),
			wantMIME: "image/png",
			wantErr:  false,
		},
		{
			name:     "raw base64 with padding, no data URI prefix",
			input:    "aGVsbG8=",
			wantData: []byte("hello"),
			wantMIME: "",
			wantErr:  false,
		},
		{
			name:     "raw base64 without padding, no data URI prefix",
			input:    "aGVsbG8",
			wantData: []byte("hello"),
			wantMIME: "",
			wantErr:  false,
		},
		{
			name:     "data URI with application/pdf mime",
			input:    "data:application/pdf;base64,dGVzdA==",
			wantData: []byte("test"),
			wantMIME: "application/pdf",
			wantErr:  false,
		},
		{
			name:     "data URI with image/jpeg mime",
			input:    "data:image/jpeg;base64,dGVzdA==",
			wantData: []byte("test"),
			wantMIME: "image/jpeg",
			wantErr:  false,
		},
		{
			name:     "empty data URI",
			input:    "data:image/png;base64,",
			wantData: []byte{},
			wantMIME: "image/png",
			wantErr:  false,
		},
		{
			name:     "empty raw base64",
			input:    "",
			wantData: []byte{},
			wantMIME: "",
			wantErr:  false,
		},
		{
			name:  "binary data",
			input: "data:application/octet-stream;base64,AAECAwQFBgcICQoLDA0ODw==",
			wantData: []byte{
				0,
				1,
				2,
				3,
				4,
				5,
				6,
				7,
				8,
				9,
				10,
				11,
				12,
				13,
				14,
				15,
			},
			wantMIME: "application/octet-stream",
			wantErr:  false,
		},
		{
			name:       "missing semicolon",
			input:      "data:image/png_base64,data",
			wantErr:    true,
			errMessage: "malformed data URI: missing semicolon",
		},
		{
			name:       "missing comma after encoding",
			input:      "data:image/png;base64",
			wantErr:    true,
			errMessage: "malformed data URI: missing comma after encoding",
		},
		{
			name:       "unsupported encoding",
			input:      "data:image/png;hex,deadbeef",
			wantErr:    true,
			errMessage: `unsupported data URI encoding "hex" (only base64 supported)`,
		},
		{
			name:       "invalid base64 data",
			input:      "data:image/png;base64,!!!invalid!!!",
			wantErr:    true,
			errMessage: "base64 decode failed:",
		},
		{
			name:       "invalid raw base64",
			input:      "!!!",
			wantErr:    true,
			errMessage: "base64 decode failed:",
		},
		{
			name:     "data URI with text/plain mime type",
			input:    "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==",
			wantData: []byte("Hello, World!"),
			wantMIME: "text/plain",
			wantErr:  false,
		},
		{
			name:     "data URI with empty mime type",
			input:    "data:;base64,dGVzdA==",
			wantData: []byte("test"),
			wantMIME: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, mime, err := parseDataURI(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errMessage != "" &&
					!strings.Contains(err.Error(), tt.errMessage) {
					t.Fatalf(
						"expected error %q, got %q",
						tt.errMessage,
						err.Error(),
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != string(tt.wantData) {
				t.Fatalf("data mismatch: got %v, want %v", data, tt.wantData)
			}
			if mime != tt.wantMIME {
				t.Fatalf("mime mismatch: got %q, want %q", mime, tt.wantMIME)
			}
		})
	}
}

func TestGenerateMultimodalEmbeddings_WrongModel(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.EmbeddingModel{
				ID:       "text-embedding-004",
				Name:     "Gemini Text Embedding 004",
				Provider: model.ProviderGemini,
				APIModel: "text-embedding-004",
			},
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for wrong model")
	}
	if !strings.Contains(
		err.Error(),
		"does not support multimodal embeddings",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_MissingMimeType(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{ContentData: []byte{0x89, 0x50, 0x4e, 0x47}},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for missing MimeType")
	}
	if !strings.Contains(
		err.Error(),
		"MimeType required when ContentData is set",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_EmptyImageBase64(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{Type: "image_base64"},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for empty ImageBase64")
	}
	if !strings.Contains(
		err.Error(),
		"image_base64 part has no ContentData or ImageBase64",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_ImageBase64NoMime(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{Type: "image_base64", ImageBase64: "aGVsbG8="},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for missing MimeType on image_base64")
	}
	if !strings.Contains(
		err.Error(),
		"MimeType is required for image_base64 content",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_InvalidImageBase64(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{
					Type:        "image_base64",
					ImageBase64: "!!!invalid!!!",
					MimeType:    "image/png",
				},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "decode image_base64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_EmptyImageURL(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{Type: "image_url"},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for empty ImageURL")
	}
	if !strings.Contains(err.Error(), "image_url part has empty ImageURL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_InvalidImageURL(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{Type: "image_url", ImageURL: "https://example.com/image.jpg"},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for unsupported image URL")
	}
	if !strings.Contains(err.Error(), "is not a supported URI") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_UnsupportedContentType(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	_, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{Content: []embeddings.MultimodalContent{
				{Type: "video"},
			}},
		},
	)
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateMultimodalEmbeddings_EmptyInput(t *testing.T) {
	c := &Client{
		options: Options{
			model: model.GeminiEmbeddingModels[model.GeminiEmbedding2],
		},
	}
	resp, err := c.GenerateMultimodalEmbeddings(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Embeddings) != 0 {
		t.Fatalf("expected 0 embeddings, got %d", len(resp.Embeddings))
	}
}

func TestGenerateMultimodalEmbeddings_Integration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	imgPath := "../../examples/embeddings/gemini/black-dog.jpg"
	imgBytes, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("reading black-dog.jpg: %v", err)
	}

	c := NewEmbedding(
		WithAPIKey(apiKey),
		WithModel(model.GeminiEmbeddingModels[model.GeminiEmbedding2]),
		WithDimensions(768),
	)

	resp, err := c.GenerateMultimodalEmbeddings(
		context.Background(),
		[]embeddings.MultimodalInput{
			{
				Content: []embeddings.MultimodalContent{
					{ContentData: imgBytes, MimeType: "image/jpeg"},
					{Type: "text", Text: "a cute black dog"},
				},
			},
		},
		"RETRIEVAL_DOCUMENT",
	)
	if err != nil {
		t.Fatalf("generating multimodal embedding: %v", err)
	}
	if len(resp.Embeddings) == 0 {
		t.Fatal("expected at least one embedding")
	}
	if len(resp.Embeddings[0]) == 0 {
		t.Fatal("expected non-zero dimension embedding")
	}
}
