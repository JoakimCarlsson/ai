package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
)

const createSessionsTableSQL = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`

const createMessagesTableSQL = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    parts JSONB NOT NULL,
    model TEXT,
    created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS messages_session_idx ON messages(session_id, created_at)`

type sessionStore struct {
	db          *sql.DB
	idGenerator IDGenerator
}

// SessionStore creates a new PostgreSQL-backed session store.
// It automatically creates the sessions and messages tables if they don't exist.
func SessionStore(ctx context.Context, connString string, opts ...Option) (session.Store, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	db, err := openDB(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if _, err := db.ExecContext(ctx, createSessionsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create sessions table: %w", err)
	}

	if _, err := db.ExecContext(ctx, createMessagesTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create messages table: %w", err)
	}

	return &sessionStore{db: db, idGenerator: options.idGenerator}, nil
}

func (s *sessionStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1)", id,
	).Scan(&exists)
	return exists, err
}

func (s *sessionStore) Create(ctx context.Context, id string) (session.Session, error) {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (id) VALUES ($1)", id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return &pgSession{db: s.db, id: id, idGenerator: s.idGenerator}, nil
}

func (s *sessionStore) Load(ctx context.Context, id string) (session.Session, error) {
	return &pgSession{db: s.db, id: id, idGenerator: s.idGenerator}, nil
}

func (s *sessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

type pgSession struct {
	db          *sql.DB
	id          string
	idGenerator IDGenerator
}

func (s *pgSession) ID() string {
	return s.id
}

func (s *pgSession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
	query := `
		SELECT parts
		FROM messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	if limit != nil {
		query = fmt.Sprintf(`
			SELECT parts FROM (
				SELECT parts, created_at
				FROM messages
				WHERE session_id = $1
				ORDER BY created_at DESC
				LIMIT %d
			) sub ORDER BY created_at ASC
		`, *limit)
	}

	rows, err := s.db.QueryContext(ctx, query, s.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []message.Message
	for rows.Next() {
		var msgJSON []byte

		if err := rows.Scan(&msgJSON); err != nil {
			return nil, err
		}

		var msg message.Message
		if err := json.Unmarshal(msgJSON, &msg); err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	if messages == nil {
		messages = []message.Message{}
	}

	return messages, rows.Err()
}

func (s *pgSession) AddMessages(ctx context.Context, msgs []message.Message) error {
	for _, msg := range msgs {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO messages (id, session_id, role, parts, model, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, s.idGenerator(), s.id, string(msg.Role), msgJSON, string(msg.Model), msg.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *pgSession) PopMessage(ctx context.Context) (*message.Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var msgID string
	var msgJSON []byte

	err = tx.QueryRowContext(ctx, `
		SELECT id, parts
		FROM messages
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, s.id).Scan(&msgID, &msgJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM messages WHERE id = $1", msgID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var msg message.Message
	if err := json.Unmarshal(msgJSON, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (s *pgSession) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM messages WHERE session_id = $1", s.id)
	return err
}
