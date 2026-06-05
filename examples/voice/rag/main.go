// Package main demonstrates voice.WithKnowledgeBase: retrieval-
// augmented grounding for a voice agent.
//
// At startup the server loads every *.md file under ./data into a
// rag.KnowledgeBase backed by the in-process store. Each WebSocket
// session gets a fresh voice.Agent wired with WithKnowledgeBase(kb)
// plus rag.SearchTool(kb) for follow-up questions.
//
// Wires AssemblyAI (STT) -> OpenAI (LLM + embeddings) -> Deepgram (TTS).
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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	embeddingsopenai "github.com/joakimcarlsson/ai/embeddings/openai"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/rag"
	"github.com/joakimcarlsson/ai/rag/chunkers/fixed"
	ragmem "github.com/joakimcarlsson/ai/rag/store/memory"
	"github.com/joakimcarlsson/ai/session"
	sttassemblyai "github.com/joakimcarlsson/ai/stt/assemblyai"
	ttsdeepgram "github.com/joakimcarlsson/ai/tts/deepgram"
	"github.com/joakimcarlsson/ai/voice"
)

//go:embed prompt.md
var systemPrompt string

const demoSessionID = "voice-rag-session"

func main() {
	openaiKey := requireEnv("OPENAI_API_KEY")
	assemblyKey := requireEnv("ASSEMBLYAI_API_KEY")
	deepgramKey := requireEnv("DEEPGRAM_API_KEY")
	deepgramVoice := envOr("DEEPGRAM_VOICE", string(model.DeepgramAura2Thalia))
	addr := envOr("LISTEN_ADDR", ":8080")

	ctx := context.Background()

	embedder := embeddingsopenai.NewEmbedding(
		embeddingsopenai.WithAPIKey(openaiKey),
		embeddingsopenai.WithModel(
			model.OpenAIEmbeddingModels[model.TextEmbedding3Small],
		),
	)

	kb := rag.New("voice-docs", embedder, ragmem.New(),
		rag.WithChunker(fixed.Default),
	)

	docs, err := loadDocs("data")
	if err != nil {
		slog.Error("load docs", "err", err)
		os.Exit(1)
	}
	if len(docs) == 0 {
		slog.Error("no markdown files found in ./data")
		os.Exit(1)
	}
	if err := kb.Ingest(ctx, docs); err != nil {
		slog.Error("ingest", "err", err)
		os.Exit(1)
	}
	slog.Info("knowledge base ready", "documents", len(docs))

	sessionStore := session.MemoryStore()

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/ws",
		wsHandler(
			openaiKey,
			assemblyKey,
			deepgramKey,
			deepgramVoice,
			kb,
			sessionStore,
		),
	)

	slog.Info("rag voice example listening", "addr", addr)
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
	kb rag.KnowledgeBase,
	sessionStore session.Store,
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

		agent := voice.New(llmClient, sttClient, ttsClient,
			voice.WithSystemPrompt(systemPrompt),
			voice.WithBargeIn(voice.BargeInInterrupt),
			voice.WithSession(demoSessionID, sessionStore),
			voice.WithKnowledgeBase(kb),
			voice.WithTools(rag.SearchTool(kb)),
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

func loadDocs(dir string) ([]rag.Document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var docs []rag.Document
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		docs = append(docs, rag.Document{
			ID:      strings.TrimSuffix(e.Name(), ".md"),
			Content: string(body),
		})
	}
	return docs, nil
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
