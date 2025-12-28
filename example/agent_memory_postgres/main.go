package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/agent"
	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type PgSessionStore struct {
	db    *sql.DB
	table string
}

func NewPgSessionStore(db *sql.DB, table string) (*PgSessionStore, error) {
	s := &PgSessionStore{db: db, table: table}
	if err := s.createTable(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PgSessionStore) createTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			messages JSONB DEFAULT '[]'
		)
	`, s.table)
	_, err := s.db.Exec(query)
	return err
}

func (s *PgSessionStore) Exists(ctx context.Context, id string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)`, s.table)
	var exists bool
	err := s.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (s *PgSessionStore) Create(ctx context.Context, id string) (session.Session, error) {
	query := fmt.Sprintf(`INSERT INTO %s (id, messages) VALUES ($1, '[]')`, s.table)
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	return &PgSession{db: s.db, table: s.table, id: id}, nil
}

func (s *PgSessionStore) Load(ctx context.Context, id string) (session.Session, error) {
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return &PgSession{db: s.db, table: s.table, id: id}, nil
}

func (s *PgSessionStore) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.table)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

type PgSession struct {
	db    *sql.DB
	table string
	id    string
}

func (s *PgSession) ID() string {
	return s.id
}

func (s *PgSession) GetMessages(ctx context.Context, limit *int) ([]message.Message, error) {
	query := fmt.Sprintf(`SELECT messages FROM %s WHERE id = $1`, s.table)
	var messagesJSON []byte
	err := s.db.QueryRowContext(ctx, query, s.id).Scan(&messagesJSON)
	if err != nil {
		return nil, err
	}

	var messages []message.Message
	if err := json.Unmarshal(messagesJSON, &messages); err != nil {
		return nil, err
	}

	if limit != nil && *limit < len(messages) {
		return messages[len(messages)-*limit:], nil
	}
	return messages, nil
}

func (s *PgSession) AddMessages(ctx context.Context, msgs []message.Message) error {
	existing, err := s.GetMessages(ctx, nil)
	if err != nil {
		return err
	}

	existing = append(existing, msgs...)
	messagesJSON, err := json.Marshal(existing)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`UPDATE %s SET messages = $1 WHERE id = $2`, s.table)
	_, err = s.db.ExecContext(ctx, query, messagesJSON, s.id)
	return err
}

func (s *PgSession) PopMessage(ctx context.Context) (*message.Message, error) {
	messages, err := s.GetMessages(ctx, nil)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}

	msg := messages[len(messages)-1]
	messages = messages[:len(messages)-1]

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`UPDATE %s SET messages = $1 WHERE id = $2`, s.table)
	_, err = s.db.ExecContext(ctx, query, messagesJSON, s.id)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *PgSession) Clear(ctx context.Context) error {
	query := fmt.Sprintf(`UPDATE %s SET messages = '[]' WHERE id = $1`, s.table)
	_, err := s.db.ExecContext(ctx, query, s.id)
	return err
}

type PgMemory struct {
	db       *sql.DB
	embedder embeddings.Embedding
	table    string
}

func NewPgMemory(db *sql.DB, embedder embeddings.Embedding, table string) (*PgMemory, error) {
	m := &PgMemory{db: db, embedder: embedder, table: table}
	if err := m.createTable(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *PgMemory) createTable() error {
	_, err := m.db.Exec(`CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			content TEXT NOT NULL,
			embedding vector(1536),
			metadata JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`, m.table)
	if _, err := m.db.Exec(query); err != nil {
		return err
	}

	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_owner_id_idx ON %s (owner_id)
	`, m.table, m.table)
	_, err = m.db.Exec(indexQuery)
	return err
}

func (m *PgMemory) Store(ctx context.Context, id string, fact string, metadata map[string]any) error {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	memID := uuid.New().String()
	vec := pgvector.NewVector(resp.Embeddings[0])

	query := fmt.Sprintf(`
		INSERT INTO %s (id, owner_id, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, m.table)

	_, err = m.db.ExecContext(ctx, query, memID, id, fact, vec, metadataJSON)
	return err
}

func (m *PgMemory) Search(ctx context.Context, id string, query string, limit int) ([]memory.Entry, error) {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	vec := pgvector.NewVector(resp.Embeddings[0])

	sqlQuery := fmt.Sprintf(`
		SELECT id, owner_id, content, metadata, created_at, 1 - (embedding <=> $1) as score
		FROM %s
		WHERE owner_id = $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`, m.table)

	rows, err := m.db.QueryContext(ctx, sqlQuery, vec, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []memory.Entry
	for rows.Next() {
		var entry memory.Entry
		var metadataJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&entry.ID, &entry.OwnerID, &entry.Content, &metadataJSON, &createdAt, &entry.Score); err != nil {
			return nil, err
		}

		entry.CreatedAt = createdAt
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &entry.Metadata)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (m *PgMemory) GetAll(ctx context.Context, id string, limit int) ([]memory.Entry, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT id, owner_id, content, metadata, created_at
		FROM %s
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, m.table)

	rows, err := m.db.QueryContext(ctx, sqlQuery, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []memory.Entry
	for rows.Next() {
		var entry memory.Entry
		var metadataJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&entry.ID, &entry.OwnerID, &entry.Content, &metadataJSON, &createdAt); err != nil {
			return nil, err
		}

		entry.CreatedAt = createdAt
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &entry.Metadata)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (m *PgMemory) Delete(ctx context.Context, memoryID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, m.table)
	result, err := m.db.ExecContext(ctx, query, memoryID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", memoryID)
	}
	return nil
}

func (m *PgMemory) Update(ctx context.Context, memoryID string, fact string, metadata map[string]any) error {
	resp, err := m.embedder.GenerateEmbeddings(ctx, []string{fact})
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	vec := pgvector.NewVector(resp.Embeddings[0])

	query := fmt.Sprintf(`
		UPDATE %s
		SET content = $1, embedding = $2, metadata = $3
		WHERE id = $4
	`, m.table)

	result, err := m.db.ExecContext(ctx, query, fact, vec, metadataJSON, memoryID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", memoryID)
	}
	return nil
}

func main() {
	ctx := context.Background()

	db, err := sql.Open("postgres", "postgres://postgres:password@localhost:5432/example?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	embedder, err := embeddings.NewEmbedding(model.ProviderOpenAI,
		embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		embeddings.WithModel(model.OpenAIEmbeddingModels[model.TextEmbedding3Small]),
	)
	if err != nil {
		log.Fatal(err)
	}

	llmClient, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llm.WithModel(model.OpenAIModels[model.GPT5Nano]),
		llm.WithMaxTokens(2000),
	)
	if err != nil {
		log.Fatal(err)
	}

	pgMemory, err := NewPgMemory(db, embedder, "memories")
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := NewPgSessionStore(db, "sessions")
	if err != nil {
		log.Fatal(err)
	}

	agent1 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities.`),
		agent.WithMemory("alice", pgMemory,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("conv-1", sessionStore),
	)

	response, err := agent1.Chat(ctx, "Hi! My name is Alice and I love Italian food.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	agent2 := agent.New(llmClient,
		agent.WithSystemPrompt(`You are a personal assistant with memory capabilities.`),
		agent.WithMemory("alice", pgMemory,
			memory.AutoExtract(),
			memory.AutoDedup(),
		),
		agent.WithSession("conv-2", sessionStore),
	)

	response, err = agent2.Chat(ctx, "Can you recommend a restaurant for me?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
