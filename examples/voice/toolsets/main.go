// Package main demonstrates voice.WithToolsets — dynamic, per-call tool
// resolution. The agent has one always-available tool (lookup_order) and a
// dynamic toolset (admin_tools) that returns the staff-only refund and
// block tools only when the WebSocket request authenticates as staff.
//
// Connect to /ws         → user mode (only lookup_order is available).
// Connect to /ws?role=admin → staff mode (refund + block become available).
//
// The role is stashed into the conversation context; the toolset's
// Tools(ctx) method reads it on every LLM call. Toggling between
// connections proves the per-call resolution: the LLM sees a different
// tool list on each turn.
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

//go:embed prompt.md
var systemPrompt string

// roleKey is the context value used by the WS handler to stash the
// authenticated role. The toolset reads it back to decide which tools to
// expose.
type roleKey struct{}

func withRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

func roleFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(roleKey{}).(string); ok {
		return r
	}
	return "user"
}

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

	slog.Info("toolsets example listening", "addr", addr,
		"hint", "connect to /ws?role=admin to unlock staff tools")
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

		role := r.URL.Query().Get("role")
		if role == "" {
			role = "user"
		}
		ctx = withRole(ctx, role)
		slog.Info("ws connected", "role", role)

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

		agent := voice.New(llmClient, sttClient, ttsClient,
			voice.WithSystemPrompt(systemPrompt),
			voice.WithBargeIn(voice.BargeInInterrupt),

			// Always-available tool, registered statically.
			voice.WithTools(lookupOrderTool{}),

			// Dynamic toolset: returns admin tools only when the
			// authenticated role permits them. The Tools(ctx) method
			// is invoked before EVERY LLM call, so a role change
			// (e.g., elevation through your auth flow) takes effect
			// on the next turn.
			voice.WithToolsets(adminToolset{}),
		)

		transport := newWSTransport(conn)
		convo, err := agent.StartConversation(ctx, transport)
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

// adminToolset returns the refund and block tools when the conversation
// context carries role=admin, and nothing otherwise. This is the heart of
// the demo: voice's runner calls Tools(ctx) before every LLM call, so the
// LLM literally sees a different tool list depending on who's connected.
type adminToolset struct{}

func (adminToolset) Name() string { return "admin" }

func (adminToolset) Tools(ctx context.Context) []tool.BaseTool {
	if roleFromContext(ctx) != "admin" {
		return nil
	}
	return []tool.BaseTool{issueRefundTool{}, blockCustomerTool{}}
}

// lookupOrderTool is always available. Returns a hardcoded order summary.
type lookupOrderTool struct{}

type lookupOrderInput struct {
	OrderID string `json:"order_id" desc:"the order id" required:"true"`
}

func (lookupOrderTool) Info() tool.Info {
	return tool.NewInfo(
		"lookup_order",
		"Looks up an order by its id and returns status and total.",
		lookupOrderInput{},
	)
}

func (lookupOrderTool) Run(
	_ context.Context,
	c tool.Call,
) (tool.Response, error) {
	var in lookupOrderInput
	_ = json.Unmarshal([]byte(c.Input), &in)
	if in.OrderID == "" {
		return tool.NewTextErrorResponse("order_id is required"), nil
	}
	return tool.NewTextResponse(fmt.Sprintf(
		"Order %s: shipped, total $42.00, delivered yesterday.", in.OrderID,
	)), nil
}

// issueRefundTool — staff-only.
type issueRefundTool struct{}

func (issueRefundTool) Info() tool.Info {
	return tool.Info{
		Name: "issue_refund",
		Description: "Issues a refund to the user's payment method on file. " +
			"Staff only.",
		Parameters: map[string]any{},
	}
}

func (issueRefundTool) Run(
	_ context.Context,
	_ tool.Call,
) (tool.Response, error) {
	return tool.NewTextResponse(
		"Refund issued: $25.00 will appear on the customer's statement within 3-5 business days.",
	), nil
}

// blockCustomerTool — staff-only.
type blockCustomerTool struct{}

func (blockCustomerTool) Info() tool.Info {
	return tool.Info{
		Name:        "block_customer",
		Description: "Blocks a customer for fraud. Staff only.",
		Parameters:  map[string]any{},
	}
}

func (blockCustomerTool) Run(
	_ context.Context,
	_ tool.Call,
) (tool.Response, error) {
	return tool.NewTextResponse(
		"Customer blocked. Future orders will be declined.",
	), nil
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
