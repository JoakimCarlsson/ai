package openrouter_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/model"
	ttsopenai "github.com/joakimcarlsson/ai/tts/openai"
	"github.com/joakimcarlsson/ai/tts/openrouter"
)

// newSpeechServer records the decoded request body and path, and replies with
// raw audio bytes the way OpenRouter's /audio/speech does.
func newSpeechServer(
	t *testing.T,
	body *map[string]any,
	path *string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
			}
			if body != nil {
				if err := json.Unmarshal(raw, body); err != nil {
					t.Errorf("decode request body: %v", err)
				}
			}
			if path != nil {
				*path = r.URL.Path
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("ID3fake-mp3-bytes"))
		}))
}

// TestGenerateAudio confirms the wrapper speaks the OpenAI Audio Speech shape
// and hands the raw bytes back with the response content type.
func TestGenerateAudio(t *testing.T) {
	var body map[string]any
	var path string
	srv := newSpeechServer(t, &body, &path)
	defer srv.Close()

	client := openrouter.NewGeneration(
		ttsopenai.WithAPIKey("test-key"),
		ttsopenai.WithBaseURL(srv.URL),
		ttsopenai.WithModel(
			model.OpenRouterAudioModels[model.OpenRouterMAIVoice2],
		),
		ttsopenai.WithVoice("en-US-Harper:MAI-Voice-2"),
		ttsopenai.WithOutputFormat("mp3"),
	)

	resp, err := client.GenerateAudio(context.Background(), "hello there")
	if err != nil {
		t.Fatalf("GenerateAudio: %v", err)
	}

	if path != "/audio/speech" {
		t.Errorf("request path = %q, want /audio/speech", path)
	}
	if body["model"] != "microsoft/mai-voice-2" {
		t.Errorf("model = %v, want microsoft/mai-voice-2", body["model"])
	}
	if body["input"] != "hello there" {
		t.Errorf("input = %v, want hello there", body["input"])
	}
	if body["voice"] != "en-US-Harper:MAI-Voice-2" {
		t.Errorf("voice = %v, want en-US-Harper:MAI-Voice-2", body["voice"])
	}
	if body["response_format"] != "mp3" {
		t.Errorf("response_format = %v, want mp3", body["response_format"])
	}
	if string(resp.AudioData) != "ID3fake-mp3-bytes" {
		t.Errorf("AudioData = %q, want the server bytes", resp.AudioData)
	}
	if resp.ContentType != "audio/mpeg" {
		t.Errorf("ContentType = %q, want audio/mpeg", resp.ContentType)
	}
}

// TestArbitraryModel confirms a speech model id with no entry in the model
// package reaches the wire, matching llm/openrouter's documented behaviour.
func TestArbitraryModel(t *testing.T) {
	var body map[string]any
	srv := newSpeechServer(t, &body, nil)
	defer srv.Close()

	client := openrouter.NewGeneration(
		ttsopenai.WithBaseURL(srv.URL),
		ttsopenai.WithModel(model.AudioModel{
			APIModel: "some-vendor/unreleased-tts",
		}),
	)

	if _, err := client.GenerateAudio(context.Background(), "x"); err != nil {
		t.Fatalf("GenerateAudio: %v", err)
	}
	if body["model"] != "some-vendor/unreleased-tts" {
		t.Errorf("model = %v, want some-vendor/unreleased-tts", body["model"])
	}
}

// TestWithModelID confirms the shorthand for uncatalogued models reaches the
// wire and tags the provider.
func TestWithModelID(t *testing.T) {
	var body map[string]any
	srv := newSpeechServer(t, &body, nil)
	defer srv.Close()

	client := openrouter.NewGeneration(
		ttsopenai.WithBaseURL(srv.URL),
		openrouter.WithModelID("minimax/speech-2.8-hd"),
	)

	if _, err := client.GenerateAudio(context.Background(), "x"); err != nil {
		t.Fatalf("GenerateAudio: %v", err)
	}
	if body["model"] != "minimax/speech-2.8-hd" {
		t.Errorf("model = %v, want minimax/speech-2.8-hd", body["model"])
	}
	if got := client.Model().Provider; got != model.ProviderOpenRouter {
		t.Errorf("Model().Provider = %q, want %q", got,
			model.ProviderOpenRouter)
	}
}

// TestWireRoutingOptions confirms the OpenRouter routing helpers reach the
// speech request body.
func TestWireRoutingOptions(t *testing.T) {
	var body map[string]any
	srv := newSpeechServer(t, &body, nil)
	defer srv.Close()

	client := openrouter.NewGeneration(
		ttsopenai.WithBaseURL(srv.URL),
		ttsopenai.WithModel(model.AudioModel{APIModel: "m"}),
		openrouter.WithProviderRouting([]string{"openai", "azure"}, false),
		openrouter.WithModelFallbacks("a/b", "c/d"),
	)

	if _, err := client.GenerateAudio(context.Background(), "x"); err != nil {
		t.Fatalf("GenerateAudio: %v", err)
	}

	provider, ok := body["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider = %v (%T), want an object", body["provider"],
			body["provider"])
	}
	if provider["allow_fallbacks"] != false {
		t.Errorf("provider.allow_fallbacks = %v, want false",
			provider["allow_fallbacks"])
	}
	order, ok := provider["order"].([]any)
	if !ok || len(order) != 2 || order[0] != "openai" || order[1] != "azure" {
		t.Errorf("provider.order = %v, want [openai azure]", provider["order"])
	}

	models, ok := body["models"].([]any)
	if !ok || len(models) != 2 || models[0] != "a/b" || models[1] != "c/d" {
		t.Errorf("models = %v, want [a/b c/d]", body["models"])
	}
}

// TestStreamAudio confirms the buffered stream parity path works through the
// wrapper.
func TestStreamAudio(t *testing.T) {
	srv := newSpeechServer(t, nil, nil)
	defer srv.Close()

	client := openrouter.NewGeneration(
		ttsopenai.WithBaseURL(srv.URL),
		ttsopenai.WithModel(model.AudioModel{APIModel: "m"}),
	)

	chunks, err := client.StreamAudio(context.Background(), "x")
	if err != nil {
		t.Fatalf("StreamAudio: %v", err)
	}

	var data []byte
	var sawDone bool
	for chunk := range chunks {
		if chunk.Done {
			sawDone = true
			continue
		}
		data = append(data, chunk.Data...)
	}
	if !sawDone {
		t.Error("stream ended without a Done chunk")
	}
	if string(data) != "ID3fake-mp3-bytes" {
		t.Errorf("streamed data = %q, want the server bytes", data)
	}
}

// TestDefaultBaseURL pins the documented endpoint.
func TestDefaultBaseURL(t *testing.T) {
	if openrouter.DefaultBaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("DefaultBaseURL = %q, want https://openrouter.ai/api/v1",
			openrouter.DefaultBaseURL)
	}
}
