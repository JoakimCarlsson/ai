package azure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/message"
)

const completionOK = `{"id":"x","object":"chat.completion",` +
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

// TestWithHTTPClientTransportUsed confirms a client injected via WithHTTPClient
// handles outgoing requests on the Azure-native SDK path (endpoint + apiVersion
// set): the wrapped transport's counter increments, proving the SDK default
// client was replaced.
func TestWithHTTPClientTransportUsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, completionOK)
		}))
	defer srv.Close()

	var n int
	client := NewLLM(
		WithAPIKey("test-key"),
		WithEndpoint(srv.URL),
		WithAPIVersion("2024-02-01"),
		WithDeployment("gpt-4o-mini"),
		WithHTTPClient(&http.Client{
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

// TestWithAttachments covers the three states of the attachment-capability
// override: unset leaves whatever resolveDeployment produced, an explicit true
// declares support for a deployment name the registry cannot recognise, and an
// explicit false overrides a table match that reports true.
func TestWithAttachments(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want bool
	}{
		{
			name: "unset on unknown deployment stays false",
			opts: []Option{WithDeployment("unknown-deployment")},
			want: false,
		},
		{
			name: "explicit true on unknown deployment",
			opts: []Option{
				WithDeployment("unknown-deployment"),
				WithAttachments(true),
			},
			want: true,
		},
		{
			name: "explicit false overrides a table match",
			opts: []Option{
				WithDeployment("gpt-4o"),
				WithAttachments(false),
			},
			want: false,
		},
		{
			name: "unset on a table match keeps registry value",
			opts: []Option{WithDeployment("gpt-4o")},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLLM(tt.opts...).Model().SupportsAttachments
			if got != tt.want {
				t.Errorf("SupportsAttachments = %v, want %v", got, tt.want)
			}
		})
	}
}
