package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const indexBody = `{"providers":[{"id":"demo","name":"Demo","models":[
	{"id":"demo-1","provider":"demo","name":"Demo 1","kind":"chat",
	 "prices":[{"metric":"input_tokens","unit":"per_1m_tokens",
	            "amount":2,"currency":"USD","dims":{"tier":"standard"}}],
	 "attrs":{"api_id":"demo-1"},"limits":{"context_window":128000},
	 "lists":{"features":["reasoning"],"input_modalities":["text"]}},
	{"id":"demo-voice","provider":"demo","name":"Demo Voice","kind":"speech",
	 "attrs":{"api_id":"demo-voice"}}
]}]}`

func serve(t *testing.T, status int, body string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		},
	))
	t.Cleanup(srv.Close)
	return srv.URL + "/api.json"
}

func TestFetchIndexReadsAServedDocument(t *testing.T) {
	index, err := fetchIndex(context.Background(), serve(t, 200, indexBody))
	if err != nil {
		t.Fatalf("fetchIndex: %v", err)
	}

	served := index.models("demo", "chat")
	if len(served) != 1 || served[0].apiID() != "demo-1" {
		t.Fatalf("chat models = %v", served)
	}
	if got := index.models("demo", "speech"); len(got) != 1 {
		t.Errorf("speech models = %v, want the one served", got)
	}
	if got := index.models("demo", "rerank"); len(got) != 0 {
		t.Errorf("rerank models = %v, want none", got)
	}
	if got := index.models("nobody", "chat"); len(got) != 0 {
		t.Errorf("models for an unlisted provider = %v, want none", got)
	}

	m := modelFor(chat("demo", "llm/demo", "demo"), served[0])
	if m.fields["CostPer1MIn"] != "2" {
		t.Errorf("CostPer1MIn = %q, want 2", m.fields["CostPer1MIn"])
	}
}

func TestFetchIndexReportsABadStatus(t *testing.T) {
	_, err := fetchIndex(context.Background(), serve(t, 404, "not found"))
	if err == nil {
		t.Fatal("want an error for a 404")
	}
	if !strings.Contains(err.Error(), "unexpected status 404") {
		t.Errorf("error = %v, want it to name the status", err)
	}
}

func TestFetchIndexReportsUndecodableBody(t *testing.T) {
	_, err := fetchIndex(context.Background(), serve(t, 200, "<html>nope"))
	if err == nil {
		t.Fatal("want an error for a body that is not the document")
	}
	if !strings.Contains(err.Error(), "decoding") {
		t.Errorf("error = %v, want a decoding error", err)
	}
}

func TestFetchIndexReportsAnUnreachableHost(t *testing.T) {
	url := serve(t, 200, indexBody)
	srv := httptest.NewServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))
	dead := srv.URL + "/api.json"
	srv.Close()

	if _, err := fetchIndex(context.Background(), dead); err == nil {
		t.Fatal("want an error when the host does not answer")
	} else if !strings.Contains(err.Error(), "fetching") {
		t.Errorf("error = %v, want a fetch error", err)
	}

	if _, err := fetchIndex(context.Background(), url); err != nil {
		t.Fatalf("the live server still answers: %v", err)
	}
}
