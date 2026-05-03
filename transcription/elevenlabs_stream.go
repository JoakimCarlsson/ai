package transcription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	elevenLabsStreamReadDeadline     = 30 * time.Second
	elevenLabsStreamHandshakeTimeout = 10 * time.Second
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
	lang := e.options.streamLanguageCode
	if lang == "" {
		lang = opts.Language
	}

	q := url.Values{}
	q.Set("audio_format", fmt.Sprintf("pcm_%d", sampleRate))
	if lang != "" {
		q.Set("language_code", lang)
	}
	for _, kt := range e.options.streamKeyterms {
		q.Add("keyterms", kt)
	}
	if e.options.streamNoVerbatim != nil {
		q.Set("no_verbatim", strconv.FormatBool(*e.options.streamNoVerbatim))
	}
	if e.options.streamIncludeTimestamps != nil {
		q.Set("include_timestamps", strconv.FormatBool(*e.options.streamIncludeTimestamps))
	}
	if e.options.streamIncludeLanguageDetect != nil {
		q.Set("include_language_detection",
			strconv.FormatBool(*e.options.streamIncludeLanguageDetect))
	}
	if e.options.streamVADThreshold != nil {
		q.Set("vad_threshold",
			strconv.FormatFloat(*e.options.streamVADThreshold, 'f', -1, 64))
	}
	if e.options.streamMinSpeechDurationMs != nil {
		q.Set("min_speech_duration_ms", strconv.Itoa(*e.options.streamMinSpeechDurationMs))
	}
	if e.options.streamMinSilenceDurationMs != nil {
		q.Set("min_silence_duration_ms", strconv.Itoa(*e.options.streamMinSilenceDurationMs))
	}
	if e.options.streamTimestampsGranularity != "" {
		q.Set("timestamps_granularity", e.options.streamTimestampsGranularity)
	}
	if e.options.streamDisableLogging != nil {
		q.Set("disable_logging", strconv.FormatBool(*e.options.streamDisableLogging))
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
			if !isCleanWSClose(err) && !errors.Is(err, net.ErrClosed) {
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

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			_ = conn.Close()
			<-done
			return
		case frame, ok := <-audio:
			if !ok {
				_ = conn.Close()
				<-done
				return
			}
			payload := fmt.Sprintf(
				`{"message_type":"input_audio_chunk","audio_base_64":"%s"}`,
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
	MessageType string  `json:"message_type"`
	Text        string  `json:"text"`
	Transcript  string  `json:"transcript"`
	Confidence  float64 `json:"confidence"`
	Words       []struct {
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
	switch resp.MessageType {
	case "partial_transcript":
		isFinal = false
	case "committed_transcript", "committed_transcript_with_timestamps":
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
