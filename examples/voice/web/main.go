// Package main runs a small HTTP server that exposes a voice agent over a
// WebSocket. The browser client (in ./ui) captures mic audio at 16 kHz mono
// PCM, streams it to /ws as binary frames, and plays back the agent's audio
// response. Text events (transcripts, tool calls, deltas) are sent as JSON.
//
// Wires AssemblyAI (STT) -> OpenAI (LLM) -> Deepgram (TTS).
//
//	OPENAI_API_KEY=... ASSEMBLYAI_API_KEY=... DEEPGRAM_API_KEY=... go run .
//	# then in another terminal:
//	cd ui && npm install && npm run dev
//
// The Vite dev server proxies /ws to localhost:8080.
package main

import (
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/prompt"
	sttassemblyai "github.com/joakimcarlsson/ai/stt/assemblyai"
	"github.com/joakimcarlsson/ai/tool"
	ttsdeepgram "github.com/joakimcarlsson/ai/tts/deepgram"
	"github.com/joakimcarlsson/ai/voice"
)

//go:embed prompt.md
var systemPromptTemplate string

// toolSoundClip is a synthesized 2-second loop of soft "thinking" clicks at
// 16 kHz mono PCM16 LE. Replace with a real recording for production use.
var toolSoundClip = generateThinkingClicks(16000, 2*time.Second)

func generateThinkingClicks(sampleRate int, duration time.Duration) []byte {
	samples := int(float64(sampleRate) * duration.Seconds())
	out := make([]byte, samples*2)
	burstInterval := sampleRate / 3 // a click every ~330ms
	burstLen := sampleRate / 25     // 40ms burst
	for i := 0; i < samples; i++ {
		var sample int16
		pos := i % burstInterval
		if pos < burstLen {
			t := float64(pos) / float64(burstLen)
			envelope := 1.0 - t
			// Mix two frequencies for a more pronounced "tap" sound.
			phase1 := 2 * math.Pi * 600 * float64(pos) / float64(sampleRate)
			phase2 := 2 * math.Pi * 1100 * float64(pos) / float64(sampleRate)
			signal := 0.6*math.Sin(phase1) + 0.4*math.Sin(phase2)
			sample = int16(envelope * 16000 * signal)
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func main() {
	openaiKey := requireEnv("OPENAI_API_KEY")
	assemblyKey := requireEnv("ASSEMBLYAI_API_KEY")
	deepgramKey := requireEnv("DEEPGRAM_API_KEY")
	deepgramVoice := envOr("DEEPGRAM_VOICE", string(model.DeepgramAura2Thalia))
	agentName := envOr("AGENT_NAME", "Aura")

	addr := envOr("LISTEN_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/ws",
		wsHandler(
			openaiKey,
			assemblyKey,
			deepgramKey,
			deepgramVoice,
			agentName,
		),
	)

	slog.Info("voice example listening", "addr", addr)
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
	openaiKey, assemblyKey, deepgramKey, deepgramVoice, agentName string,
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

		systemPrompt, err := prompt.Process(
			systemPromptTemplate,
			map[string]any{
				"AgentName": agentName,
				"Today":     time.Now().Format("Monday, January 2, 2006"),
			},
		)
		if err != nil {
			slog.Error("render system prompt", "err", err)
			_ = conn.Close(
				websocket.StatusInternalError,
				"prompt render failed",
			)
			return
		}

		llmClient := llmopenai.NewLLM(
			llmopenai.WithAPIKey(openaiKey),
			llmopenai.WithModel(model.OpenAIModels[model.GPT54Nano]),
			llmopenai.WithMaxTokens(4096),
		)

		slog.Info("voice agent ready",
			"agent", agentName,
			"tool", currentTimeTool{}.Info().Name,
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
			voice.WithTools(currentTimeTool{}),
			voice.WithFiller(voice.FillerConfig{
				Timeout: 1500 * time.Millisecond,
				Message: "One moment.",
			}),
			voice.WithToolSound(voice.ToolSoundConfig{
				Audio:    toolSoundClip,
				Behavior: voice.ToolSoundAlways,
			}),
		)

		transport := newWSTransport(conn)
		conv, err := agent.StartConversation(ctx, transport)
		if err != nil {
			writeJSON(ctx, conn, eventEnvelope{
				Type:  "error",
				Error: err.Error(),
			})
			return
		}

		go forwardEvents(ctx, conv, conn)

		if err := conv.Wait(); err != nil &&
			!errors.Is(err, context.Canceled) {
			slog.Warn("conversation ended", "err", err)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "session ended")
	}
}

// eventEnvelope is the JSON shape sent to the browser for non-audio events.
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
	conv *voice.Conversation,
	conn *websocket.Conn,
) {
	for evt := range conv.Events() {
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

// wsTransport adapts a *websocket.Conn to voice.AudioTransport.
//
// Read returns one binary frame per call (mono PCM 16 kHz from the browser).
// Text frames from the browser are silently dropped here; control messages
// could be added later if needed.
//
// Write sends one TTS audio frame as a binary message. A mutex serializes
// writes so the audio path does not race with the events forwarder.
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
		// Drop text frames; they are reserved for future control messages.
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

// currentTimeTool returns the current wall-clock time as a sentence the agent
// can read aloud. Demonstrates how a voice agent invokes tools mid-turn: the
// LLM calls this when the user asks for the time, the result lands in history,
// and the LLM continues its reply incorporating it.
type currentTimeTool struct{}

func (currentTimeTool) Info() tool.Info {
	return tool.Info{
		Name: "get_current_time",
		Description: "Returns the current date and time. " +
			"Call this whenever the user asks what time, what day, or what date it is. " +
			"Takes no arguments. Do not ask the user for a time zone first.",
		Parameters: map[string]any{},
	}
}

func (currentTimeTool) Run(ctx context.Context, _ tool.Call) (tool.Response, error) {
	// Artificial 2s delay so the tool-sound loop is audible long enough
	// to demo. Respects ctx so it cancels cleanly if the conversation ends.
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return tool.Response{}, ctx.Err()
	}
	now := time.Now()
	return tool.NewTextResponse(
		now.Format("Monday, January 2, 2006 at 3:04 PM"),
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
