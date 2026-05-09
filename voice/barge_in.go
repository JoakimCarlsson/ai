package voice

import "sync/atomic"

// BargeInPolicy controls whether the agent stops speaking when the user
// starts to talk over it.
type BargeInPolicy int

const (
	// BargeInIgnore lets the agent finish whatever it is saying. STT partials
	// arriving during agent speech are observed but cause no action. Default.
	BargeInIgnore BargeInPolicy = iota

	// BargeInInterrupt cancels the current LLM/TTS turn the moment STT emits
	// a non-empty partial during agent speech. Server-side queued audio is
	// dropped, the browser flushes its audio queue (via the
	// EventAgentInterrupted event), and the assistant's history entry is
	// truncated to whatever was actually spoken.
	BargeInInterrupt
)

// turnState carries per-turn coordination state between the runner's
// goroutines and streamLLMAndSpeak. It is reset at the start of each user
// turn. All fields are safe for concurrent access.
type turnState struct {
	// agentSpeaking reports whether TTS audio is currently flowing for this
	// turn. Pipeline.go sets it on EventTTSStarted and clears it on
	// EventTTSEnded; the STT consumer reads it to decide whether to fire
	// barge-in.
	agentSpeaking atomic.Bool

	// dropAudio tells the audio-out pump to discard frames pulled from the
	// ttsAudio channel. Set when barge-in fires; reset when the next turn
	// begins so the next turn's audio plays normally.
	dropAudio atomic.Bool

	// cancelTurn cancels the per-turn context. Set by the turn driver before
	// runAssistantTurn is called and cleared after it returns. The STT
	// consumer calls it on barge-in.
	cancelTurn atomic.Pointer[func()]

	// spokenSoFar stores a running accumulator of assistant text that has
	// been pushed into TTS for the current turn. Used to truncate the
	// history entry on barge-in.
	spokenSoFar atomic.Pointer[string]
}

// setSpoken stores text as the current spoken-so-far value.
func (s *turnState) setSpoken(text string) {
	s.spokenSoFar.Store(&text)
}

// loadSpoken returns the current spoken-so-far text, or "" if unset.
func (s *turnState) loadSpoken() string {
	if p := s.spokenSoFar.Load(); p != nil {
		return *p
	}
	return ""
}

// fireCancel runs the per-turn cancel func if one is set.
func (s *turnState) fireCancel() {
	if p := s.cancelTurn.Load(); p != nil {
		(*p)()
	}
}
