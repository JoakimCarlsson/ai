package voice

import (
	"context"
	"time"

	"github.com/joakimcarlsson/ai/message"
)

// defaultSourceDeadline caps how long a FillerSource may run when the
// FillerConfig.SourceDeadline is unset.
const defaultSourceDeadline = 800 * time.Millisecond

// FillerSource generates a filler phrase given the conversation history so far.
// Errors and empty strings cause the package to fall back to FillerConfig.Message.
type FillerSource func(ctx context.Context, history []message.Message) (string, error)

// FillerConfig configures filler audio that fires when the agent's first audio
// output is delayed past Timeout. Mirrors ElevenAgents' soft_timeout_config
// while leaving the dynamic-generation strategy up to the consumer.
//
// Filler is only spoken when the TTS client implements
// tts.StreamingTextProvider; the single-shot fallback path can't speak a
// filler during the LLM wait.
type FillerConfig struct {
	// Timeout is how long to wait after the LLM stream opens before speaking
	// a filler. A non-positive value disables filler entirely.
	Timeout time.Duration

	// Message is the static phrase spoken when Timeout fires. Required when
	// Timeout > 0 and Source is nil. Also serves as the fallback when Source
	// returns an error or empty string.
	Message string

	// Source, when non-nil, generates the filler dynamically using the
	// conversation history. The package calls this on a background goroutine
	// when the timeout fires, and speaks the result if it returns within
	// SourceDeadline. On error, empty result, or deadline miss, the package
	// falls back to Message.
	Source FillerSource

	// SourceDeadline caps how long Source may run before the package gives up
	// and falls back to Message. Defaults to 800ms when zero. Ignored when
	// Source is nil.
	SourceDeadline time.Duration
}
