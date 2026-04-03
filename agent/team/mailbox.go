package team

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Message represents a message sent between team members.
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Mailbox defines async message passing between team members.
type Mailbox interface {
	Send(ctx context.Context, msg Message) error
	Read(ctx context.Context, recipient string) ([]Message, error)
	RegisterRecipient(name string)
	Close() error
}

type channelMailbox struct {
	mu         sync.RWMutex
	queues     map[string][]Message
	recipients map[string]struct{}
	idGen      atomic.Int64
	closed     atomic.Bool
}

// NewChannelMailbox creates a default in-memory channel-based mailbox.
func NewChannelMailbox() Mailbox {
	return &channelMailbox{
		queues:     make(map[string][]Message),
		recipients: make(map[string]struct{}),
	}
}

func (m *channelMailbox) RegisterRecipient(name string) {
	m.mu.Lock()
	m.recipients[name] = struct{}{}
	if _, ok := m.queues[name]; !ok {
		m.queues[name] = nil
	}
	m.mu.Unlock()
}

func (m *channelMailbox) Send(_ context.Context, msg Message) error {
	if m.closed.Load() {
		return fmt.Errorf("mailbox is closed")
	}

	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg-%d", m.idGen.Add(1))
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if msg.To == "*" {
		for name := range m.recipients {
			if name == msg.From {
				continue
			}
			m.queues[name] = append(m.queues[name], msg)
		}
		return nil
	}

	m.queues[msg.To] = append(m.queues[msg.To], msg)
	return nil
}

func (m *channelMailbox) Read(
	_ context.Context,
	recipient string,
) ([]Message, error) {
	if m.closed.Load() {
		return nil, fmt.Errorf("mailbox is closed")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := m.queues[recipient]
	if len(msgs) == 0 {
		return nil, nil
	}

	m.queues[recipient] = nil
	return msgs, nil
}

func (m *channelMailbox) Close() error {
	m.closed.Store(true)
	return nil
}
