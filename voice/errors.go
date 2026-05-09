package voice

import "errors"

// ErrNoAudioTransport is returned by StartConversation when the supplied
// AudioTransport is nil.
var ErrNoAudioTransport = errors.New("voice: audio transport is nil")
