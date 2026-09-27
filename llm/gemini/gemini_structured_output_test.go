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

// capturingRT wraps redirectRT, decoding each outgoing body into body first so
// a test can assert on what actually crossed the wire rather than on an
// internal struct.
type capturingRT struct {
	redirect redirectRT
	body     *map[string]any
}

// newCapturingRT returns a capturingRT that sends every request to host and
// decodes its body into body.
func newCapturingRT(host string, body *map[string]any) capturingRT {
	return capturingRT{
		redirect: redirectRT{
			base: http.DefaultTransport,
			host: host,
			n:    new(int),
		},
		body: body,
	}
}

// RoundTrip decodes the request body into c.body, restores it, and hands the
// request to the wrapped redirectRT.
func (c capturingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, c.body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
	}
	return c.redirect.RoundTrip(r)
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
			Transport: newCapturingRT(srv.Listener.Addr().String(), body),
		}),
	)
}

// trivialSchema returns a one-field structured-output schema, enough to put a
// request on the structured-output path.
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
