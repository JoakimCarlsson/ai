package voice

import (
	"context"
	"strings"
	"time"

	"github.com/joakimcarlsson/ai/message"
)

// toolSoundChunkSize is the number of PCM bytes sent per loop iteration. At
// 16 kHz mono signed-16-bit PCM this is ~100ms.
const toolSoundChunkSize = 3200

// ToolSoundBehavior controls when a tool sound plays.
type ToolSoundBehavior int

const (
	// ToolSoundAuto plays the sound only if the agent emitted any spoken
	// content in the same iteration before invoking the tool. Mirrors
	// ElevenAgents' "auto" mode.
	ToolSoundAuto ToolSoundBehavior = iota

	// ToolSoundAlways plays the sound on every tool invocation regardless
	// of whether the agent spoke first. Mirrors ElevenAgents' "always" mode.
	ToolSoundAlways
)

// ToolSoundConfig configures ambient audio that plays while one or more tools
// are executing during an assistant turn. The audio loops until the tools
// finish and is stopped immediately.
type ToolSoundConfig struct {
	// Audio is the PCM clip to loop. Format must match what the configured
	// tts.Generation client emits (typically signed 16-bit little-endian PCM
	// mono at 16 kHz). Empty disables tool sound entirely.
	Audio []byte

	// Behavior controls when the sound plays. Zero value is ToolSoundAuto.
	Behavior ToolSoundBehavior
}

// shouldPlayToolSound decides whether the configured sound should play for a
// tool batch given whether the agent spoke any content in the same iteration.
func shouldPlayToolSound(cfg ToolSoundConfig, spoke bool) bool {
	if len(cfg.Audio) == 0 {
		return false
	}
	if cfg.Behavior == ToolSoundAlways {
		return true
	}
	return spoke
}

// toolSoundChunkInterval paces the looper so chunks are emitted at real-time
// playback rate. ~100ms matches a 3200-byte chunk at 16 kHz mono PCM16. Without
// pacing the looper would dump tens of seconds of audio into the audio queue
// in milliseconds, making the next assistant turn play behind a long tail of
// already-scheduled clicks in the browser.
const toolSoundChunkInterval = 100 * time.Millisecond

// loopToolSound feeds clip bytes into out, looping back to the start when the
// end is reached, paced at real-time playback rate so the audio queue tracks
// wall-clock instead of running ahead. Returns when ctx is canceled.
func loopToolSound(ctx context.Context, clip []byte, out chan<- []byte) {
	if len(clip) == 0 {
		return
	}
	ticker := time.NewTicker(toolSoundChunkInterval)
	defer ticker.Stop()
	for {
		for i := 0; i < len(clip); i += toolSoundChunkSize {
			end := i + toolSoundChunkSize
			if end > len(clip) {
				end = len(clip)
			}
			chunk := clip[i:end]
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}
}

// runToolsWithSound dispatches a batch of tool calls, optionally playing
// ambient audio while they run. Extracted from runAssistantTurn so the
// looper context's cancel is reachable on every return path within a single
// function scope (avoids accumulating defers across the iteration loop).
func runToolsWithSound(
	ctx context.Context,
	v *Agent,
	text string,
	toolCalls []message.ToolCall,
	history *[]message.Message,
	emit func(Event),
	ttsAudio chan<- []byte,
) error {
	spoke := strings.TrimSpace(text) != ""
	if !shouldPlayToolSound(v.toolSound, spoke) {
		return runToolCalls(ctx, v, toolCalls, history, emit)
	}

	soundCtx, soundCancel := context.WithCancel(ctx)
	defer soundCancel()

	soundDone := make(chan struct{})
	emit(Event{Type: EventToolSoundStart, Timestamp: time.Now()})
	go func() {
		defer close(soundDone)
		loopToolSound(soundCtx, v.toolSound.Audio, ttsAudio)
	}()

	runErr := runToolCalls(ctx, v, toolCalls, history, emit)

	soundCancel()
	<-soundDone
	emit(Event{Type: EventToolSoundEnd, Timestamp: time.Now()})

	return runErr
}
