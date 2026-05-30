package openrouter_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/llm/openrouter"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

// TestWireRoutingOptions confirms the OpenRouter-specific options reach the
// request body: the provider routing object, the models fallback array, and
// top_k.
func TestWireRoutingOptions(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"x","object":"chat.completion",`+
				`"choices":[{"index":0,"message":{"role":"assistant",`+
				`"content":"hi"},"finish_reason":"stop"}],`+
				`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}))
	defer srv.Close()

	client := openrouter.NewLLM(
		llmopenai.WithAPIKey("test-key"),
		llmopenai.WithBaseURL(srv.URL),
		llmopenai.WithModel(model.Model{APIModel: "openai/gpt-4o"}),
		openrouter.WithProviderRouting([]string{"openai", "azure"}, false),
		openrouter.WithModelFallbacks("anthropic/claude", "google/gemini"),
		openrouter.WithTopK(50),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	provider, ok := body["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider object, got %v (%T)",
			body["provider"], body["provider"])
	}
	if provider["allow_fallbacks"] != false {
		t.Errorf("provider.allow_fallbacks = %v, want false",
			provider["allow_fallbacks"])
	}
	order, ok := provider["order"].([]any)
	if !ok || len(order) != 2 || order[0] != "openai" || order[1] != "azure" {
		t.Errorf("provider.order = %v, want [openai azure]", provider["order"])
	}

	models, ok := body["models"].([]any)
	if !ok || len(models) != 2 || models[0] != "anthropic/claude" {
		t.Errorf("models = %v, want [anthropic/claude google/gemini]",
			body["models"])
	}

	if got, ok := body["top_k"].(float64); !ok || int64(got) != 50 {
		t.Errorf("top_k = %v (%T), want 50", body["top_k"], body["top_k"])
	}
}
