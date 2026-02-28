package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/message"
)

type sessionStore struct {
	db     *sql.DB
	prefix string
}

// SessionStore creates a new SQLite-backed session store using the provided database connection.
// It automatically creates the required tables if they don't exist.
func SessionStore(
	ctx context.Context,
	db *sql.DB,
	opts ...Option,
) (session.Store, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	prefix := options.tablePrefix

	sessionsTable := prefix + "sessions"
	messagesTable := prefix + "messages"

	createSessionsSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id         TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL
		)`, sessionsTable)

	createMessagesSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
			role       TEXT NOT NULL,
			parts      TEXT NOT NULL,
			model      TEXT,
			created_at INTEGER NOT NULL
		)`, messagesTable, sessionsTable)

	createIndexSQL := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS idx_%smessages_session ON %s(session_id, id)`,
		prefix,
		messagesTable,
	)

	if _, err := db.ExecContext(ctx, createSessionsSQL); err != nil {
		return nil, fmt.Errorf("failed to create sessions table: %w", err)
	}
	if _, err := db.ExecContext(ctx, createMessagesSQL); err != nil {
		return nil, fmt.Errorf("failed to create messages table: %w", err)
	}
	if _, err := db.ExecContext(ctx, createIndexSQL); err != nil {
		return nil, fmt.Errorf("failed to create messages index: %w", err)
	}

	return &sessionStore{db: db, prefix: prefix}, nil
}

func (s *sessionStore) Exists(ctx context.Context, id string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT EXISTS(SELECT 1 FROM %ssessions WHERE id = ?)",
		s.prefix,
	)
	var exists bool
	err := s.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (s *sessionStore) Create(
	ctx context.Context,
	id string,
) (session.Session, error) {
	query := fmt.Sprintf(
		"INSERT INTO %ssessions (id, created_at) VALUES (?, ?)",
		s.prefix,
	)
	_, err := s.db.ExecContext(ctx, query, id, time.Now().UnixNano())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return &sqliteSession{db: s.db, id: id, prefix: s.prefix}, nil
}

func (s *sessionStore) Load(
	ctx context.Context,
	id string,
) (session.Session, error) {
	return &sqliteSession{db: s.db, id: id, prefix: s.prefix}, nil
}

func (s *sessionStore) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %ssessions WHERE id = ?", s.prefix)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

type sqliteSession struct {
	db     *sql.DB
	id     string
	prefix string
}

func (s *sqliteSession) ID() string {
	return s.id
}

func (s *sqliteSession) GetMessages(
	ctx context.Context,
	limit *int,
) ([]message.Message, error) {
	table := s.prefix + "messages"

	var query string
	var args []any

	if limit != nil {
		query = fmt.Sprintf(`
			SELECT parts FROM (
				SELECT parts, id FROM %s
				WHERE session_id = ? ORDER BY id DESC LIMIT ?
			) sub ORDER BY id ASC`, table)
		args = []any{s.id, *limit}
	} else {
		query = fmt.Sprintf("SELECT parts FROM %s WHERE session_id = ? ORDER BY id ASC", table)
		args = []any{s.id}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
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

func (s *sqliteSession) AddMessages(
	ctx context.Context,
	msgs []message.Message,
) error {
	table := s.prefix + "messages"
	query := fmt.Sprintf(
		"INSERT INTO %s (session_id, role, parts, model, created_at) VALUES (?, ?, ?, ?, ?)",
		table,
	)

	for _, msg := range msgs {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		_, err = s.db.ExecContext(ctx, query,
			s.id, string(msg.Role), msgJSON, string(msg.Model), msg.CreatedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteSession) PopMessage(
	ctx context.Context,
) (*message.Message, error) {
	table := s.prefix + "messages"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var rowID int64
	var msgJSON []byte

	err = tx.QueryRowContext(
		ctx,
		fmt.Sprintf(
			"SELECT id, parts FROM %s WHERE session_id = ? ORDER BY id DESC LIMIT 1",
			table,
		),
		s.id,
	).Scan(&rowID, &msgJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE id = ?", table),
		rowID,
	)
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

func (s *sqliteSession) Clear(ctx context.Context) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE session_id = ?",
		s.prefix+"messages",
	)
	_, err := s.db.ExecContext(ctx, query, s.id)
	return err
}
