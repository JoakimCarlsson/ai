package voice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tts"
	"github.com/joakimcarlsson/ai/types"
)

const defaultReserveTokens int64 = 4096

// runAssistantTurn drives the assistant's response loop for one user message.
// It calls the LLM, speaks the streamed text via TTS, and runs any tool calls
// up to v.maxToolIterations times before yielding control back to the caller.
//
// Returns the active agent at the end of the turn — possibly different from
// the input agent if a handoff fired during this turn. The caller continues
// subsequent user turns with the returned agent.
func runAssistantTurn(
	ctx context.Context,
	v *Agent,
	history *[]message.Message,
	emit func(Event),
	ttsAudio chan<- []byte,
	state *turnState,
) (*Agent, error) {
	active := v
	i := 0
	for i < active.maxToolIterations {
		text, toolCalls, err := streamLLMAndSpeak(
			ctx,
			active,
			history,
			emit,
			ttsAudio,
			state,
		)
		if err != nil {
			return active, err
		}

		if len(toolCalls) == 0 {
			if t := strings.TrimSpace(text); t != "" {
				*history = append(*history, message.NewMessage(
					message.Assistant,
					[]message.ContentPart{message.TextContent{Text: t}},
				))
			}
			emit(
				Event{
					Type:      EventAssistantDone,
					Timestamp: time.Now(),
					Text:      text,
				},
			)
			return active, nil
		}

		appendAssistantToolCalls(history, text, toolCalls)
		if err := runToolsWithSound(
			ctx, active, text, toolCalls, history, emit, ttsAudio,
		); err != nil {
			return active, err
		}

		if h := detectHandoff(toolCalls, active.handoffs); h != nil {
			active = h.Agent
			*history = rebuildMessagesForHandoff(active, *history)
			i = 0
			continue
		}

		i++
	}

	emit(Event{Type: EventAssistantDone, Timestamp: time.Now()})
	return active, nil
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
	v *Agent,
	history *[]message.Message,
	emit func(Event),
	ttsAudio chan<- []byte,
	state *turnState,
) (string, []message.ToolCall, error) {
	stp, supportsStreaming := v.tts.(tts.StreamingTextProvider)

	llmMessages, err := applyContextStrategy(ctx, v, history)
	if err != nil {
		return "", nil, err
	}

	if state != nil && v.memory != nil && v.memoryID != "" {
		if !state.memorySearched.Load() {
			state.memorySearched.Store(true)
			recallCtx := v.recallMemoriesContext(ctx, lastUserText(*history), 5)
			if recallCtx != "" {
				state.memoryContext.Store(&recallCtx)
			}
		}
		if p := state.memoryContext.Load(); p != nil && *p != "" {
			memMsg := message.NewSystemMessage(*p)
			llmMessages = append([]message.Message{memMsg}, llmMessages...)
		}
	}

	if state != nil && v.kb != nil {
		if !state.kbSearched.Load() {
			state.kbSearched.Store(true)
			recallCtx := v.recallKBContext(ctx, lastUserText(*history), 5)
			if recallCtx != "" {
				state.kbContext.Store(&recallCtx)
			}
		}
		if p := state.kbContext.Load(); p != nil && *p != "" {
			kbMsg := message.NewSystemMessage(*p)
			llmMessages = append([]message.Message{kbMsg}, llmMessages...)
		}
	}

	llmTools := v.toolsForContext(ctx)
	if len(v.hooks) > 0 {
		hookRes, err := runPreModelCall(ctx, v.hooks, ModelCallContext{
			ConversationID: conversationIDFromCtx(ctx),
			Messages:       llmMessages,
			Tools:          llmTools,
		})
		if err != nil {
			return "", nil, err
		}
		if hookRes.Action == HookModify {
			llmMessages = hookRes.Messages
			llmTools = hookRes.Tools
		}
	}

	llmStarted := time.Now()
	events := v.llm.StreamResponse(ctx, llmMessages, llmTools)

	postModelCallFired := false
	firePostModelCall := func(callErr error) {
		if postModelCallFired || len(v.hooks) == 0 {
			return
		}
		postModelCallFired = true
		runPostModelCall(ctx, v.hooks, ModelResponseContext{
			ConversationID: conversationIDFromCtx(ctx),
			Duration:       time.Since(llmStarted),
			Error:          callErr,
		})
	}
	defer firePostModelCall(nil)

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
		if state != nil {
			state.agentSpeaking.Store(true)
		}
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
		if state != nil {
			state.agentSpeaking.Store(false)
		}
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
			firePostModelCall(ctx.Err())
			closeTTS()
			return buf.String(), toolCalls, ctx.Err()

		case ev, ok := <-events:
			if !ok {
				firePostModelCall(nil)
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
				if state != nil {
					state.setSpoken(buf.String())
				}
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
					firePostModelCall(ev.Error)
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

// applyContextStrategy runs the configured context-window strategy (if any)
// over the current history and returns the message slice that should be
// passed to the LLM. When the strategy emits a SessionUpdate the appended
// messages (typically a summary) are folded into the live history so
// subsequent turns start from the trimmed state. The runner's per-turn
// persist step then writes them to the session along with the rest of the
// turn's new messages.
func applyContextStrategy(
	ctx context.Context,
	v *Agent,
	history *[]message.Message,
) ([]message.Message, error) {
	if v.contextStrategy == nil {
		return *history, nil
	}
	maxTokens := resolveMaxTokens(v)
	if maxTokens <= 0 {
		return *history, nil
	}
	counter, err := tokens.NewCounter()
	if err != nil {
		return nil, fmt.Errorf("token counter: %w", err)
	}
	result, err := v.contextStrategy.Fit(ctx, tokens.StrategyInput{
		Messages:     *history,
		SystemPrompt: v.systemPrompt,
		Tools:        v.tools,
		Counter:      counter,
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("context strategy: %w", err)
	}
	if result == nil {
		return *history, nil
	}
	if result.SessionUpdate != nil &&
		len(result.SessionUpdate.AddMessages) > 0 {
		*history = append(*history, result.SessionUpdate.AddMessages...)
	}
	return result.Messages, nil
}

func resolveMaxTokens(v *Agent) int64 {
	if v.maxContextTokens > 0 {
		return v.maxContextTokens
	}
	cw := v.llm.Model().ContextWindow
	if cw <= 0 {
		return 0
	}
	if cw <= defaultReserveTokens {
		return cw
	}
	return cw - defaultReserveTokens
}

// llm.Event is referenced indirectly via the channel returned by
// llm.LLM.StreamResponse. The compile-time check below documents the field
// shape we rely on.
var _ = func(e llm.Event) types.EventType { return e.Type }
