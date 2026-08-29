package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/types"
)

// capturingRT redirects every request to the test server, like redirectRT in
// gemini_httpclient_test.go, and decodes the outgoing body so a test can
// assert on what actually crossed the wire rather than on an internal struct.
type capturingRT struct {
	base http.RoundTripper
	host string
	body *map[string]any
}

func (c capturingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, c.body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
	}
	r.URL.Scheme = "http"
	r.URL.Host = c.host
	return c.base.RoundTrip(r)
}

// generateContentStreamOK is generateContentOK in the SSE framing the
// streaming endpoint answers with.
const generateContentStreamOK = "data: " + generateContentOK + "\n\n"

// structuredOutputClient wires a client whose requests are captured into body
// and answered with a canned reply. contentType distinguishes the plain JSON
// of generateContent from the event stream of streamGenerateContent.
func structuredOutputClient(
	t *testing.T,
	body *map[string]any,
	contentType, reply string,
) llm.LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)

	return NewLLM(
		WithAPIKey("test-key"),
		WithModel(llm.Model{APIModel: "gemini-2.0-flash"}),
		WithHTTPClient(&http.Client{
			Transport: capturingRT{
				base: http.DefaultTransport,
				host: srv.Listener.Addr().String(),
				body: body,
			},
		}),
	)
}

func trivialSchema() *schema.StructuredOutputInfo {
	return &schema.StructuredOutputInfo{
		Name:       "answer",
		Parameters: map[string]any{"text": map[string]any{"type": "string"}},
		Required:   []string{"text"},
	}
}

// responseMIMEType digs the generation config's responseMimeType out of a
// captured request body. The genai SDK nests it under "generationConfig" on
// the v1beta generateContent shape.
func responseMIMEType(t *testing.T, body map[string]any) any {
	t.Helper()
	cfg, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("request carried no generationConfig: %#v", body)
	}
	return cfg["responseMimeType"]
}

// TestStructuredOutputRequestsJSON confirms responseMimeType is set to
// application/json alongside a response schema.
func TestStructuredOutputRequestsJSON(t *testing.T) {
	var body map[string]any
	client := structuredOutputClient(
		t,
		&body,
		"application/json",
		generateContentOK,
	)

	if _, err := client.SendMessagesWithStructuredOutput(
		context.Background(),
		[]message.Message{
			message.NewUserMessage("hi"),
		},
		nil,
		trivialSchema(),
	); err != nil {
		t.Fatalf("SendMessagesWithStructuredOutput: %v", err)
	}

	if got := responseMIMEType(t, body); got != "application/json" {
		t.Errorf(
			"generationConfig.responseMimeType = %v, want %q -- the schema alone does not request JSON",
			got,
			"application/json",
		)
	}
}

// TestStreamedStructuredOutputRequestsJSON is the same assertion on the
// streaming path, which builds its config independently.
func TestStreamedStructuredOutputRequestsJSON(t *testing.T) {
	var body map[string]any
	client := structuredOutputClient(
		t,
		&body,
		"text/event-stream",
		generateContentStreamOK,
	)

	for ev := range client.StreamResponseWithStructuredOutput(context.Background(),
		[]message.Message{message.NewUserMessage("hi")}, nil, trivialSchema()) {
		if ev.Type == types.EventError {
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	if got := responseMIMEType(t, body); got != "application/json" {
		t.Errorf("streamed generationConfig.responseMimeType = %v, want %q",
			got, "application/json")
	}
}
