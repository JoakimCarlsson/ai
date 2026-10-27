package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
)

// TestResponsesDeploymentRouting verifies API-key authentication, deployment
// metadata, version routing and the OpenAI store option on Azure requests.
func TestResponsesDeploymentRouting(t *testing.T) {
	for _, tc := range []struct {
		name           string
		endpointSuffix string
		version        string
		path           string
		query          string
		store          bool
	}{
		{name: "default v1", path: "/openai/v1/responses"},
		{name: "explicit v1", version: "v1", path: "/openai/v1/responses"},
		{name: "v1 endpoint", endpointSuffix: "/openai/v1/", version: "2025-03-01-preview", path: "/openai/v1/responses"},
		{name: "dated preview", version: "2025-03-01-preview", path: "/openai/responses", query: "2025-03-01-preview", store: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			server := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != tc.path ||
						r.URL.Query().Get("api-version") != tc.query {
						t.Errorf("URL = %s", r.URL)
					}
					if r.Header.Get("api-key") != "test-key" {
						t.Error("missing Azure authentication")
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(
						w,
						`{"id":"resp_1","status":"completed","output":[],"usage":{}}`,
					)
				}),
			)
			defer server.Close()
			client := NewResponsesLLM(
				WithEndpoint(server.URL+tc.endpointSuffix),
				WithAPIVersion(tc.version),
				WithAPIKey(
					"test-key",
				),
				WithHTTPClient(server.Client()),
				WithDeployment("gpt-5.6-sol-prod"),
				WithMaxTokens(
					42,
				),
				WithReasoningEffort(llmopenai.ReasoningEffortHigh),
				WithResponsesOptions(llmopenai.WithResponsesStore(tc.store)),
			)
			if _, err := client.SendMessages(context.Background(), []message.Message{message.NewUserMessage("hi")}, nil); err != nil {
				t.Fatal(err)
			}
			if body["model"] != "gpt-5.6-sol-prod" ||
				body["store"] != tc.store ||
				body["max_output_tokens"] != float64(42) {
				t.Fatalf("body = %v", body)
			}
			reasoning, ok := body["reasoning"].(map[string]any)
			if !ok || reasoning["effort"] != "high" {
				t.Fatalf("reasoning = %v", body["reasoning"])
			}
			include, ok := body["include"].([]any)
			if (tc.store && ok) || (!tc.store && (!ok || len(include) != 1 ||
				include[0] != "reasoning.encrypted_content")) {
				t.Fatalf("include = %v", body["include"])
			}
			model := client.Model()
			if !model.CanReason || model.CostPer1MIn == 0 ||
				model.Provider != "azure" {
				t.Fatalf("model = %+v", model)
			}
		})
	}
}

// TestGPT56SolDeploymentMetadata verifies exact and custom deployment names
// share model pricing and capabilities across both Azure constructors.
func TestGPT56SolDeploymentMetadata(t *testing.T) {
	for _, deployment := range []string{"gpt-5.6-sol", "gpt-5.6-sol-prod"} {
		chat := NewLLM(WithDeployment(deployment)).Model()
		responses := NewResponsesLLM(
			WithDeployment(deployment),
			WithResponsesStore(false),
		).Model()
		for _, m := range []struct {
			canReason bool
			apiModel  string
			cost      float64
		}{
			{chat.CanReason, chat.APIModel, chat.CostPer1MIn},
			{responses.CanReason, responses.APIModel, responses.CostPer1MIn},
		} {
			if !m.canReason || m.apiModel != deployment ||
				m.cost != Models[GPT56Sol].CostPer1MIn {
				t.Fatalf("deployment %s: %+v", deployment, m)
			}
		}
	}
}
