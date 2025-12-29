package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/joakimcarlsson/ai/message"
)

// fileStore is a file-based session store that persists conversations to disk.
type fileStore struct {
	dir string
}

// FileStore creates a file-based session store that persists conversations to disk.
// Sessions are stored as JSON files in the specified directory.
func FileStore(dir string) Store {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
	}
	return &fileStore{dir: dir}
}

func (s *fileStore) filePath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *fileStore) Exists(ctx context.Context, id string) (bool, error) {
	_, err := os.Stat(s.filePath(id))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *fileStore) Create(ctx context.Context, id string) (Session, error) {
	filePath := s.filePath(id)
	if err := os.WriteFile(filePath, []byte("[]"), 0644); err != nil {
		return nil, err
	}
	return &fileSession{id: id, filePath: filePath}, nil
}

func (s *fileStore) Load(ctx context.Context, id string) (Session, error) {
	return &fileSession{id: id, filePath: s.filePath(id)}, nil
}

func (s *fileStore) Delete(ctx context.Context, id string) error {
	return os.Remove(s.filePath(id))
}

type fileSession struct {
	id       string
	filePath string
	mu       sync.RWMutex
}

func (s *fileSession) ID() string {
	return s.id
}

func (s *fileSession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
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

func (s *fileSession) AddMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadMessages()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing = append(existing, msgs...)
	return s.saveMessages(existing)
}

func (s *fileSession) SetMessages(ctx context.Context, msgs []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveMessages(msgs)
}

func (s *fileSession) PopMessage(ctx context.Context) (*message.Message, error) {
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

func (s *fileSession) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(s.filePath)
}

func (s *fileSession) loadMessages() ([]message.Message, error) {
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

func (s *fileSession) saveMessages(messages []message.Message) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

