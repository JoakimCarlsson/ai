package transcription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	elevenLabsStreamDefaultVADSilenceMs = 700
	elevenLabsStreamReadDeadline        = 30 * time.Second
	elevenLabsStreamHandshakeTimeout    = 10 * time.Second
)

// streamTranscribe opens an ElevenLabs Scribe v2 Realtime WebSocket. Audio
// frames are base64-encoded and wrapped in JSON input_audio_chunk events
// (per ElevenLabs' protocol — they don't accept raw binary). Reader parses
// partial_transcript / committed_transcript events into StreamResult.
func (e *elevenLabsClient) streamTranscribe(
	ctx context.Context,
	audio <-chan []byte,
	options ...Option,
) (<-chan StreamResult, error) {
	opts := Options{}
	for _, o := range options {
		o(&opts)
	}

	sampleRate := opts.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}
	vadSilence := elevenLabsStreamDefaultVADSilenceMs
	if opts.EndpointingMs != nil {
		vadSilence = *opts.EndpointingMs
	}
	lang := opts.Language

	q := url.Values{}
	q.Set("sample_rate", strconv.Itoa(sampleRate))
	if lang != "" {
		q.Set("language_code", lang)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.elevenlabs.io",
		Path:     "/v1/speech-to-text/realtime",
		RawQuery: q.Encode(),
	}
	hdr := http.Header{}
	hdr.Set("xi-api-key", e.providerOptions.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: elevenLabsStreamHandshakeTimeout}
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
		`{"type":"session.update","vad":{"min_silence_ms":%d}}`,
		vadSilence,
	)
	if err := send(websocket.TextMessage, []byte(cfg)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
	})

	go runElevenLabsReader(conn, out, done)
	go runElevenLabsWriter(ctx, conn, audio, out, done, send)

	return out, nil
}

func runElevenLabsReader(
	conn *websocket.Conn,
	out chan<- StreamResult,
	done chan<- struct{},
) {
	defer close(done)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !isCleanWSClose(err) {
				out <- StreamResult{Error: err}
			}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(elevenLabsStreamReadDeadline))
		pr, ok := parseElevenLabsStream(msg)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runElevenLabsWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- StreamResult,
	done <-chan struct{},
	send func(int, []byte) error,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	closeSession := func() {
		_ = send(websocket.TextMessage, []byte(`{"type":"session.close"}`))
	}

	audioOpen := true
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			closeSession()
			<-done
			return
		case frame, ok := <-audio:
			if !ok {
				if audioOpen {
					closeSession()
					audioOpen = false
				}
				audio = nil
				continue
			}
			payload := fmt.Sprintf(
				`{"type":"input_audio_chunk","audio":"%s"}`,
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

type elevenLabsStreamResp struct {
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
	Words      []struct {
		Text       string  `json:"text"`
		Start      float64 `json:"start"`
		End        float64 `json:"end"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
}

func parseElevenLabsStream(raw []byte) (StreamResult, bool) {
	var resp elevenLabsStreamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return StreamResult{}, false
	}
	isFinal := false
	switch resp.Type {
	case "partial_transcript":
		isFinal = false
	case "committed_transcript":
		isFinal = true
	default:
		return StreamResult{}, false
	}
	text := resp.Transcript
	if text == "" {
		text = resp.Text
	}
	if text == "" {
		return StreamResult{}, false
	}
	words := make([]Word, len(resp.Words))
	for i, w := range resp.Words {
		words[i] = Word{Word: w.Text, Start: w.Start, End: w.End}
	}
	return StreamResult{
		Text:       text,
		Confidence: resp.Confidence,
		IsFinal:    isFinal,
		WordCount:  len(resp.Words),
		Words:      words,
	}, true
}

// WithElevenLabsStreamVADSilenceMs sets the minimum silence (ms) ElevenLabs'
// VAD waits before emitting committed_transcript.
func WithElevenLabsStreamVADSilenceMs(ms int) Option {
	return func(options *Options) {
		options.EndpointingMs = &ms
	}
}
