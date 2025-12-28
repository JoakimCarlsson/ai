package agent

import (
	"context"
	"sync"

	"github.com/joakimcarlsson/ai/message"
)

type MemorySession struct {
	id       string
	messages []message.Message
	mu       sync.RWMutex
}

func NewMemorySession(id string) *MemorySession {
	return &MemorySession{
		id:       id,
		messages: make([]message.Message, 0),
	}
}

func (s *MemorySession) ID() string {
	return s.id
}

func (s *MemorySession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit == nil || *limit >= len(s.messages) {
		result := make([]message.Message, len(s.messages))
		copy(result, s.messages)
		return result, nil
	}

	start := len(s.messages) - *limit
	if start < 0 {
		start = 0
	}
	result := make([]message.Message, len(s.messages)-start)
	copy(result, s.messages[start:])
	return result, nil
}

func (s *MemorySession) AddMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msgs...)
	return nil
}

func (s *MemorySession) PopMessage(ctx context.Context) (*message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messages) == 0 {
		return nil, nil
	}

	msg := s.messages[len(s.messages)-1]
	s.messages = s.messages[:len(s.messages)-1]
	return &msg, nil
}

func (s *MemorySession) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = make([]message.Message, 0)
	return nil
}

