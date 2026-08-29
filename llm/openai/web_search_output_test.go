package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
)

// responseWithOutput wraps output items in a completed Responses body.
func responseWithOutput(items string) string {
	return `{"id":"resp_1","object":"response","status":"completed",` +
		`"output":[` + items + `],"usage":{"input_tokens":1,"output_tokens":1}}`
}

func searchClient(t *testing.T, body string) llm.LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	return NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-test"}),
	)
}

func searchCalls(t *testing.T, resp *llm.Response) []any {
	t.Helper()
	raw, ok := resp.ProviderMetadata["openai.web_search_calls"]
	if !ok {
		return nil
	}
	calls, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("web_search_calls has type %T, want []map[string]any", raw)
	}
	out := make([]any, len(calls))
	for i, c := range calls {
		out[i] = c
	}
	return out
}

const searchItem = `{"type":"web_search_call","id":"ws_1","status":"completed",` +
	`"action":{"type":"search","queries":["imdb rating for dune"]}}`

// TestASearchCallIsSurfaced confirms a web_search_call output item appears
// in ProviderMetadata with its id and action.
func TestASearchCallIsSurfaced(t *testing.T) {
	client := searchClient(t, responseWithOutput(searchItem))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	calls := searchCalls(t, resp)
	if len(calls) != 1 {
		t.Fatalf(
			"%d search calls surfaced, want 1 (metadata: %v)",
			len(calls),
			resp.ProviderMetadata,
		)
	}
	c := calls[0].(map[string]any)
	if c["action"] != "search" {
		t.Errorf("action = %v, want \"search\"", c["action"])
	}
	if c["id"] != "ws_1" {
		t.Errorf("id = %v, want ws_1", c["id"])
	}
}

// TestThreeSearchesAreThreeCalls confirms one entry per item, never a
// deduplicated or collapsed total.
func TestThreeSearchesAreThreeCalls(t *testing.T) {
	items := searchItem + "," +
		`{"type":"web_search_call","id":"ws_2","status":"completed","action":{"type":"search","queries":["b"]}},` +
		`{"type":"web_search_call","id":"ws_3","status":"completed","action":{"type":"search","queries":["c"]}}`
	client := searchClient(t, responseWithOutput(items))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	if got := len(searchCalls(t, resp)); got != 3 {
		t.Errorf("%d search calls surfaced, want 3", got)
	}
}

// TestSearchActionsAreDistinguishable confirms each item's action
// (search/open_page/find_in_page) is preserved rather than summed away.
func TestSearchActionsAreDistinguishable(t *testing.T) {
	items := searchItem + "," +
		`{"type":"web_search_call","id":"ws_2","status":"completed","action":{"type":"open_page","url":"https://example.com"}},` +
		`{"type":"web_search_call","id":"ws_3","status":"completed","action":{"type":"find_in_page","pattern":"rating"}}`
	client := searchClient(t, responseWithOutput(items))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	calls := searchCalls(t, resp)
	if len(calls) != 3 {
		t.Fatalf("%d calls, want 3", len(calls))
	}
	want := []string{"search", "open_page", "find_in_page"}
	for i, w := range want {
		if got := calls[i].(map[string]any)["action"]; got != w {
			t.Errorf("call %d action = %v, want %q", i, got, w)
		}
	}
}

// TestNoSearchMeansNoMetadataEntry confirms a turn with no search produces
// no web_search_calls entry at all, rather than an empty one.
func TestNoSearchMeansNoMetadataEntry(t *testing.T) {
	items := `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`
	client := searchClient(t, responseWithOutput(items))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	if _, ok := resp.ProviderMetadata["openai.web_search_calls"]; ok {
		t.Error("a turn with no search produced a web_search_calls entry")
	}
}

// TestCitationsSurviveAlongsideSearches confirms citations and search calls
// coexist in the same metadata map.
func TestCitationsSurviveAlongsideSearches(t *testing.T) {
	items := searchItem + `,{"type":"message","role":"assistant","content":[{"type":"output_text",` +
		`"text":"Dune scored 8.0.","annotations":[{"type":"url_citation",` +
		`"url":"https://imdb.com/x","title":"Dune","start_index":0,"end_index":5}]}]}`
	client := searchClient(t, responseWithOutput(items))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	if got := len(searchCalls(t, resp)); got != 1 {
		t.Errorf("%d search calls, want 1", got)
	}
	if _, ok := resp.ProviderMetadata["openai.url_citations"]; !ok {
		t.Error(
			"citations were lost when a search call shared the metadata map",
		)
	}
	if resp.Content != "Dune scored 8.0." {
		t.Errorf("content = %q", resp.Content)
	}
}
