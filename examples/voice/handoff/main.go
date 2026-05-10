// Package main runs a small HTTP server that demonstrates voice.WithHandoffs.
// A "triage" agent answers general questions and transfers to a "billing"
// specialist for refund / charge questions. Both agents share the same STT
// and TTS clients so the audio path doesn't blink at the handoff boundary.
//
// Wires AssemblyAI (STT) -> OpenAI (LLM) -> Deepgram (TTS).
//
//	OPENAI_API_KEY=... ASSEMBLYAI_API_KEY=... DEEPGRAM_API_KEY=... go run .
//	# then in another terminal:
//	cd ui && npm install && npm run dev
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	sttassemblyai "github.com/joakimcarlsson/ai/stt/assemblyai"
	"github.com/joakimcarlsson/ai/tool"
	ttsdeepgram "github.com/joakimcarlsson/ai/tts/deepgram"
	"github.com/joakimcarlsson/ai/voice"
)

//go:embed triage_prompt.md
var triagePrompt string

//go:embed billing_prompt.md
var billingPrompt string

func main() {
	openaiKey := requireEnv("OPENAI_API_KEY")
	assemblyKey := requireEnv("ASSEMBLYAI_API_KEY")
	deepgramKey := requireEnv("DEEPGRAM_API_KEY")
	deepgramVoice := envOr("DEEPGRAM_VOICE", string(model.DeepgramAura2Thalia))
	addr := envOr("LISTEN_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/ws",
		wsHandler(openaiKey, assemblyKey, deepgramKey, deepgramVoice),
	)

	slog.Info("handoff example listening", "addr", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		slog.Error("http listen", "err", err)
		os.Exit(1)
	}
}

func wsHandler(
	openaiKey, assemblyKey, deepgramKey, deepgramVoice string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{
				"localhost:5173",
				"127.0.0.1:5173",
				"localhost:8080",
			},
		})
		if err != nil {
			slog.Warn("ws accept", "err", err)
			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		llmClient := llmopenai.NewLLM(
			llmopenai.WithAPIKey(openaiKey),
			llmopenai.WithModel(model.OpenAIModels[model.GPT54Mini]),
			llmopenai.WithMaxTokens(2048),
		)

		sttClient := sttassemblyai.NewSpeechToText(
			sttassemblyai.WithAPIKey(assemblyKey),
			sttassemblyai.WithModel(
				model.AssemblyAITranscriptionModels[model.AssemblyAIU3RTPro],
			),
		)

		ttsClient := ttsdeepgram.NewGeneration(
			ttsdeepgram.WithAPIKey(deepgramKey),
			ttsdeepgram.WithModelName(deepgramVoice),
			ttsdeepgram.WithEncoding("linear16"),
			ttsdeepgram.WithSampleRate(16000),
		)

		// Specialist: handles refunds via the issueRefundTool.
		specialist := voice.New(llmClient, sttClient, ttsClient,
			voice.WithSystemPrompt(billingPrompt),
			voice.WithTools(issueRefundTool{}),
			voice.WithBargeIn(voice.BargeInInterrupt),
		)

		// Triage: general front-of-house. Hands off to billing when the
		// user asks about money.
		triage := voice.New(llmClient, sttClient, ttsClient,
			voice.WithSystemPrompt(triagePrompt),
			voice.WithBargeIn(voice.BargeInInterrupt),
			voice.WithHandoffs(voice.HandoffConfig{
				Name:        "billing",
				Description: "Use this when the user asks about charges, refunds, invoices, or anything involving money.",
				Agent:       specialist,
			}),
		)

		transport := newWSTransport(conn)
		convo, err := triage.StartConversation(ctx, transport)
		if err != nil {
			writeJSON(ctx, conn, eventEnvelope{
				Type:  "error",
				Error: err.Error(),
			})
			return
		}

		go forwardEvents(ctx, convo, conn)

		if err := convo.Wait(); err != nil &&
			!errors.Is(err, context.Canceled) {
			slog.Warn("conversation ended", "err", err)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "session ended")
	}
}

type eventEnvelope struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Tool     string `json:"tool,omitempty"`
	ToolID   string `json:"toolId,omitempty"`
	ToolArgs string `json:"toolArgs,omitempty"`
	Output   string `json:"output,omitempty"`
	IsError  bool   `json:"isError,omitempty"`
	Error    string `json:"error,omitempty"`
}

func forwardEvents(
	ctx context.Context,
	convo *voice.Conversation,
	conn *websocket.Conn,
) {
	for evt := range convo.Events() {
		env := eventEnvelope{Type: string(evt.Type), Text: evt.Text}
		if evt.ToolCall != nil {
			env.Tool = evt.ToolCall.Name
			env.ToolID = evt.ToolCall.ID
			env.ToolArgs = evt.ToolCall.Input
		}
		if evt.ToolResult != nil {
			env.Output = evt.ToolResult.Output
			env.IsError = evt.ToolResult.IsError
		}
		if evt.Error != nil {
			env.Error = evt.Error.Error()
		}
		writeJSON(ctx, conn, env)
	}
}

type wsTransport struct {
	conn   *websocket.Conn
	writeM sync.Mutex
}

func newWSTransport(conn *websocket.Conn) *wsTransport {
	return &wsTransport{conn: conn}
}

func (t *wsTransport) Read(ctx context.Context) ([]byte, error) {
	for {
		typ, data, err := t.conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		if typ == websocket.MessageBinary {
			return data, nil
		}
	}
}

func (t *wsTransport) Write(ctx context.Context, frame []byte) error {
	t.writeM.Lock()
	defer t.writeM.Unlock()
	return t.conn.Write(ctx, websocket.MessageBinary, frame)
}

func (t *wsTransport) Close() error {
	return t.conn.Close(websocket.StatusNormalClosure, "transport closed")
}

func writeJSON(ctx context.Context, conn *websocket.Conn, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = conn.Write(ctx, websocket.MessageText, b)
}

// issueRefundTool is the specialist agent's tool. It does nothing real —
// just confirms a refund of a hardcoded amount so the demo has something
// to call. Replace with a real billing API in production.
type issueRefundTool struct{}

func (issueRefundTool) Info() tool.Info {
	return tool.Info{
		Name: "issue_refund",
		Description: "Issues a refund to the user's payment method on file. " +
			"Use after confirming with the user. Takes no arguments.",
		Parameters: map[string]any{},
	}
}

func (issueRefundTool) Run(_ context.Context, _ tool.Call) (tool.Response, error) {
	return tool.NewTextResponse(
		"Refund issued: $25.00 will appear on the user's statement within 3-5 business days.",
	), nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s is required\n", name)
		os.Exit(1)
	}
	return v
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
