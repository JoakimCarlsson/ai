package jev

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/decision"
)

func TestDecideEncodesQuestionsAndDecodesAnswers(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/systemone" ||
				r.Header.Get("Authorization") != "Bearer k" {
				t.Errorf(
					"unexpected request %s auth=%q",
					r.URL.Path,
					r.Header.Get("Authorization"),
				)
			}
			if r.Header.Get("TypeSafe-Danger-Zone-Images") != "" {
				t.Error("images header sent without images or box")
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &got)
			_, _ = w.Write([]byte(`{"model":"jev-1.14.0","answers":{
			"d":{"type":"choice","choice":"a","confidence":1,"probabilities":{"a":1}},
			"s":{"type":"score","score":0.7,"confidence":0.5,"legend":{"0":"x","1":"y"},"probabilities":{"0":0.3,"1":0.7}},
			"n":{"type":"noul","noul":0.99}},
			"usage":{"input_tokens":10,"output_tokens":2,"cost":0.5}}`))
		}),
	)
	defer srv.Close()

	d := NewDecider(WithAPIKey("k"), WithBaseURL(srv.URL))
	resp, err := d.Decide(
		context.Background(),
		"state",
		map[string]decision.Question{
			"d": decision.Choice("i", map[string]string{"a": "A"}),
			"s": decision.Score("i", "x", "y"),
			"n": decision.NoulWithCriteria("i", "yes", "no"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	qs := got["questions"].(map[string]any)
	if qs["d"].(map[string]any)["type"] != "choice" ||
		qs["s"].(map[string]any)["type"] != "score" {
		t.Errorf("question types not encoded: %v", qs)
	}
	crit := qs["n"].(map[string]any)["criteria"].(map[string]any)
	if crit["true"] != "yes" || crit["false"] != "no" {
		t.Errorf("noul criteria not encoded: %v", crit)
	}
	if got["model"] != "jev-latest" {
		t.Errorf("model = %v", got["model"])
	}

	if c, ok := resp.Choice("d"); !ok || c.Choice != "a" {
		t.Errorf("choice = %+v", c)
	}
	if s, ok := resp.Score("s"); !ok || len(s.Probabilities) != 2 ||
		s.Legend[1] != "y" {
		t.Errorf("score = %+v", s)
	}
	if n, ok := resp.Noul("n"); !ok || n.Probability != 0.99 {
		t.Errorf("noul = %+v", n)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.ReportedCost != 0.5 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestDecideBoxAndImages(t *testing.T) {
	var got map[string]any
	var images, longCtx string
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			images = r.Header.Get("TypeSafe-Danger-Zone-Images")
			longCtx = r.Header.Get("TypeSafe-Danger-Zone-Long-Context")
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &got)
			_, _ = w.Write([]byte(`{"model":"m","answers":{"b":{"type":"box","box":{"top":0.2,"bottom":0.66,"left":0.19,"right":0.68},"confidence":null}},"usage":{"input_tokens":1,"output_tokens":1}}`))
		}),
	)
	defer srv.Close()

	img := decision.NewImage("png", []byte("png-bytes"))
	d := NewDecider(WithAPIKey("k"), WithBaseURL(srv.URL), WithLongContext())
	resp, err := d.Decide(
		context.Background(),
		img.Ref,
		map[string]decision.Question{"b": decision.Box("the thing")},
		decision.WithImages(img),
	)
	if err != nil {
		t.Fatal(err)
	}
	if images != "true" || longCtx != "true" {
		t.Errorf("headers images=%q longContext=%q", images, longCtx)
	}
	asset := got["assets"].(map[string]any)[img.Ref].(map[string]any)
	if asset["type"] != "image" || asset["format"] != "png" ||
		asset["data_base64"] != "cG5nLWJ5dGVz" {
		t.Errorf("asset = %v", asset)
	}
	b, ok := resp.Box("b")
	if !ok || b.Box.Top != 0.2 || b.Box.Right != 0.68 || b.Confidence != nil {
		t.Errorf("box = %+v", b)
	}
}

func TestDecideRejectsBadInput(t *testing.T) {
	d := NewDecider(WithAPIKey("k"), WithBaseURL("http://unused.invalid"))
	ctx := context.Background()
	if _, err := d.Decide(ctx, "s", nil); err == nil {
		t.Error("want error for no questions")
	}
	_, err := d.Decide(ctx, "s",
		map[string]decision.Question{"n": decision.Noul("q")},
		decision.WithImages(decision.NewImage("gif", nil)))
	if err == nil {
		t.Error("want error for unsupported image format")
	}
}

func TestDecideReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
		}),
	)
	defer srv.Close()

	d := NewDecider(WithAPIKey("k"), WithBaseURL(srv.URL))
	_, err := d.Decide(context.Background(), "s",
		map[string]decision.Question{"n": decision.Noul("q")})
	var apiErr *decision.APIError
	if !errors.As(err, &apiErr) || apiErr.GetStatusCode() != 429 ||
		apiErr.GetRetryAfter() != "3" {
		t.Fatalf("err = %v", err)
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"models":[
			{"name":"jev-latest","description":"d","release_date":"2026-10-22T02:00:00+00:00"},
			{"name":"old","description":"d","release_date":"2026-09-15"}]}`))
		}),
	)
	defer srv.Close()

	l, ok := NewDecider(WithAPIKey("k"), WithBaseURL(srv.URL)).(decision.ModelLister)
	if !ok {
		t.Fatal("decider is not a ModelLister")
	}
	ms, err := l.ListModels(context.Background())
	if err != nil || len(ms) != 2 {
		t.Fatalf("models = %v err = %v", ms, err)
	}
	if ms[0].Name != "jev-latest" || ms[0].ReleaseDate.Year() != 2026 ||
		ms[1].ReleaseDate.Month() != 9 {
		t.Errorf("models = %+v", ms)
	}
}
