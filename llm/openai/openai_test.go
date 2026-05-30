package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

// TestPreparedParamsStopSequencesArray verifies that all provided stop
// sequences are sent as an array (OfStringArray), not just the first one.
func TestPreparedParamsStopSequencesArray(t *testing.T) {
	c := &Client{options: Options{
		stopSequences: []string{"END", "STOP", "HALT"},
	}}

	params := c.preparedParams(nil, nil)

	if params.Stop.OfString.Valid() {
		t.Fatalf(
			"expected OfString to be unset, got %q",
			params.Stop.OfString.Value,
		)
	}
	got := params.Stop.OfStringArray
	want := []string{"END", "STOP", "HALT"}
	if len(got) != len(want) {
		t.Fatalf(
			"expected %d stop sequences, got %d: %v",
			len(want),
			len(got),
			got,
		)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stop[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestPreparedParamsStopSequencesCappedAtFour verifies that the OpenAI-native
// limit of 4 stop sequences is enforced.
func TestPreparedParamsStopSequencesCappedAtFour(t *testing.T) {
	c := &Client{options: Options{
		stopSequences: []string{"1", "2", "3", "4", "5", "6"},
	}}

	params := c.preparedParams(nil, nil)

	if len(params.Stop.OfStringArray) != 4 {
		t.Fatalf("expected stop sequences capped at 4, got %d: %v",
			len(params.Stop.OfStringArray), params.Stop.OfStringArray)
	}
}

// TestRequestOptionsTopK verifies that top_k yields a request option only on
// the compatible-provider path: it requires both WithTopK and a custom base
// URL, since OpenAI/Azure proper reject top_k.
func TestRequestOptionsTopK(t *testing.T) {
	k := int64(40)

	none := (&Client{options: Options{baseURL: "https://example.test"}}).requestOptions()
	if len(none) != 0 {
		t.Errorf(
			"expected no request options when top_k unset, got %d",
			len(none),
		)
	}

	native := (&Client{options: Options{topK: &k}}).requestOptions()
	if len(native) != 0 {
		t.Errorf("expected no top_k injection without a custom base URL "+
			"(OpenAI/Azure reject it), got %d", len(native))
	}

	compat := (&Client{options: Options{topK: &k, baseURL: "https://example.test"}}).requestOptions()
	if len(compat) != 1 {
		t.Fatalf(
			"expected one request option on the compatible path, got %d",
			len(compat),
		)
	}
}

// TestWireTopKAndStop confirms that top_k reaches the request body and that all
// stop sequences are serialized as a JSON array on the wire.
func TestWireTopKAndStop(t *testing.T) {
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

	llm := NewLLM(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithModel(model.Model{APIModel: "gpt-4o-mini"}),
		WithTopK(40),
		WithStopSequences("END", "STOP", "HALT"),
	)

	_, err := llm.SendMessages(
		context.Background(),
		[]message.Message{message.NewUserMessage("hello")},
		nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if got, ok := body["top_k"].(float64); !ok || int64(got) != 40 {
		t.Errorf("expected top_k=40 in request body, got %v (%T)",
			body["top_k"], body["top_k"])
	}

	stop, ok := body["stop"].([]any)
	if !ok {
		t.Fatalf("expected stop to be a JSON array, got %v (%T)",
			body["stop"], body["stop"])
	}
	want := []string{"END", "STOP", "HALT"}
	if len(stop) != len(want) {
		t.Fatalf("expected %d stop sequences on the wire, got %d: %v",
			len(want), len(stop), stop)
	}
	for i, v := range want {
		if stop[i] != v {
			t.Errorf("wire stop[%d] = %v, want %q", i, stop[i], v)
		}
	}
}
