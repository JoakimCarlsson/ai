package voice

import (
	"errors"
	"fmt"
)

// ErrNoAudioTransport is returned by StartConversation when the supplied
// AudioTransport is nil.
var ErrNoAudioTransport = errors.New("voice: audio transport is nil")

// ErrUserMessageDenied is the sentinel returned via EventError when an
// OnUserMessage hook denies a user turn. Use errors.Is to detect.
var ErrUserMessageDenied = errors.New("voice: user message denied by hook")

func errUserMessageDenied(reason string) error {
	if reason == "" {
		return ErrUserMessageDenied
	}
	return fmt.Errorf("%w: %s", ErrUserMessageDenied, reason)
}
