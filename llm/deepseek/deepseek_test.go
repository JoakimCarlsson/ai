package deepseek_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm/deepseek"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

// TestCacheHitTokens verifies a DeepSeek response with a cache hit populates
// CacheReadTokens (read from the top-level prompt_cache_hit_tokens usage field)
// and ReasoningTokens (from completion_tokens_details).
func TestCacheHitTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"x","object":"chat.completion",`+
				`"choices":[{"index":0,"message":{"role":"assistant",`+
				`"content":"hi"},"finish_reason":"stop"}],`+
				`"usage":{"prompt_tokens":100,"completion_tokens":40,`+
				`"total_tokens":140,"prompt_cache_hit_tokens":80,`+
				`"prompt_cache_miss_tokens":20,`+
				`"completion_tokens_details":{"reasoning_tokens":15}}}`)
		}))
	defer srv.Close()

	client := deepseek.NewLLM(
		llmopenai.WithAPIKey("test-key"),
		llmopenai.WithBaseURL(srv.URL),
		llmopenai.WithModel(model.Model{APIModel: "deepseek-chat"}),
	)

	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if resp.Usage.CacheReadTokens != 80 {
		t.Errorf("CacheReadTokens = %d, want 80", resp.Usage.CacheReadTokens)
	}
	if resp.Usage.InputTokens != 20 {
		t.Errorf("InputTokens = %d, want 20", resp.Usage.InputTokens)
	}
	if resp.Usage.ReasoningTokens != 15 {
		t.Errorf("ReasoningTokens = %d, want 15", resp.Usage.ReasoningTokens)
	}
}
