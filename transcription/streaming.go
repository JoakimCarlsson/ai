package transcription

import (
	"context"
	"errors"
)

// ErrStreamingNotSupported is returned by StreamTranscribe when the underlying
// provider only supports request/response transcription. Detect ahead of time
// via SupportsStreaming.
var ErrStreamingNotSupported = errors.New(
	"transcription: streaming not supported by this provider",
)

// StreamResult is one event emitted by StreamTranscribe. Interim results have
// IsFinal=false; the settled transcript is emitted with IsFinal=true. Errors
// are sent as a final StreamResult{Error: ...} value before the channel closes.
type StreamResult struct {
	Text       string
	Confidence float64
	IsFinal    bool
	WordCount  int
	Words      []Word
	Error      error
}

// streamingSpeechToTextClient is the internal sub-interface a provider client
// implements when it supports streaming. baseSpeechToText resolves it once at
// construction and uses the cached value to back StreamTranscribe.
type streamingSpeechToTextClient interface {
	streamTranscribe(
		ctx context.Context,
		audio <-chan []byte,
		options ...Option,
	) (<-chan StreamResult, error)
}

// WithStreamSampleRate declares the PCM sample rate (Hz) of the audio fed
// into the streaming session. Defaults to 16000 when supported.
func WithStreamSampleRate(hz int) Option {
	return func(options *Options) {
		options.SampleRate = hz
	}
}

// WithStreamChannels declares the channel count of the audio fed into the
// streaming session. Defaults to 1 when supported.
func WithStreamChannels(n int) Option {
	return func(options *Options) {
		options.Channels = n
	}
}
