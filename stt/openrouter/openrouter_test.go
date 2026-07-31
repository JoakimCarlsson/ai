package openrouter_test

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
	sttopenai "github.com/joakimcarlsson/ai/stt/openai"
	"github.com/joakimcarlsson/ai/stt/openrouter"
)

// verboseJSON is the OpenAI-shaped verbose_json body OpenRouter returns from
// its OpenAI-compatible upstreams, complete with segments and words.
const verboseJSON = `{
	"task": "transcribe",
	"language": "en",
	"duration": 2.5,
	"text": "Hello there.",
	"segments": [{
		"id": 0, "start": 0.0, "end": 2.5, "text": "Hello there.",
		"tokens": [50364, 2425], "temperature": 0.0,
		"avg_logprob": -0.31, "compression_ratio": 0.72,
		"no_speech_prob": 0.01
	}],
	"words": [
		{"word": "Hello", "start": 0.0, "end": 1.0},
		{"word": "there", "start": 1.0, "end": 2.5}
	],
	"usage": {"seconds": 2.5, "input_tokens": 83, "output_tokens": 30,
	          "total_tokens": 113, "cost": 0.000508}
}`

// transcriptionRequest is what the server observed: the request path plus the
// multipart form fields, since the wrapper takes OpenRouter's OpenAI-style
// multipart path rather than its JSON input_audio variant.
type transcriptionRequest struct {
	path     string
	fields   map[string]string
	filename string
	fileSize int
}

func newTranscriptionServer(
	t *testing.T,
	observed *transcriptionRequest,
	response string,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if observed != nil {
				observed.path = r.URL.Path
				observed.fields = map[string]string{}
				readMultipart(t, r, observed)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, response)
		}))
}

func readMultipart(
	t *testing.T,
	r *http.Request,
	observed *transcriptionRequest,
) {
	t.Helper()
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		t.Errorf("parse content type: %v", err)
		return
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Errorf("read multipart part: %v", err)
			return
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Errorf("read part body: %v", err)
			return
		}
		if part.FormName() == "file" {
			observed.filename = part.FileName()
			observed.fileSize = len(data)
			continue
		}
		observed.fields[part.FormName()] = string(data)
	}
}

// TestTranscribe confirms the wrapper posts to OpenRouter's transcription path
// and that segments, words and timestamps decode from the verbose_json body —
// the fields most likely to diverge from OpenAI's shape.
func TestTranscribe(t *testing.T) {
	var observed transcriptionRequest
	srv := newTranscriptionServer(t, &observed, verboseJSON)
	defer srv.Close()

	client := openrouter.NewSpeechToText(
		sttopenai.WithAPIKey("test-key"),
		sttopenai.WithBaseURL(srv.URL),
		sttopenai.WithModel(
			model.OpenRouterTranscriptionModels[model.OpenRouterWhisper1],
		),
	)

	resp, err := client.Transcribe(
		context.Background(),
		[]byte("fake-audio-bytes"),
		stt.WithLanguage("en"),
		stt.WithTimestampGranularities("word", "segment"),
	)
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}

	if observed.path != "/audio/transcriptions" {
		t.Errorf("request path = %q, want /audio/transcriptions", observed.path)
	}
	if observed.fields["model"] != "openai/whisper-1" {
		t.Errorf("model = %q, want openai/whisper-1", observed.fields["model"])
	}
	if observed.fields["language"] != "en" {
		t.Errorf("language = %q, want en", observed.fields["language"])
	}
	if observed.fields["response_format"] != "verbose_json" {
		t.Errorf("response_format = %q, want verbose_json",
			observed.fields["response_format"])
	}
	if observed.fileSize != len("fake-audio-bytes") {
		t.Errorf("uploaded %d bytes, want %d", observed.fileSize,
			len("fake-audio-bytes"))
	}

	if resp.Text != "Hello there." {
		t.Errorf("Text = %q, want Hello there.", resp.Text)
	}
	if resp.Language != "en" {
		t.Errorf("Language = %q, want en", resp.Language)
	}
	if resp.Duration != 2.5 {
		t.Errorf("Duration = %v, want 2.5", resp.Duration)
	}
	if len(resp.Segments) != 1 {
		t.Fatalf("len(Segments) = %d, want 1", len(resp.Segments))
	}
	if resp.Segments[0].End != 2.5 || resp.Segments[0].Text != "Hello there." {
		t.Errorf("Segments[0] = %+v, want the full utterance 0.0-2.5",
			resp.Segments[0])
	}
	if len(resp.Words) != 2 {
		t.Fatalf("len(Words) = %d, want 2", len(resp.Words))
	}
	if resp.Words[1].Word != "there" || resp.Words[1].Start != 1.0 {
		t.Errorf("Words[1] = %+v, want there at 1.0", resp.Words[1])
	}
}

// TestArbitraryModel confirms a transcription model id with no entry in the
// model package reaches the wire, matching llm/openrouter's documented
// behaviour.
func TestArbitraryModel(t *testing.T) {
	var observed transcriptionRequest
	srv := newTranscriptionServer(t, &observed, `{"text":"hi"}`)
	defer srv.Close()

	client := openrouter.NewSpeechToText(
		sttopenai.WithBaseURL(srv.URL),
		sttopenai.WithModel(model.TranscriptionModel{
			APIModel: "some-vendor/unreleased-stt",
		}),
	)

	if _, err := client.Transcribe(
		context.Background(), []byte("audio"),
	); err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if observed.fields["model"] != "some-vendor/unreleased-stt" {
		t.Errorf("model = %q, want some-vendor/unreleased-stt",
			observed.fields["model"])
	}
}

// TestWithModelID confirms the shorthand for uncatalogued models reaches the
// wire and tags the provider.
func TestWithModelID(t *testing.T) {
	var observed transcriptionRequest
	srv := newTranscriptionServer(t, &observed, `{"text":"hi"}`)
	defer srv.Close()

	client := openrouter.NewSpeechToText(
		sttopenai.WithBaseURL(srv.URL),
		openrouter.WithModelID("nvidia/parakeet-tdt-0.6b-v3"),
	)

	if _, err := client.Transcribe(
		context.Background(), []byte("audio"), stt.WithResponseFormat("json"),
	); err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if observed.fields["model"] != "nvidia/parakeet-tdt-0.6b-v3" {
		t.Errorf("model = %q, want nvidia/parakeet-tdt-0.6b-v3",
			observed.fields["model"])
	}
	if got := client.Model().Provider; got != model.ProviderOpenRouter {
		t.Errorf("Model().Provider = %q, want %q", got,
			model.ProviderOpenRouter)
	}
}

// TestTranscribePlainJSON covers the response_format="json" path callers need
// for the upstreams that reject verbose_json with HTTP 400.
func TestTranscribePlainJSON(t *testing.T) {
	var observed transcriptionRequest
	srv := newTranscriptionServer(t, &observed, `{"text":"Hello there.",
		"usage":{"input_tokens":83,"output_tokens":30,"total_tokens":113}}`)
	defer srv.Close()

	client := openrouter.NewSpeechToText(
		sttopenai.WithBaseURL(srv.URL),
		sttopenai.WithModel(
			model.OpenRouterTranscriptionModels[model.OpenRouterVoxtralMiniTranscribe],
		),
	)

	resp, err := client.Transcribe(
		context.Background(),
		[]byte("audio"),
		stt.WithResponseFormat("json"),
	)
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if observed.fields["response_format"] != "json" {
		t.Errorf("response_format = %q, want json",
			observed.fields["response_format"])
	}
	if resp.Text != "Hello there." {
		t.Errorf("Text = %q, want Hello there.", resp.Text)
	}
	if len(resp.Segments) != 0 {
		t.Errorf("Segments = %+v, want none for a plain json body",
			resp.Segments)
	}
}

// TestTranslateNotSupported confirms Translate fails fast rather than issuing a
// request to a route OpenRouter does not publish.
func TestTranslateNotSupported(t *testing.T) {
	var reached bool
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			reached = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"text":"should not happen"}`)
		}))
	defer srv.Close()

	client := openrouter.NewSpeechToText(
		sttopenai.WithBaseURL(srv.URL),
		sttopenai.WithModel(model.TranscriptionModel{APIModel: "m"}),
	)

	_, err := client.Translate(context.Background(), []byte("audio"))
	if !errors.Is(err, openrouter.ErrTranslationNotSupported) {
		t.Errorf("Translate error = %v, want ErrTranslationNotSupported", err)
	}
	if reached {
		t.Error("Translate issued an HTTP request; it should fail before that")
	}
}

// TestStreamingNotSupported confirms the wrapper inherits the openai package's
// non-streaming behaviour rather than silently claiming support.
func TestStreamingNotSupported(t *testing.T) {
	client := openrouter.NewSpeechToText(
		sttopenai.WithModel(model.TranscriptionModel{APIModel: "m"}),
	)

	if client.SupportsStreaming() {
		t.Error("SupportsStreaming = true, want false")
	}
	_, err := client.StreamTranscribe(context.Background(), nil)
	if !errors.Is(err, stt.ErrStreamingNotSupported) {
		t.Errorf("StreamTranscribe error = %v, want ErrStreamingNotSupported",
			err)
	}
}

// TestModel confirms the configured model survives the wrapper.
func TestModel(t *testing.T) {
	client := openrouter.NewSpeechToText(
		sttopenai.WithModel(
			model.OpenRouterTranscriptionModels[model.OpenRouterGPT4oTranscribe],
		),
	)
	if got := client.Model().APIModel; got != "openai/gpt-4o-transcribe" {
		t.Errorf("Model().APIModel = %q, want openai/gpt-4o-transcribe", got)
	}
}

// TestDefaultBaseURL pins the documented endpoint.
func TestDefaultBaseURL(t *testing.T) {
	if openrouter.DefaultBaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("DefaultBaseURL = %q, want https://openrouter.ai/api/v1",
			openrouter.DefaultBaseURL)
	}
}
