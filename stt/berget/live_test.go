package berget

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/stt"
)

func liveAudio(t *testing.T) (string, []byte) {
	t.Helper()
	k := os.Getenv("BERGET_API_KEY")
	audio := os.Getenv("BERGET_TEST_AUDIO")
	if k == "" || audio == "" {
		t.Skip("set BERGET_API_KEY and BERGET_TEST_AUDIO")
	}
	data, err := os.ReadFile(audio)
	if err != nil {
		t.Fatal(err)
	}
	return k, data
}

func TestLive(t *testing.T) {
	k, data := liveAudio(t)
	c := NewSpeechToText(
		WithAPIKey(k),
		WithModel(model.BergetTranscriptionModels[model.BergetFasterWhisperLargeV3]),
		WithTimeout(60*time.Second),
	)
	r, err := c.Transcribe(context.Background(), data, stt.WithFilename("speech.wav"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("text=%q language=%q segments=%d words=%d", r.Text, r.Language, len(r.Segments), len(r.Words))
	if !strings.Contains(strings.ToLower(r.Text), "quick brown fox") {
		t.Fatalf("transcript missing expected phrase: %q", r.Text)
	}
	if len(r.Words) == 0 {
		t.Fatal("no word timestamps parsed")
	}
}

func TestLiveAllModels(t *testing.T) {
	k, data := liveAudio(t)
	ids := []model.ID{
		model.BergetFasterWhisperLargeV3,
		model.BergetKBWhisperLarge,
		model.BergetNBWhisperLarge,
	}
	var ok int
	for _, id := range ids {
		c := NewSpeechToText(WithAPIKey(k), WithModel(model.BergetTranscriptionModels[id]), WithTimeout(60*time.Second))
		r, err := c.Transcribe(context.Background(), data, stt.WithFilename("speech.wav"))
		if err != nil {
			t.Logf("FAIL %-40s %v", id, err)
			continue
		}
		ok++
		t.Logf("OK   %-40s text=%q words=%d", id, r.Text, len(r.Words))
	}
	t.Logf("transcription models reachable: %d/%d", ok, len(ids))
	if ok == 0 {
		t.Fatal("no transcription models reachable")
	}
}

func TestTranslateUnsupported(t *testing.T) {
	c := NewSpeechToText(WithAPIKey("x"), WithModel(model.BergetTranscriptionModels[model.BergetFasterWhisperLargeV3]))
	_, err := c.Translate(context.Background(), []byte("audio"))
	if !errors.Is(err, ErrTranslationNotSupported) {
		t.Fatalf("got %v, want ErrTranslationNotSupported", err)
	}
}
