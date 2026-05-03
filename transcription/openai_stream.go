package transcription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	openAIRealtimeDefaultModel        = "gpt-4o-transcribe"
	openAIRealtimeDefaultVADSilenceMs = 500
	openAIRealtimeReadDeadline        = 30 * time.Second
	openAIRealtimeHandshakeTimeout    = 10 * time.Second
)

// streamTranscribe opens an OpenAI Realtime WebSocket session in
// transcription-only mode. Audio frames are base64-encoded PCM16 sent as
// input_audio_buffer.append events. Reader accumulates deltas per item_id
// and emits StreamResult{IsFinal:false} on each delta and IsFinal:true on
// the completed event.
func (o *openaiClient) streamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	opts := Options{}
	for _, op := range options {
		op(&opts)
	}

	model := o.providerOptions.model.APIModel
	if model == "" {
		model = openAIRealtimeDefaultModel
	}
	silenceMs := openAIRealtimeDefaultVADSilenceMs
	if opts.EndpointingMs != nil {
		silenceMs = *opts.EndpointingMs
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.openai.com",
		Path:     "/v1/realtime",
		RawQuery: "intent=transcription",
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+o.providerOptions.apiKey)
	hdr.Set("OpenAI-Beta", "realtime=v1")

	dialer := websocket.Dialer{HandshakeTimeout: openAIRealtimeHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	out := make(chan StreamResult)
	done := make(chan struct{})

	var writeMu sync.Mutex
	send := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	cfg := fmt.Sprintf(
		`{"type":"session.update","session":{"input_audio_format":"pcm16","input_audio_transcription":{"model":%q},"turn_detection":{"type":"server_vad","silence_duration_ms":%d}}}`,
		model, silenceMs,
	)
	if err := send(websocket.TextMessage, []byte(cfg)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(openAIRealtimeReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(openAIRealtimeReadDeadline))
	})

	go runOpenAIRealtimeReader(conn, out, done)
	go runOpenAIRealtimeWriter(ctx, conn, audio, out, done, send)

	return out, nil
}

func runOpenAIRealtimeReader(
	conn *websocket.Conn,
	out chan<- StreamResult,
	done chan<- struct{},
) {
	defer close(done)
	deltas := map[string]string{}
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !isCleanWSClose(err) {
				out <- StreamResult{Error: err}
			}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(openAIRealtimeReadDeadline))
		pr, ok := parseOpenAIRealtime(msg, deltas)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runOpenAIRealtimeWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- StreamResult,
	done <-chan struct{},
	send func(int, []byte) error,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	commit := func() {
		_ = send(websocket.TextMessage, []byte(`{"type":"input_audio_buffer.commit"}`))
	}

	audioOpen := true
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			commit()
			<-done
			return
		case frame, ok := <-audio:
			if !ok {
				if audioOpen {
					commit()
					audioOpen = false
				}
				audio = nil
				continue
			}
			payload := fmt.Sprintf(
				`{"type":"input_audio_buffer.append","audio":"%s"}`,
				base64.StdEncoding.EncodeToString(frame),
			)
			if err := send(websocket.TextMessage, []byte(payload)); err != nil {
				out <- StreamResult{Error: err}
				_ = conn.Close()
				<-done
				return
			}
		}
	}
}

type openAIRealtimeEvent struct {
	Type       string `json:"type"`
	ItemID     string `json:"item_id"`
	Delta      string `json:"delta"`
	Transcript string `json:"transcript"`
}

func parseOpenAIRealtime(raw []byte, deltas map[string]string) (StreamResult, bool) {
	var ev openAIRealtimeEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return StreamResult{}, false
	}
	switch ev.Type {
	case "conversation.item.input_audio_transcription.delta":
		if ev.Delta == "" {
			return StreamResult{}, false
		}
		deltas[ev.ItemID] += ev.Delta
		return StreamResult{
			Text:    deltas[ev.ItemID],
			IsFinal: false,
		}, true
	case "conversation.item.input_audio_transcription.completed":
		text := ev.Transcript
		if text == "" {
			text = deltas[ev.ItemID]
		}
		delete(deltas, ev.ItemID)
		if text == "" {
			return StreamResult{}, false
		}
		return StreamResult{
			Text:    text,
			IsFinal: true,
		}, true
	}
	return StreamResult{}, false
}

// WithOpenAIRealtimeVADSilenceMs sets the server VAD silence threshold (ms)
// after which OpenAI Realtime emits a completed transcription event.
func WithOpenAIRealtimeVADSilenceMs(ms int) Option {
	return func(options *Options) {
		options.EndpointingMs = &ms
	}
}
