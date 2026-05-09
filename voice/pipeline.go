package voice

import (
	"context"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/types"
)

// runAssistantTurn drives the assistant's response loop for one user message.
// It calls the LLM, speaks the streamed text via TTS, and runs any tool calls
// up to v.maxToolIterations times before yielding control back to the caller.
func runAssistantTurn(
	ctx context.Context,
	v *VoiceAgent,
	history *[]message.Message,
	emit func(Event),
	ttsAudio chan<- []byte,
) error {
	for i := 0; i < v.maxToolIterations; i++ {
		text, toolCalls, err := streamLLMAndSpeak(ctx, v, history, emit, ttsAudio)
		if err != nil {
			return err
		}

		if len(toolCalls) == 0 {
			emit(Event{Type: EventAssistantDone, Timestamp: time.Now(), Text: text})
			return nil
		}

		appendAssistantToolCalls(history, text, toolCalls)
		if err := runToolsWithSound(
			ctx, v, text, toolCalls, history, emit, ttsAudio,
		); err != nil {
			return err
		}
	}

	emit(Event{Type: EventAssistantDone, Timestamp: time.Now()})
	return nil
}

// streamLLMAndSpeak runs the LLM stream and concurrently feeds completed
// sentences to TTS as they arrive. TTS is opened lazily on the first content
// delta so iterations that go straight to a tool call never dial TTS.
//
// If the TTS client implements tts.StreamingTextProvider the audio path runs
// end-to-end concurrent with the LLM (low time-to-first-byte). Otherwise the
// function buffers the full LLM text and falls back to single-shot
// tts.Generation.StreamAudio.
func streamLLMAndSpeak(
	ctx context.Context,
	v *VoiceAgent,
	history *[]message.Message,
	emit func(Event),
	ttsAudio chan<- []byte,
) (string, []message.ToolCall, error) {
	stp, supportsStreaming := v.tts.(tts.StreamingTextProvider)

	events := v.llm.StreamResponse(ctx, *history, v.tools)

	var (
		buf             strings.Builder
		toolCalls       []message.ToolCall
		chunker         sentenceChunker
		textIn          chan string
		audioOut        <-chan tts.Chunk
		audioPumpDone   chan struct{}
		ttsErr          error
		gotFirstContent bool
		fillerFired     bool
	)

	startTTS := func() error {
		if textIn != nil {
			return nil
		}
		ch := make(chan string, 16)
		out, err := stp.StreamAudioFromText(ctx, ch)
		if err != nil {
			close(ch)
			return err
		}
		emit(Event{Type: EventTTSStarted, Timestamp: time.Now()})
		textIn = ch
		audioOut = out
		audioPumpDone = make(chan struct{})
		go func() {
			defer close(audioPumpDone)
			for chunk := range audioOut {
				if chunk.Error != nil {
					ttsErr = chunk.Error
					continue
				}
				if len(chunk.Data) == 0 {
					continue
				}
				select {
				case ttsAudio <- chunk.Data:
				case <-ctx.Done():
					return
				}
			}
		}()
		return nil
	}

	flushSentence := func(piece string) {
		if textIn == nil || piece == "" {
			return
		}
		select {
		case textIn <- piece:
		case <-ctx.Done():
		}
	}

	closeTTS := func() {
		if textIn == nil {
			return
		}
		if rem := chunker.flushRemainder(); rem != "" {
			select {
			case textIn <- rem:
			case <-ctx.Done():
			}
		}
		close(textIn)
		textIn = nil
		<-audioPumpDone
		emit(Event{Type: EventTTSEnded, Timestamp: time.Now()})
	}

	queueFillerText := func(text string) {
		if !supportsStreaming || text == "" {
			return
		}
		if err := startTTS(); err != nil {
			return
		}
		select {
		case textIn <- text:
			emit(Event{
				Type:      EventFiller,
				Timestamp: time.Now(),
				Text:      text,
			})
		case <-ctx.Done():
		}
	}

	fillerTimerCh := make(chan struct{}, 1)
	fillerTextCh := make(chan string, 1)
	var fillerTimer *time.Timer
	if v.filler.Timeout > 0 &&
		(v.filler.Message != "" || v.filler.Source != nil) {
		fillerTimer = time.AfterFunc(v.filler.Timeout, func() {
			select {
			case fillerTimerCh <- struct{}{}:
			default:
			}
		})
		defer fillerTimer.Stop()
	}

	streamDone := false
	for !streamDone {
		select {
		case <-ctx.Done():
			closeTTS()
			return buf.String(), toolCalls, ctx.Err()

		case ev, ok := <-events:
			if !ok {
				streamDone = true
				continue
			}
			switch ev.Type {
			case types.EventContentDelta:
				if ev.Content == "" {
					continue
				}
				if !gotFirstContent {
					gotFirstContent = true
					if fillerTimer != nil {
						fillerTimer.Stop()
					}
				}
				buf.WriteString(ev.Content)
				emit(Event{
					Type:      EventAssistantDelta,
					Timestamp: time.Now(),
					Text:      ev.Content,
				})
				if !supportsStreaming {
					continue
				}
				if err := startTTS(); err != nil {
					supportsStreaming = false
					continue
				}
				for _, sentence := range chunker.push(ev.Content) {
					flushSentence(sentence)
				}
			case types.EventComplete:
				if ev.Response != nil && len(ev.Response.ToolCalls) > 0 {
					toolCalls = ev.Response.ToolCalls
				}
			case types.EventError:
				if ev.Error != nil {
					closeTTS()
					return "", nil, ev.Error
				}
			}

		case <-fillerTimerCh:
			if gotFirstContent || fillerFired {
				continue
			}
			if v.filler.Source == nil {
				queueFillerText(v.filler.Message)
				fillerFired = true
				continue
			}
			deadline := v.filler.SourceDeadline
			if deadline <= 0 {
				deadline = defaultSourceDeadline
			}
			snapshot := append([]message.Message(nil), *history...)
			fallback := v.filler.Message
			source := v.filler.Source
			go func() {
				sourceCtx, cancel := context.WithTimeout(ctx, deadline)
				defer cancel()
				text := fallback
				if t, err := source(sourceCtx, snapshot); err == nil &&
					t != "" {
					text = t
				}
				select {
				case fillerTextCh <- text:
				default:
				}
			}()

		case msg := <-fillerTextCh:
			if gotFirstContent || fillerFired || msg == "" {
				continue
			}
			queueFillerText(msg)
			fillerFired = true
		}
	}

	text := buf.String()

	if textIn != nil {
		closeTTS()
		if ttsErr != nil {
			return text, toolCalls, ttsErr
		}
		return text, toolCalls, nil
	}

	if len(toolCalls) == 0 && strings.TrimSpace(text) != "" {
		if err := speakOneShot(ctx, v.tts, strings.TrimSpace(text), emit, ttsAudio); err != nil {
			return text, toolCalls, err
		}
	}
	return text, toolCalls, nil
}

// speakOneShot is the fallback path for TTS providers that do not implement
// tts.StreamingTextProvider. The LLM text is buffered to completion and sent
// via tts.Generation.StreamAudio.
func speakOneShot(
	ctx context.Context,
	ttsClient tts.Generation,
	text string,
	emit func(Event),
	ttsAudio chan<- []byte,
) error {
	emit(Event{Type: EventTTSStarted, Timestamp: time.Now()})
	chunks, err := ttsClient.StreamAudio(ctx, text)
	if err != nil {
		emit(Event{Type: EventTTSEnded, Timestamp: time.Now()})
		return err
	}
	for chunk := range chunks {
		if chunk.Error != nil {
			emit(Event{Type: EventTTSEnded, Timestamp: time.Now()})
			return chunk.Error
		}
		if len(chunk.Data) == 0 {
			continue
		}
		select {
		case ttsAudio <- chunk.Data:
		case <-ctx.Done():
			emit(Event{Type: EventTTSEnded, Timestamp: time.Now()})
			return ctx.Err()
		}
	}
	emit(Event{Type: EventTTSEnded, Timestamp: time.Now()})
	return nil
}

func appendAssistantToolCalls(
	history *[]message.Message,
	text string,
	calls []message.ToolCall,
) {
	parts := make([]message.ContentPart, 0, len(calls)+1)
	if t := strings.TrimSpace(text); t != "" {
		parts = append(parts, message.TextContent{Text: t})
	}
	for _, c := range calls {
		parts = append(parts, c)
	}
	*history = append(*history, message.NewMessage(message.Assistant, parts))
}

// llm.Event is referenced indirectly via the channel returned by
// llm.LLM.StreamResponse. The compile-time check below documents the field
// shape we rely on.
var _ = func(e llm.Event) types.EventType { return e.Type }
