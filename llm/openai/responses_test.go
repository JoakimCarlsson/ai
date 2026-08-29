package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"

	"github.com/joakimcarlsson/ai/message"
)

const responsesOK = `{"id":"resp_1","object":"response","status":"completed",` +
	`"output":[{"type":"message","role":"assistant",` +
	`"content":[{"type":"output_text","text":"hi"}]}],` +
	`"usage":{"input_tokens":1,"output_tokens":1}}`

// TestResponsesWithHTTPClientTransportUsed confirms a client injected via
// WithResponsesHTTPClient handles outgoing requests: the wrapped transport's
// counter increments, proving the SDK default client was replaced.
func TestResponsesWithHTTPClientTransportUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, responsesOK)
		}))
	defer srv.Close()

	var n int
	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		WithResponsesHTTPClient(&http.Client{
			Transport: countingRT{RoundTripper: http.DefaultTransport, n: &n},
		}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if n == 0 {
		t.Error("injected transport was not used for the request")
	}
}

// captureResponsesBody returns a test server that decodes the request body
// into body and replies with a minimal successful Responses payload. Inline
// rather than shared, so this test file stays independent of the other
// Responses tests.
func captureResponsesBody(body *map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, responsesOK)
		}))
}

// TestWireWebSearchUserLocationCarriesItsType_Responses confirms
// user_location.type is sent as "approximate" whenever a caller configures a
// UserLocation, since the field is required by the API and would otherwise
// be dropped from the JSON entirely.
func TestWireWebSearchUserLocationCarriesItsType_Responses(t *testing.T) {
	var body map[string]any
	srv := captureResponsesBody(&body)
	defer srv.Close()

	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		WithWebSearch(WebSearchOpts{
			UserLocation: &UserLocation{Timezone: "Europe/Bucharest"},
		}),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	tools, ok := body["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools = %v, want the web_search tool", body["tools"])
	}
	web, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tools[0] = %v (%T), want an object", tools[0], tools[0])
	}
	loc, ok := web["user_location"].(map[string]any)
	if !ok {
		t.Fatalf(
			"tools[0].user_location = %v, want an object",
			web["user_location"],
		)
	}
	if got := loc["type"]; got != "approximate" {
		t.Errorf(
			"tools[0].user_location.type = %v, want %q -- the API refuses the whole request without it",
			got,
			"approximate",
		)
	}
	if got := loc["timezone"]; got != "Europe/Bucharest" {
		t.Errorf("tools[0].user_location.timezone = %v, want the caller's", got)
	}
}

// TestWireWebSearchWithNoUserLocationSendsNone_Responses confirms a caller
// who configures no UserLocation gets no user_location field at all, rather
// than an otherwise-empty object carrying only a type.
func TestWireWebSearchWithNoUserLocationSendsNone_Responses(t *testing.T) {
	var body map[string]any
	srv := captureResponsesBody(&body)
	defer srv.Close()

	client := NewResponsesLLM(
		WithResponsesAPIKey("test-key"),
		WithResponsesBaseURL(srv.URL),
		WithResponsesModel(llm.Model{APIModel: "gpt-4o-mini"}),
		WithWebSearch(),
	)

	if _, err := client.SendMessages(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil); err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	tools, _ := body["tools"].([]any)
	if len(tools) == 0 {
		t.Fatalf("tools = %v, want the web_search tool", body["tools"])
	}
	web, _ := tools[0].(map[string]any)
	if _, present := web["user_location"]; present {
		t.Errorf(
			"tools[0].user_location = %v, want it absent when the caller named no location",
			web["user_location"],
		)
	}
}
