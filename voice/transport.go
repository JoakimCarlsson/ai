package voice

import "context"

// AudioTransport is the duplex audio channel for a conversation. Implementations
// adapt WebSockets, telephony streams, in-memory pipes (for tests), files, etc.
//
// Read returns a frame of mono PCM audio. The returned slice is owned by the
// caller; the transport must not modify it after returning. Returning a nil
// error with a zero-length frame is allowed and treated as a no-op.
//
// Write is called for each TTS audio frame. Sample rate, channel count, and
// PCM encoding are determined by how the consumer constructed the STT and TTS
// clients passed to New; the voice package neither sets nor inspects them.
//
// Close is called when the conversation ends. It must be safe to call once.
type AudioTransport interface {
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, frame []byte) error
	Close() error
}
