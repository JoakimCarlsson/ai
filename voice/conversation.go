package voice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Conversation is a running voice session. Construct one via
// VoiceAgent.StartConversation; consume Events while it runs and call Wait to
// observe the final result. Cancel the context passed to StartConversation to
// terminate the conversation.
type Conversation struct {
	id     string
	events chan Event
	done   chan struct{}

	errMu sync.Mutex
	err   error
}

// ID returns a stable identifier for this conversation, useful for logging
// and tracing correlation.
func (c *Conversation) ID() string {
	return c.id
}

// Events returns the channel of events emitted during the conversation. The
// channel is closed when the conversation ends. The consumer must drain it;
// failing to do so will block the pipeline.
func (c *Conversation) Events() <-chan Event {
	return c.events
}

// Wait blocks until the conversation ends and returns any unrecoverable
// error that terminated it. Returns nil when the conversation ended due to
// ctx cancellation or a clean transport close.
func (c *Conversation) Wait() error {
	<-c.done
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.err
}

func (c *Conversation) setErr(err error) {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	if c.err == nil {
		c.err = err
	}
}

// StartConversation opens a new conversation over the given audio transport
// and runs the pipeline in the background. Cancel ctx to terminate.
func (v *VoiceAgent) StartConversation(
	ctx context.Context,
	audio AudioTransport,
) (*Conversation, error) {
	if audio == nil {
		return nil, ErrNoAudioTransport
	}
	c := &Conversation{
		id:     newConversationID(),
		events: make(chan Event, 32),
		done:   make(chan struct{}),
	}
	go c.run(ctx, v, audio)
	return c, nil
}

func newConversationID() string {
	var buf [12]byte
	_, _ = rand.Read(buf[:])
	return "conv-" + hex.EncodeToString(buf[:])
}
