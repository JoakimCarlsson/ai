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
// VoiceAgent.StartConversation and never returns until the conversation ends
// (ctx cancelled, transport closed, or unrecoverable error). The error is
// stored on the Conversation and surfaced via Wait; the events channel is
// closed before run returns.
func (c *Conversation) run(ctx context.Context, v *VoiceAgent, audio AudioTransport) {
	defer close(c.events)
	defer close(c.done)
	defer audio.Close()

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
		var sawPartialThisUtterance bool
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
			if r.IsFinal {
				emit(Event{Type: EventUserTranscriptFinal, Text: r.Text})
				sawPartialThisUtterance = false
				select {
				case finals <- r.Text:
				case <-gctx.Done():
					return gctx.Err()
				}
				continue
			}
			if !sawPartialThisUtterance {
				emit(Event{Type: EventUserSpeechStart})
				sawPartialThisUtterance = true
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
			emit(Event{Type: EventUserTranscriptPartial, Text: r.Text})
		}
		return nil
	})

	g.Go(func() error {
		history := initialHistory(v.systemPrompt)
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

				err := runAssistantTurn(turnCtx, v, &history, emit, ttsAudio, state)

				turnCancel()
				state.cancelTurn.Store(nil)

				if errors.Is(err, context.Canceled) && state.dropAudio.Load() {
					spoken := strings.TrimSpace(state.loadSpoken())
					if spoken != "" {
						history = append(history, message.NewMessage(
							message.Assistant,
							[]message.ContentPart{
								message.TextContent{Text: spoken + " [interrupted]"},
							},
						))
					}
					continue
				}
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					return err
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
