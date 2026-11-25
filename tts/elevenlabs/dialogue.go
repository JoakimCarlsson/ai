package elevenlabs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/tts"
)

// dialogueOnlyModelPrefix names the models ElevenLabs serves only through the
// Text to Dialogue endpoints, not through /text-to-speech.
const dialogueOnlyModelPrefix = "eleven_v4"

// dialogueInput is one entry of the inputs list of a Text to Dialogue request
// or WebSocket frame.
type dialogueInput struct {
	Text    string `json:"text"`
	VoiceID string `json:"voice_id"`
	NewTurn bool   `json:"new_turn,omitempty"`
}

// dialogueRequest is the JSON body of POST /text-to-dialogue and
// /text-to-dialogue/with-timestamps.
type dialogueRequest struct {
	Inputs                 []dialogueInput `json:"inputs"`
	ModelID                string          `json:"model_id"`
	LanguageCode           string          `json:"language_code,omitempty"`
	Seed                   *uint32         `json:"seed,omitempty"`
	ApplyTextNormalization string          `json:"apply_text_normalization,omitempty"`
}

// dialogueWSStartMessage is the first frame of a Text to Dialogue WebSocket
// session, registering the voices it may use.
type dialogueWSStartMessage struct {
	Voices        []string       `json:"voices"`
	VoiceSettings *voiceSettings `json:"voice_settings,omitempty"`
}

// dialogueWSInputMessage carries text for registered voices over the Text to
// Dialogue WebSocket.
type dialogueWSInputMessage struct {
	Inputs []dialogueInput `json:"inputs"`
}

// usesDialogueEndpoint reports whether modelID must be sent to the Text to
// Dialogue endpoints instead of /text-to-speech.
func usesDialogueEndpoint(modelID string) bool {
	return strings.HasPrefix(modelID, dialogueOnlyModelPrefix)
}

// GenerateDialogue renders turns as one audio file through POST
// /text-to-dialogue, or /text-to-dialogue/with-timestamps when
// [tts.WithAlignmentEnabled] is set. It fails when turns is empty, a turn has
// no voice, or the API rejects the request. Implements [tts.DialogueProvider].
func (c *Client) GenerateDialogue(
	ctx context.Context,
	turns []tts.DialogueTurn,
	options ...tts.GenerationOption,
) (*tts.Response, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return c.generateDialogue(ctx, turns, opts)
}

// generateDialogue sends turns to the Text to Dialogue REST endpoint with the
// parsed opts and returns the audio, with alignment data when requested.
func (c *Client) generateDialogue(
	ctx context.Context,
	turns []tts.DialogueTurn,
	opts *tts.GenerationOptions,
) (*tts.Response, error) {
	if len(turns) == 0 {
		return nil, fmt.Errorf("dialogue needs at least one turn")
	}
	inputs := make([]dialogueInput, len(turns))
	chars := 0
	for i, turn := range turns {
		if turn.VoiceID == "" {
			return nil, fmt.Errorf("dialogue turn %d has no voice ID", i)
		}
		inputs[i] = dialogueInput{Text: turn.Text, VoiceID: turn.VoiceID}
		chars += len(turn.Text)
	}

	jsonData, err := json.Marshal(dialogueRequest{
		Inputs:                 inputs,
		ModelID:                c.modelID,
		LanguageCode:           opts.LanguageCode,
		Seed:                   opts.Seed,
		ApplyTextNormalization: opts.TextNormalization,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	outputFormat := c.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	path := "/text-to-dialogue"
	if opts.EnableAlignment {
		path += "/with-timestamps"
	}
	endpoint := c.baseURL + path + "?output_format=" + url.QueryEscape(outputFormat)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	usage := tts.Usage{Characters: int64(chars)}
	if count, err := strconv.ParseInt(
		resp.Header.Get("x-character-count"), 10, 64,
	); err == nil {
		usage.Characters = count
	}

	if !opts.EnableAlignment {
		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = contentTypeForFormat(outputFormat)
		}
		return &tts.Response{
			AudioData:   audioData,
			ContentType: contentType,
			Usage:       usage,
			Model:       c.modelID,
		}, nil
	}

	var timestampsResp ttsWithTimestampsResponse
	if err := json.NewDecoder(resp.Body).Decode(&timestampsResp); err != nil {
		return nil, fmt.Errorf("failed to decode timestamps response: %w", err)
	}
	audioData, err := base64.StdEncoding.DecodeString(timestampsResp.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 audio: %w", err)
	}
	return &tts.Response{
		AudioData:           audioData,
		ContentType:         contentTypeForFormat(outputFormat),
		Usage:               usage,
		Model:               c.modelID,
		Alignment:           toAlignmentData(timestampsResp.Alignment),
		NormalizedAlignment: toAlignmentData(timestampsResp.NormalizedAlignment),
	}, nil
}

// StreamDialogue opens the Text to Dialogue WebSocket, registers voiceIDs,
// sends each turn from turns as it arrives and emits audio chunks as the
// server produces them. Closing turns, or cancelling ctx, sends close_socket so
// the server flushes the remaining audio. It fails when voiceIDs is empty or
// the connection cannot be opened; later failures arrive as chunk errors.
// Implements [tts.DialogueProvider].
func (c *Client) StreamDialogue(
	ctx context.Context,
	voiceIDs []string,
	turns <-chan tts.DialogueTurn,
	options ...tts.GenerationOption,
) (<-chan tts.Chunk, error) {
	opts := &tts.GenerationOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return c.streamDialogue(ctx, voiceIDs, turns, opts)
}

// streamDialogue is StreamDialogue with already parsed opts.
func (c *Client) streamDialogue(
	ctx context.Context,
	voiceIDs []string,
	turns <-chan tts.DialogueTurn,
	opts *tts.GenerationOptions,
) (<-chan tts.Chunk, error) {
	if len(voiceIDs) == 0 {
		return nil, fmt.Errorf("dialogue stream needs at least one voice ID")
	}

	conn, send, err := c.dialDialogueWS(ctx, voiceIDs, opts)
	if err != nil {
		return nil, err
	}

	chunkChan := make(chan tts.Chunk, 10)
	go pumpDialogueTurns(ctx, turns, send, chunkChan)
	go runStreamReader(ctx, conn, chunkChan)
	return chunkChan, nil
}

// pumpDialogueTurns sends every non-empty turn from turns as an inputs frame
// and sends close_socket once turns is closed or ctx is done. A failure to
// encode or send is reported on out and ends the pump.
func pumpDialogueTurns(
	ctx context.Context,
	turns <-chan tts.DialogueTurn,
	send func(int, []byte) error,
	out chan<- tts.Chunk,
) {
	closeSocket := []byte(`{"close_socket":true}`)
	for {
		select {
		case <-ctx.Done():
			_ = send(websocket.TextMessage, closeSocket)
			return
		case turn, ok := <-turns:
			if !ok {
				_ = send(websocket.TextMessage, closeSocket)
				return
			}
			if turn.Text == "" {
				continue
			}
			msg, err := json.Marshal(dialogueWSInputMessage{
				Inputs: []dialogueInput{{
					Text:    turn.Text,
					VoiceID: turn.VoiceID,
					NewTurn: turn.NewTurn,
				}},
			})
			if err == nil {
				err = send(websocket.TextMessage, msg)
			}
			if err != nil {
				select {
				case out <- tts.Chunk{Error: fmt.Errorf("failed to send dialogue turn: %w", err)}:
				case <-ctx.Done():
				}
				_ = send(websocket.TextMessage, closeSocket)
				return
			}
		}
	}
}

// dialDialogueWS opens the Text to Dialogue WebSocket, sends the frame that
// registers voiceIDs, and returns the connection with a goroutine-safe send
// function. It fails when the URL is invalid, the handshake fails or the first
// frame cannot be sent.
func (c *Client) dialDialogueWS(
	ctx context.Context,
	voiceIDs []string,
	opts *tts.GenerationOptions,
) (*websocket.Conn, func(int, []byte) error, error) {
	wsURL, err := c.buildDialogueStreamURL(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build ws url: %w", err)
	}

	hdr := http.Header{}
	hdr.Set("xi-api-key", c.apiKey)

	dialer := websocket.Dialer{HandshakeTimeout: wsHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, wsURL, hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial websocket: %w", err)
	}

	conn.SetReadLimit(wsReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	})

	var writeMu sync.Mutex
	send := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	start, err := json.Marshal(dialogueWSStartMessage{
		Voices:        voiceIDs,
		VoiceSettings: c.buildVoiceSettings(opts),
	})
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to marshal voices: %w", err)
	}
	if err := send(websocket.TextMessage, start); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to register voices: %w", err)
	}
	return conn, send, nil
}

// buildDialogueStreamURL returns the wss:// (or ws:// for an http base URL)
// address of the Text to Dialogue WebSocket with the model, output format and
// the request options of opts as query parameters.
func (c *Client) buildDialogueStreamURL(
	opts *tts.GenerationOptions,
) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if base.Scheme == "http" {
		scheme = "ws"
	}
	outputFormat := c.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	q := url.Values{}
	q.Set("model_id", c.modelID)
	if outputFormat != "" {
		q.Set("output_format", outputFormat)
	}
	if opts.EnableAlignment {
		q.Set("sync_alignment", "true")
	}
	if opts.LanguageCode != "" {
		q.Set("language_code", opts.LanguageCode)
	}
	if opts.Seed != nil {
		q.Set("seed", strconv.FormatUint(uint64(*opts.Seed), 10))
	}
	if opts.TextNormalization != "" {
		q.Set("apply_text_normalization", opts.TextNormalization)
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     base.Host,
		Path:     base.Path + "/text-to-dialogue/stream-input",
		RawQuery: q.Encode(),
	}
	return u.String(), nil
}
