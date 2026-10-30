package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joakimcarlsson/ai/decision"
	"github.com/joakimcarlsson/ai/decision/jev"
)

func TestNewDeciderUsesOpenRouterKeyModelAndModelsEndpoint(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "ork")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ork" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/systemone":
			_, _ = w.Write([]byte(`{"model":"typesafe/jev-1.14-x","answers":{"n":{"type":"noul","noul":0.9}},"usage":{"input_tokens":1,"output_tokens":1,"cost":0.001}}`))
		case "/v1/models":
			if r.URL.Query().Get("output_modalities") != "decisions" {
				t.Errorf("query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"typesafe/jev-1.14","description":"d","created":1793325134}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	d := NewDecider(jev.WithBaseURL(srv.URL))
	if d.Model().APIModel != "~typesafe/jev-latest" {
		t.Errorf("model = %q", d.Model().APIModel)
	}
	resp, err := d.Decide(context.Background(), "s",
		map[string]decision.Question{"n": decision.Noul("q")})
	if err != nil || resp.Usage.ReportedCost != 0.001 {
		t.Fatalf("resp = %+v err = %v", resp, err)
	}
	ms, err := d.(decision.ModelLister).ListModels(context.Background())
	if err != nil || len(ms) != 1 || ms[0].Name != "typesafe/jev-1.14" ||
		ms[0].ReleaseDate.IsZero() {
		t.Fatalf("models = %+v err = %v", ms, err)
	}
}
