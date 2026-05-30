package perplexity_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/llm/perplexity"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

// TestWireSearchControlsAndMetadata confirms search-control options reach the
// request body and that citations/search_results surface into ProviderMetadata.
func TestWireSearchControlsAndMetadata(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"x","object":"chat.completion",`+
				`"choices":[{"index":0,"message":{"role":"assistant",`+
				`"content":"hi"},"finish_reason":"stop"}],`+
				`"citations":["https://a.test"],`+
				`"search_results":[{"title":"A","url":"https://a.test"}],`+
				`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}))
	defer srv.Close()

	client := perplexity.NewLLM(
		llmopenai.WithAPIKey("test-key"),
		llmopenai.WithBaseURL(srv.URL),
		llmopenai.WithModel(model.Model{APIModel: "sonar"}),
		perplexity.WithSearchDomainFilter("a.test", "-b.test"),
		perplexity.WithSearchRecencyFilter("week"),
		perplexity.WithReturnRelatedQuestions(true),
	)

	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	domains, ok := body["search_domain_filter"].([]any)
	if !ok || len(domains) != 2 || domains[0] != "a.test" {
		t.Errorf("search_domain_filter = %v, want [a.test -b.test]",
			body["search_domain_filter"])
	}
	if body["search_recency_filter"] != "week" {
		t.Errorf("search_recency_filter = %v, want week",
			body["search_recency_filter"])
	}
	if body["return_related_questions"] != true {
		t.Errorf("return_related_questions = %v, want true",
			body["return_related_questions"])
	}

	cites, ok := resp.ProviderMetadata[perplexity.MetadataKeyCitations].([]any)
	if !ok || len(cites) != 1 || cites[0] != "https://a.test" {
		t.Errorf("citations metadata = %v", resp.ProviderMetadata)
	}
	results, ok := resp.ProviderMetadata[perplexity.MetadataKeySearchResults].([]any)
	if !ok || len(results) != 1 {
		t.Errorf("search_results metadata = %v", resp.ProviderMetadata)
	}
}
