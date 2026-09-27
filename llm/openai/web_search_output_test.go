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

// streamWithOutput wraps output items in a server-sent event stream carrying
// a single response.completed event.
func streamWithOutput(items string) string {
	return "event: response.completed\n" +
		`data: {"type":"response.completed","sequence_number":1,"response":` +
		responseWithOutput(items) + "}\n\n"
}

// searchClient returns a Responses client pointed at a test server that
// replies to every request with body, served as contentType.
func searchClient(t *testing.T, contentType, body string) llm.LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	return NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-test"}),
	)
}

// searchReply sends one message to a server replying with items as a
// non-streamed Responses body and returns the response.
func searchReply(t *testing.T, items string) *llm.Response {
	t.Helper()
	client := searchClient(t, "application/json", responseWithOutput(items))
	resp, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	return resp
}

// searchStream streams one message from a server replying with items in a
// response.completed event and returns the completed response.
func searchStream(t *testing.T, items string) *llm.Response {
	t.Helper()
	client := searchClient(t, "text/event-stream", streamWithOutput(items))
	return drainStream(t, client.StreamResponse(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil))
}

// searchCalls returns the openai.web_search_calls entries of resp, or nil
// when there are none, failing the test if the entry has an unexpected type.
func searchCalls(t *testing.T, resp *llm.Response) []map[string]any {
	t.Helper()
	raw, ok := resp.ProviderMetadata["openai.web_search_calls"]
	if !ok {
		return nil
	}
	calls, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("web_search_calls has type %T, want []map[string]any", raw)
	}
	return calls
}

const searchItem = `{"type":"web_search_call","id":"ws_1","status":"completed",` +
	`"action":{"type":"search","queries":["imdb rating for dune"]}}`

// TestASearchCallIsSurfaced confirms a web_search_call output item appears
// in ProviderMetadata with its id and action.
func TestASearchCallIsSurfaced(t *testing.T) {
	resp := searchReply(t, searchItem)
	calls := searchCalls(t, resp)
	if len(calls) != 1 {
		t.Fatalf(
			"%d search calls surfaced, want 1 (metadata: %v)",
			len(calls),
			resp.ProviderMetadata,
		)
	}
	if calls[0]["action"] != "search" {
		t.Errorf("action = %v, want \"search\"", calls[0]["action"])
	}
	if calls[0]["id"] != "ws_1" {
		t.Errorf("id = %v, want ws_1", calls[0]["id"])
	}
}

// TestThreeSearchesAreThreeCalls confirms one entry per item, never a
// deduplicated or collapsed total.
func TestThreeSearchesAreThreeCalls(t *testing.T) {
	items := searchItem + "," +
		`{"type":"web_search_call","id":"ws_2","status":"completed","action":{"type":"search","queries":["b"]}},` +
		`{"type":"web_search_call","id":"ws_3","status":"completed","action":{"type":"search","queries":["c"]}}`
	if got := len(searchCalls(t, searchReply(t, items))); got != 3 {
		t.Errorf("%d search calls surfaced, want 3", got)
	}
}

// TestSearchActionsAreDistinguishable confirms each item's action
// (search/open_page/find_in_page) is preserved rather than summed away.
func TestSearchActionsAreDistinguishable(t *testing.T) {
	items := searchItem + "," +
		`{"type":"web_search_call","id":"ws_2","status":"completed","action":{"type":"open_page","url":"https://example.com"}},` +
		`{"type":"web_search_call","id":"ws_3","status":"completed","action":{"type":"find_in_page","pattern":"rating"}}`
	calls := searchCalls(t, searchReply(t, items))
	if len(calls) != 3 {
		t.Fatalf("%d calls, want 3", len(calls))
	}
	want := []string{"search", "open_page", "find_in_page"}
	for i, w := range want {
		if got := calls[i]["action"]; got != w {
			t.Errorf("call %d action = %v, want %q", i, got, w)
		}
	}
}

// TestNoSearchMeansNoMetadataEntry confirms a turn with no search produces
// no web_search_calls entry at all, rather than an empty one.
func TestNoSearchMeansNoMetadataEntry(t *testing.T) {
	items := `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`
	resp := searchReply(t, items)
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
	resp := searchReply(t, items)
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

// TestADeprecatedSingularQueryIsReported confirms a search action that only
// fills the deprecated singular query field still reports it under queries.
func TestADeprecatedSingularQueryIsReported(t *testing.T) {
	items := `{"type":"web_search_call","id":"ws_1","status":"completed",` +
		`"action":{"type":"search","query":"imdb rating for dune"}}`
	calls := searchCalls(t, searchReply(t, items))
	if len(calls) != 1 {
		t.Fatalf("%d calls, want 1", len(calls))
	}
	queries, _ := calls[0]["queries"].([]string)
	if len(queries) != 1 || queries[0] != "imdb rating for dune" {
		t.Errorf("queries = %v, want [imdb rating for dune]", queries)
	}
}

// TestAStreamedSearchCallIsSurfaced confirms the streaming path reports the
// web_search_call items of the completed response, alongside the same entry
// shape the non-streaming path produces.
func TestAStreamedSearchCallIsSurfaced(t *testing.T) {
	calls := searchCalls(t, searchStream(t, searchItem))
	if len(calls) != 1 {
		t.Fatalf("%d streamed search calls surfaced, want 1", len(calls))
	}
	if calls[0]["id"] != "ws_1" || calls[0]["action"] != "search" {
		t.Errorf("streamed call = %v, want ws_1 with action search", calls[0])
	}
	queries, _ := calls[0]["queries"].([]string)
	if len(queries) != 1 || queries[0] != "imdb rating for dune" {
		t.Errorf("queries = %v, want [imdb rating for dune]", queries)
	}
}
