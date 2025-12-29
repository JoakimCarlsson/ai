package session

import (
	"context"
	"sync"

	"github.com/joakimcarlsson/ai/message"
)

// memoryStore is an in-memory session store for ephemeral conversations.
type memoryStore struct {
	sessions sync.Map
}

// MemoryStore creates an in-memory session store for ephemeral conversations.
// Useful for testing or when persistence is not required.
func MemoryStore() Store {
	return &memoryStore{}
}

func (s *memoryStore) Exists(ctx context.Context, id string) (bool, error) {
	_, ok := s.sessions.Load(id)
	return ok, nil
}

func (s *memoryStore) Create(ctx context.Context, id string) (Session, error) {
	session := &memorySession{
		id:       id,
		messages: make([]message.Message, 0),
	}
	s.sessions.Store(id, session)
	return session, nil
}

func (s *memoryStore) Load(ctx context.Context, id string) (Session, error) {
	val, ok := s.sessions.Load(id)
	if !ok {
		return nil, nil
	}
	return val.(*memorySession), nil
}

func (s *memoryStore) Delete(ctx context.Context, id string) error {
	s.sessions.Delete(id)
	return nil
}

type memorySession struct {
	id       string
	messages []message.Message
	mu       sync.RWMutex
}

func (s *memorySession) ID() string {
	return s.id
}

func (s *memorySession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
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

func (s *memorySession) AddMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msgs...)
	return nil
}

func (s *memorySession) SetMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = make([]message.Message, len(msgs))
	copy(s.messages, msgs)
	return nil
}

func (s *memorySession) PopMessage(ctx context.Context) (*message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messages) == 0 {
		return nil, nil
	}

	msg := s.messages[len(s.messages)-1]
	s.messages = s.messages[:len(s.messages)-1]
	return &msg, nil
}

func (s *memorySession) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = make([]message.Message, 0)
	return nil
}
