package groq

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	openaisdk "github.com/openai/openai-go/v3"
)

const compoundCompletionOK = `{"id":"x","object":"chat.completion",` +
	`"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},` +
	`"finish_reason":"stop"}],` +
	`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

// countingRT is an http.RoundTripper that increments a counter on every request
// before delegating to the wrapped transport, used to prove an injected client's
// transport actually handled the outgoing request.
type countingRT struct {
	http.RoundTripper
	n *int
}

func (c countingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	*c.n++
	return c.RoundTripper.RoundTrip(r)
}

// TestCompoundWithHTTPClientTransportUsed confirms a client injected via
// WithCompoundHTTPClient handles outgoing requests: the wrapped transport's
// counter increments, proving the SDK default client was replaced.
func TestCompoundWithHTTPClientTransportUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, compoundCompletionOK)
		}))
	defer srv.Close()

	var n int
	client := NewCompoundLLM(
		WithCompoundAPIKey("test-key"),
		WithCompoundBaseURL(srv.URL),
		WithCompoundModel(model.Model{APIModel: "groq/compound"}),
		WithCompoundHTTPClient(&http.Client{
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

// TestCompoundPreparedParamsStopSequencesArray verifies that the Groq compound
// client sends every provided stop sequence as an array, not just the first.
func TestCompoundPreparedParamsStopSequencesArray(t *testing.T) {
	c := &compoundClient{options: CompoundOptions{
		stopSequences: []string{"END", "STOP", "HALT"},
	}}

	params := c.preparedParams(
		[]openaisdk.ChatCompletionMessageParamUnion{},
		nil,
	)

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

// TestCompoundPreparedParamsStopSequencesCappedAtFour verifies the Groq stop
// limit of 4 is enforced, matching OpenAI.
func TestCompoundPreparedParamsStopSequencesCappedAtFour(t *testing.T) {
	c := &compoundClient{options: CompoundOptions{
		stopSequences: []string{"1", "2", "3", "4", "5", "6"},
	}}

	params := c.preparedParams(
		[]openaisdk.ChatCompletionMessageParamUnion{},
		nil,
	)

	if len(params.Stop.OfStringArray) != 4 {
		t.Fatalf(
			"expected stop sequences capped at 4, got %d: %v",
			len(params.Stop.OfStringArray),
			params.Stop.OfStringArray,
		)
	}
}
