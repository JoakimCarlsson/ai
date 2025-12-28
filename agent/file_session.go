package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/joakimcarlsson/ai/message"
)

type FileSession struct {
	id       string
	filePath string
	mu       sync.RWMutex
}

func NewFileSession(id string, directory string) (*FileSession, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}

	return &FileSession{
		id:       id,
		filePath: filepath.Join(directory, id+".json"),
	}, nil
}

func (s *FileSession) ID() string {
	return s.id
}

func (s *FileSession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages, err := s.loadMessages()
	if err != nil {
		return nil, err
	}

	if limit == nil || *limit >= len(messages) {
		return messages, nil
	}

	start := len(messages) - *limit
	if start < 0 {
		start = 0
	}
	return messages[start:], nil
}

func (s *FileSession) AddMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadMessages()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing = append(existing, msgs...)
	return s.saveMessages(existing)
}

func (s *FileSession) PopMessage(ctx context.Context) (*message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, err := s.loadMessages()
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, nil
	}

	msg := messages[len(messages)-1]
	messages = messages[:len(messages)-1]

	if err := s.saveMessages(messages); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (s *FileSession) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(s.filePath)
}

func (s *FileSession) loadMessages() ([]message.Message, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []message.Message{}, nil
		}
		return nil, err
	}

	var messages []message.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *FileSession) saveMessages(messages []message.Message) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}
