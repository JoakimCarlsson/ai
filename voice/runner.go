package voice

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/stt"
	"golang.org/x/sync/errgroup"
)

const (
	audioInBufferSize  = 64
	audioOutBufferSize = 64
	finalsBufferSize   = 8
)

// run drives a single conversation. It is invoked from a goroutine spawned by
// Agent.StartConversation and never returns until the conversation ends
// (ctx cancelled, transport closed, or unrecoverable error). The error is
// stored on the Conversation and surfaced via Wait; the events channel is
// closed before run returns.
func (c *Conversation) run(
	ctx context.Context,
	v *Agent,
	audio AudioTransport,
) {
	defer close(c.events)
	defer close(c.done)
	defer audio.Close()

	ctx = withConversationID(ctx, c.id)
	lifecycle := ConversationLifecycleContext{ConversationID: c.id}
	runOnConversationStart(ctx, v.hooks, lifecycle)
	defer runOnConversationEnd(ctx, v.hooks, lifecycle)

	emit := func(evt Event) {
		if evt.Timestamp.IsZero() {
			evt.Timestamp = time.Now()
		}
		select {
		case c.events <- evt:
		case <-ctx.Done():
		}
	}

	g, gctx := errgroup.WithContext(ctx)

	audioIn := make(chan []byte, audioInBufferSize)
	finals := make(chan string, finalsBufferSize)
	ttsAudio := make(chan []byte, audioOutBufferSize)

	state := c.state

	sttResults, err := v.stt.StreamTranscribe(gctx, audioIn)
	if err != nil {
		emit(Event{Type: EventError, Error: err})
		c.setErr(err)
		return
	}

	emit(Event{Type: EventReady})

	g.Go(func() error {
		defer close(audioIn)
		for {
			frame, err := audio.Read(gctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			if len(frame) == 0 {
				continue
			}
			select {
			case audioIn <- frame:
			case <-gctx.Done():
				return gctx.Err()
			}
		}
	})

	g.Go(func() error {
		var sawSpeechThisUtterance bool
		for r := range sttResults {
			if r.Error != nil {
				if errors.Is(r.Error, context.Canceled) {
					return nil
				}
				return r.Error
			}
			if r.Text == "" {
				continue
			}
			if !sawSpeechThisUtterance {
				emit(Event{Type: EventUserSpeechStart})
				sawSpeechThisUtterance = true
				if v.bargeIn == BargeInInterrupt && state.agentSpeaking.Load() {
					spoken := state.loadSpoken()
					state.dropAudio.Store(true)
					state.fireCancel()
					emit(Event{
						Type: EventAgentInterrupted,
						Text: spoken,
					})
				}
			}
			if r.IsFinal {
				emit(Event{Type: EventUserTranscriptFinal, Text: r.Text})
				sawSpeechThisUtterance = false

				text := r.Text
				if len(v.hooks) > 0 {
					hookRes, err := runOnUserMessage(
						gctx,
						v.hooks,
						UserMessageContext{
							ConversationID: c.id,
							Text:           r.Text,
						},
					)
					if err != nil {
						return err
					}
					if hookRes.Action == HookDeny {
						emit(
							Event{
								Type:  EventError,
								Error: errUserMessageDenied(hookRes.DenyReason),
							},
						)
						continue
					}
					if hookRes.Action == HookModify {
						text = hookRes.Text
					}
				}
				select {
				case finals <- text:
				case <-gctx.Done():
					return gctx.Err()
				}
				continue
			}
			emit(Event{Type: EventUserTranscriptPartial, Text: r.Text})
		}
		return nil
	})

	g.Go(func() error {
		history, sessionPersisted, err := loadInitialHistory(gctx, v)
		if err != nil {
			return err
		}

		persistNew := func() error {
			if v.session == nil || sessionPersisted >= len(history) {
				return nil
			}
			newMsgs := history[sessionPersisted:]
			if err := v.session.AddMessages(gctx, newMsgs); err != nil {
				return err
			}
			sessionPersisted = len(history)
			return nil
		}

		activeAgent := v

		if v.initialMessage != "" && !historyHasNonSystem(history) {
			drainAudio(ttsAudio)
			turnCtx, turnCancel := context.WithCancel(gctx)
			cancelFn := func() { turnCancel() }
			state.cancelTurn.Store(&cancelFn)
			state.setSpoken("")
			state.dropAudio.Store(false)
			state.agentSpeaking.Store(false)

			err := speakInitialMessage(
				turnCtx, v, v.initialMessage, emit, ttsAudio, state,
			)

			turnCancel()
			state.cancelTurn.Store(nil)

			switch {
			case errors.Is(err, context.Canceled) && state.dropAudio.Load():
				history = append(history, message.NewMessage(
					message.Assistant,
					[]message.ContentPart{message.TextContent{
						Text: v.initialMessage + " [interrupted]",
					}},
				))
			case err != nil && !errors.Is(err, context.Canceled):
				return err
			case err == nil:
				history = append(history, message.NewMessage(
					message.Assistant,
					[]message.ContentPart{message.TextContent{
						Text: v.initialMessage,
					}},
				))
			}
			if perr := persistNew(); perr != nil {
				return perr
			}
		}

		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case userText, ok := <-finals:
				if !ok {
					return nil
				}
				if strings.TrimSpace(userText) == "" {
					continue
				}
				history = append(history, message.NewUserMessage(userText))

				drainAudio(ttsAudio)

				turnCtx, turnCancel := context.WithCancel(gctx)
				cancelFn := func() { turnCancel() }
				state.cancelTurn.Store(&cancelFn)
				state.setSpoken("")
				state.dropAudio.Store(false)
				state.agentSpeaking.Store(false)

				newAgent, err := runAssistantTurn(
					turnCtx,
					activeAgent,
					&history,
					emit,
					ttsAudio,
					state,
				)
				activeAgent = newAgent

				turnCancel()
				state.cancelTurn.Store(nil)

				if errors.Is(err, context.Canceled) && state.dropAudio.Load() {
					spoken := strings.TrimSpace(state.loadSpoken())
					if spoken != "" {
						history = append(history, message.NewMessage(
							message.Assistant,
							[]message.ContentPart{
								message.TextContent{
									Text: spoken + " [interrupted]",
								},
							},
						))
					}
					if perr := persistNew(); perr != nil {
						return perr
					}
					state.memorySearched.Store(false)
					state.memoryContext.Store(nil)
					continue
				}
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return err
				}
				if perr := persistNew(); perr != nil {
					return perr
				}
				state.memorySearched.Store(false)
				state.memoryContext.Store(nil)
				if activeAgent.autoExtract && activeAgent.session != nil &&
					activeAgent.memory != nil && activeAgent.memoryID != "" {
					ag := activeAgent
					go func() { _ = ag.extractAndStoreMemories(context.Background()) }()
				}
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case frame, ok := <-ttsAudio:
				if !ok {
					return nil
				}
				if state.dropAudio.Load() {
					continue
				}
				if err := audio.Write(gctx, frame); err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return err
				}
			}
		}
	})

	runErr := g.Wait()
	close(ttsAudio)

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		emit(Event{Type: EventError, Error: runErr})
		c.setErr(runErr)
	}
	emit(Event{Type: EventConversationEnd})

	// Drain any STT results still in flight to release the streaming session.
	//nolint:revive // empty body: discarding remaining results is the intent
	for range sttResults {
	}
}

// initialHistory constructs the starting message history for a conversation.
// If a system prompt is set, it becomes the first message; otherwise history
// starts empty and grows as user turns are committed.
func initialHistory(systemPrompt string) []message.Message {
	if strings.TrimSpace(systemPrompt) == "" {
		return nil
	}
	return []message.Message{message.NewSystemMessage(systemPrompt)}
}

// loadInitialHistory builds the starting history. When v.session is set it
// loads existing messages from the store; if the session is empty and a
// system prompt is configured, it persists the system prompt as the first
// message. Returns the history and the count of messages already persisted
// to the session (0 when no session is configured).
func loadInitialHistory(
	ctx context.Context,
	v *Agent,
) ([]message.Message, int, error) {
	if v.session == nil {
		return initialHistory(v.systemPrompt), 0, nil
	}
	existing, err := v.session.GetMessages(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	if len(existing) > 0 {
		return existing, len(existing), nil
	}
	if strings.TrimSpace(v.systemPrompt) == "" {
		return nil, 0, nil
	}
	sysMsg := message.NewSystemMessage(v.systemPrompt)
	if err := v.session.AddMessages(ctx, []message.Message{sysMsg}); err != nil {
		return nil, 0, err
	}
	return []message.Message{sysMsg}, 1, nil
}

// historyHasNonSystem reports whether the message slice contains any
// non-system message. Used to skip the initial-message greeting when
// resuming a session that already has user or assistant turns.
func historyHasNonSystem(h []message.Message) bool {
	for _, m := range h {
		if m.Role != message.System {
			return true
		}
	}
	return false
}

func drainAudio(ch <-chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// avoid unused import if no stt symbol is referenced directly
var _ stt.SpeechToText
