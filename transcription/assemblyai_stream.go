package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	assemblyAIStreamDefaultEndOfTurnSilenceMs = 700
	assemblyAIStreamReadDeadline              = 30 * time.Second
	assemblyAIStreamHandshakeTimeout          = 10 * time.Second
	assemblyAIStreamMinChunkMs                = 100
)

// streamTranscribe opens an AssemblyAI v3 Universal-Streaming WebSocket.
// Reader parses Turn frames into StreamResult; writer forwards audio frames
// as binary messages. Auth is via query-param token (v3 convention).
func (a *assemblyAIClient) streamTranscribe(
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
	endOfTurn := assemblyAIStreamDefaultEndOfTurnSilenceMs
	if a.options.streamEndOfTurnSilenceMs != nil {
		endOfTurn = *a.options.streamEndOfTurnSilenceMs
	}

	speechModel := a.options.streamSpeechModel
	if speechModel == "" {
		speechModel = "universal-streaming-english"
	}
	formatTurns := true
	if a.options.streamFormatTurns != nil {
		formatTurns = *a.options.streamFormatTurns
	}
	q := url.Values{}
	q.Set("token", a.providerOptions.apiKey)
	q.Set("sample_rate", strconv.Itoa(sampleRate))
	q.Set("encoding", "pcm_s16le")
	q.Set("format_turns", strconv.FormatBool(formatTurns))
	q.Set("speech_model", speechModel)
	q.Set("min_end_of_turn_silence_when_confident", strconv.Itoa(endOfTurn))
	if a.options.streamEndOfTurnConfidenceThreshold != nil {
		q.Set("end_of_turn_confidence_threshold",
			strconv.FormatFloat(*a.options.streamEndOfTurnConfidenceThreshold, 'f', -1, 64))
	}
	if a.options.streamMaxTurnSilence != nil {
		q.Set("max_turn_silence", strconv.Itoa(*a.options.streamMaxTurnSilence))
	}
	if a.options.streamPunctuationFilter != nil {
		q.Set("punctuation_filter", strconv.FormatBool(*a.options.streamPunctuationFilter))
	}
	if a.options.streamWordFinalizationMaxWaitMs != nil {
		q.Set("word_finalization_max_wait_time",
			strconv.Itoa(*a.options.streamWordFinalizationMaxWaitMs))
	}
	if a.options.streamExtraSessionInformation != nil {
		q.Set("enable_extra_session_information",
			strconv.FormatBool(*a.options.streamExtraSessionInformation))
	}
	for _, kt := range a.options.streamKeyterms {
		q.Add("keyterms_prompt", kt)
	}

	u := url.URL{
		Scheme:   "wss",
		Host:     "streaming.assemblyai.com",
		Path:     "/v3/ws",
		RawQuery: q.Encode(),
	}

	dialer := websocket.Dialer{HandshakeTimeout: assemblyAIStreamHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, u.String(), nil)
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
		`{"type":"UpdateConfiguration","end_of_turn_confidence_threshold":0.7,"min_end_of_turn_silence_when_confident":%d}`,
		endOfTurn,
	)
	if err := send(websocket.TextMessage, []byte(cfg)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(assemblyAIStreamReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(assemblyAIStreamReadDeadline))
	})

	minChunkBytes := sampleRate * 2 * assemblyAIStreamMinChunkMs / 1000

	go runAssemblyAIReader(conn, out, done)
	go runAssemblyAIWriter(ctx, conn, audio, out, done, send, minChunkBytes)

	return out, nil
}

func runAssemblyAIReader(
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
		_ = conn.SetReadDeadline(time.Now().Add(assemblyAIStreamReadDeadline))
		pr, ok := parseAssemblyAIStream(msg)
		if !ok {
			continue
		}
		out <- pr
	}
}

func runAssemblyAIWriter(
	ctx context.Context,
	conn *websocket.Conn,
	audio <-chan []byte,
	out chan<- StreamResult,
	done <-chan struct{},
	send func(int, []byte) error,
	minChunkBytes int,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 0, minChunkBytes*2)
	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		err := send(websocket.BinaryMessage, buf)
		buf = buf[:0]
		return err
	}

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			_ = flush()
			return
		case frame, ok := <-audio:
			if !ok {
				_ = flush()
				return
			}
			buf = append(buf, frame...)
			if len(buf) >= minChunkBytes {
				if err := flush(); err != nil {
					out <- StreamResult{Error: err}
					_ = conn.Close()
					<-done
					return
				}
			}
		}
	}
}

type assemblyAIStreamResp struct {
	Type            string  `json:"type"`
	Transcript      string  `json:"transcript"`
	EndOfTurn       bool    `json:"end_of_turn"`
	TurnIsFormatted bool    `json:"turn_is_formatted"`
	EndOfTurnConf   float64 `json:"end_of_turn_confidence"`
	Words           []struct {
		Text        string  `json:"text"`
		Start       int64   `json:"start"`
		End         int64   `json:"end"`
		Confidence  float64 `json:"confidence"`
		WordIsFinal bool    `json:"word_is_final"`
	} `json:"words"`
}

func parseAssemblyAIStream(raw []byte) (StreamResult, bool) {
	var resp assemblyAIStreamResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return StreamResult{}, false
	}
	if resp.Type != "Turn" {
		return StreamResult{}, false
	}
	if resp.Transcript == "" {
		return StreamResult{}, false
	}
	words := make([]Word, len(resp.Words))
	for i, w := range resp.Words {
		words[i] = Word{
			Word:  w.Text,
			Start: float64(w.Start) / 1000.0,
			End:   float64(w.End) / 1000.0,
		}
	}
	conf := resp.EndOfTurnConf
	if conf == 0 && len(resp.Words) > 0 {
		conf = resp.Words[len(resp.Words)-1].Confidence
	}
	return StreamResult{
		Text:       resp.Transcript,
		Confidence: conf,
		IsFinal:    resp.EndOfTurn,
		WordCount:  len(resp.Words),
		Words:      words,
	}, true
}
