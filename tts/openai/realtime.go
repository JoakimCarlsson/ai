package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joakimcarlsson/ai/tts"
)

const (
	defaultRealtimeBaseURL  = "https://api.openai.com/v1"
	realtimeModelPrefix     = "gpt-realtime"
	realtimeHandshakeTimout = 10 * time.Second
	realtimeReadDeadline    = 60 * time.Second
	realtimeReadLimit       = 16 * 1024 * 1024
)

// realtimeFormats maps the output formats the Realtime API can produce to its
// format objects and the content type reported for them.
var realtimeFormats = map[string]struct {
	format      realtimeAudioFormat
	contentType string
}{
	"pcm":  {realtimeAudioFormat{Type: "audio/pcm", Rate: 24000}, "audio/pcm"},
	"pcmu": {realtimeAudioFormat{Type: "audio/pcmu"}, "audio/basic"},
	"pcma": {realtimeAudioFormat{Type: "audio/pcma"}, "audio/x-alaw-basic"},
}

// realtimeAudioFormat is a Realtime API audio format object.
type realtimeAudioFormat struct {
	Type string `json:"type"`
	Rate int    `json:"rate,omitempty"`
}

// realtimeVoice is the voice of a realtime session: a built-in name, or a
// custom voice by ID.
type realtimeVoice struct {
	name string
	id   string
}

// MarshalJSON encodes a built-in voice as its name and a custom voice as
// {"id": ...}, as the Realtime API expects.
func (v realtimeVoice) MarshalJSON() ([]byte, error) {
	if v.id != "" {
		return json.Marshal(map[string]string{"id": v.id})
	}
	return json.Marshal(v.name)
}

// realtimeSessionUpdate is the session.update event that configures the
// session for audio-only output.
type realtimeSessionUpdate struct {
	Type    string `json:"type"`
	Session struct {
		Type             string   `json:"type"`
		OutputModalities []string `json:"output_modalities"`
		Audio            struct {
			Output struct {
				Format realtimeAudioFormat `json:"format"`
				Voice  *realtimeVoice      `json:"voice,omitempty"`
				Speed  *float64            `json:"speed,omitempty"`
			} `json:"output"`
		} `json:"audio"`
	} `json:"session"`
}

// realtimeResponseCreate is the response.create event that asks the model to
// speak, with no conversation context.
type realtimeResponseCreate struct {
	Type     string `json:"type"`
	Response struct {
		Input            []any    `json:"input"`
		OutputModalities []string `json:"output_modalities"`
		Instructions     string   `json:"instructions"`
	} `json:"response"`
}

// realtimeServerEvent holds the fields of the server events the speech path
// reads.
type realtimeServerEvent struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Response struct {
		Status        string `json:"status"`
		StatusDetails *struct {
			Reason string `json:"reason"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"status_details"`
	} `json:"response"`
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// usesRealtime reports whether the configured model is a Realtime model,
// which OpenAI serves over the Realtime WebSocket rather than /audio/speech.
func (c *Client) usesRealtime() bool {
	return strings.HasPrefix(c.options.model.APIModel, realtimeModelPrefix)
}

// realtimeFormat returns the format for outputFormat ("" means pcm) and its
// content type. It fails for formats the Realtime API cannot produce.
func realtimeFormat(outputFormat string) (realtimeAudioFormat, string, error) {
	if outputFormat == "" {
		outputFormat = "pcm"
	}
	f, ok := realtimeFormats[outputFormat]
	if !ok {
		return realtimeAudioFormat{}, "", fmt.Errorf(
			"realtime speech supports pcm, pcmu and pcma output, not %q",
			outputFormat,
		)
	}
	return f.format, f.contentType, nil
}

// speechInstructions builds the response instructions that make the model
// say text word for word, with the configured delivery instructions first.
func (c *Client) speechInstructions(text string) string {
	var b strings.Builder
	if c.options.instructions != "" {
		b.WriteString(c.options.instructions)
		b.WriteString("\n\n")
	}
	b.WriteString("Say exactly the following text, word for word, ")
	b.WriteString("and nothing else:\n")
	b.WriteString(text)
	return b.String()
}

// realtimeURL returns the Realtime WebSocket address for the configured
// model, derived from the base URL.
func (c *Client) realtimeURL() (string, error) {
	baseURL := c.options.baseURL
	if baseURL == "" {
		baseURL = defaultRealtimeBaseURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if base.Scheme == "http" {
		scheme = "ws"
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     base.Host,
		Path:     strings.TrimSuffix(base.Path, "/") + "/realtime",
		RawQuery: url.Values{"model": {c.options.model.APIModel}}.Encode(),
	}
	return u.String(), nil
}

// streamRealtime opens a Realtime session, configures audio-only output in
// the chosen voice and format, asks the model to say text verbatim, and
// emits the audio deltas as chunks until response.done. It returns the
// content type of the audio. It fails when the format is unsupported or the
// session cannot be set up; later failures arrive as chunk errors.
func (c *Client) streamRealtime(
	ctx context.Context,
	text string,
	opts tts.GenerationOptions,
) (<-chan tts.Chunk, string, error) {
	outputFormat := c.options.outputFormat
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	format, contentType, err := realtimeFormat(outputFormat)
	if err != nil {
		return nil, "", err
	}

	wsURL, err := c.realtimeURL()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build realtime url: %w", err)
	}
	hdr := http.Header{}
	apiKey := c.options.apiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	hdr.Set("Authorization", "Bearer "+apiKey)
	dialer := websocket.Dialer{HandshakeTimeout: realtimeHandshakeTimout}
	conn, resp, err := dialer.DialContext(ctx, wsURL, hdr)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to dial realtime websocket: %w", err)
	}
	conn.SetReadLimit(realtimeReadLimit)

	var update realtimeSessionUpdate
	update.Type = "session.update"
	update.Session.Type = "realtime"
	update.Session.OutputModalities = []string{"audio"}
	update.Session.Audio.Output.Format = format
	update.Session.Audio.Output.Speed = c.options.speed
	switch {
	case c.options.customVoiceID != "":
		update.Session.Audio.Output.Voice = &realtimeVoice{id: c.options.customVoiceID}
	case c.options.voice != "":
		update.Session.Audio.Output.Voice = &realtimeVoice{name: c.options.voice}
	}

	var create realtimeResponseCreate
	create.Type = "response.create"
	create.Response.Input = []any{}
	create.Response.OutputModalities = []string{"audio"}
	create.Response.Instructions = c.speechInstructions(text)

	for _, event := range []any{update, create} {
		if err := conn.WriteJSON(event); err != nil {
			_ = conn.Close()
			return nil, "", fmt.Errorf("failed to send realtime event: %w", err)
		}
	}

	out := make(chan tts.Chunk)
	go readRealtime(ctx, conn, out)
	return out, contentType, nil
}

// readRealtime decodes Realtime server events from conn into chunks on out
// until response.done, an error event, a read failure or ctx is done, then
// closes out and conn.
func readRealtime(
	ctx context.Context,
	conn *websocket.Conn,
	out chan<- tts.Chunk,
) {
	defer close(out)
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	emit := func(chunk tts.Chunk) bool {
		select {
		case out <- chunk:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(realtimeReadDeadline))
		var event realtimeServerEvent
		if err := conn.ReadJSON(&event); err != nil {
			if ctx.Err() == nil && !errors.Is(err, net.ErrClosed) {
				emit(tts.Chunk{Error: fmt.Errorf("realtime read failed: %w", err)})
			}
			return
		}
		switch event.Type {
		case "response.output_audio.delta":
			data, err := base64.StdEncoding.DecodeString(event.Delta)
			if err != nil {
				emit(tts.Chunk{Error: fmt.Errorf("failed to decode realtime audio: %w", err)})
				return
			}
			if !emit(tts.Chunk{Data: data}) {
				return
			}
		case "error":
			emit(tts.Chunk{Error: fmt.Errorf(
				"realtime error %s: %s", event.Error.Code, event.Error.Message,
			)})
			return
		case "response.done":
			if event.Response.Status != "completed" {
				emit(tts.Chunk{Error: fmt.Errorf(
					"realtime response %s%s",
					event.Response.Status, statusDetail(event),
				)})
				return
			}
			emit(tts.Chunk{Done: true})
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(2*time.Second),
			)
			return
		}
	}
}

// statusDetail formats the reason and error message of a response that did
// not complete, or "" when the event carries none.
func statusDetail(event realtimeServerEvent) string {
	d := event.Response.StatusDetails
	if d == nil {
		return ""
	}
	detail := ": " + d.Reason
	if d.Error != nil && d.Error.Message != "" {
		detail += ": " + d.Error.Message
	}
	return detail
}

// generateRealtime speaks text through the Realtime API and returns the
// whole audio once the response is done.
func (c *Client) generateRealtime(
	ctx context.Context,
	text string,
	opts tts.GenerationOptions,
) (*tts.Response, error) {
	chunks, contentType, err := c.streamRealtime(ctx, text, opts)
	if err != nil {
		return nil, err
	}
	var audio []byte
	var streamErr error
	for chunk := range chunks {
		if chunk.Error != nil && streamErr == nil {
			streamErr = chunk.Error
		}
		audio = append(audio, chunk.Data...)
	}
	if streamErr != nil {
		return nil, streamErr
	}
	return &tts.Response{
		AudioData:   audio,
		ContentType: contentType,
		Usage:       tts.Usage{Characters: int64(len(text))},
		Model:       c.options.model.APIModel,
	}, nil
}
