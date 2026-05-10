package pgvector

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"hash/fnv"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ledgerTableFor returns the bookkeeping table name that records
// applied migrations for a given chunks table. The ledger is
// per-table so multiple knowledge bases (different chunk tables)
// in the same database have isolated migration history.
func ledgerTableFor(chunksTable string) string {
	return chunksTable + "_migrations"
}

// MigrationsFS exposes the embedded migration files as an fs.FS so
// callers who already run their own migrator (golang-migrate, goose,
// atlas, etc.) can plug the SQL into their existing pipeline instead
// of using Migrate.
//
// Files are named NNN_description.sql and contain Go-template
// placeholders for {{.Table}} and {{.Dims}}; tools that don't render
// templates need to substitute these before applying.
func MigrationsFS() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		// Cannot happen with a static embed; treat as a programming error.
		panic(fmt.Errorf("pgvector: migrations fs: %w", err))
	}
	return sub
}

// SchemaVersion returns the highest migration version embedded in
// this build. New uses this to decide whether the database is up to
// date.
func SchemaVersion() int {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return 0
	}
	max := 0
	for _, e := range entries {
		if v, ok := parseMigrationVersion(e.Name()); ok && v > max {
			max = v
		}
	}
	return max
}

// Migrate applies any pending schema migrations to the database
// reachable via connString. Idempotent and safe under concurrent
// invocation: each migration is wrapped in a transaction guarded by
// a Postgres advisory lock keyed on the configured table name, so
// racing migrators serialise instead of colliding on the catalog.
//
// Run from a privileged Postgres role at deploy time. Application
// processes should call New, which only verifies the schema and
// requires DML privileges.
//
// dims is the embedding dimensionality for migration 001. It is
// baked into the embedding column type as vector(N). For migrations
// that don't reference {{.Dims}} the value is ignored.
func Migrate(
	ctx context.Context,
	connString string,
	dims int,
	opts ...Option,
) error {
	if dims <= 0 {
		return fmt.Errorf("pgvector: dims must be positive, got %d", dims)
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if !validTableName(cfg.table) {
		return fmt.Errorf("pgvector: invalid table name %q", cfg.table)
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return fmt.Errorf("pgvector: open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("pgvector: ping: %w", err)
	}

	return apply(ctx, db, cfg.table, dims)
}

// apply runs the bootstrap (ledger creation) and then each pending
// migration in numbered order, each in its own transaction.
func apply(ctx context.Context, db *sql.DB, table string, dims int) error {
	if err := ensureLedger(ctx, db, table); err != nil {
		return err
	}

	migs, err := listMigrations()
	if err != nil {
		return err
	}

	current, err := readVersion(ctx, db, table)
	if err != nil {
		return err
	}

	for _, m := range migs {
		if m.version <= current {
			continue
		}
		if err := applyOne(ctx, db, table, dims, m); err != nil {
			return err
		}
	}
	return nil
}

// ensureLedger creates the per-table bookkeeping table the first
// time Migrate runs against this chunks table. Idempotent; uses an
// advisory lock keyed on the chunks table so concurrent first-time
// callers don't race on the catalog.
func ensureLedger(ctx context.Context, db *sql.DB, table string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgvector: begin ledger tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"SELECT pg_advisory_xact_lock($1)", advisoryLockKey(table),
	); err != nil {
		return fmt.Errorf("pgvector: ensure-ledger lock: %w", err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    version    INT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, ledgerTableFor(table))); err != nil {
		return fmt.Errorf("pgvector: create ledger: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgvector: commit ledger: %w", err)
	}
	return nil
}

func readVersion(ctx context.Context, db *sql.DB, table string) (int, error) {
	var current int
	err := db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM "+ledgerTableFor(table),
	).Scan(&current)
	if err != nil {
		return 0, fmt.Errorf("pgvector: read ledger: %w", err)
	}
	return current, nil
}

type migration struct {
	version int
	name    string
}

func listMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("pgvector: list embedded migrations: %w", err)
	}
	var out []migration
	for _, e := range entries {
		v, ok := parseMigrationVersion(e.Name())
		if !ok {
			continue
		}
		out = append(out, migration{version: v, name: e.Name()})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].version < out[j].version
	})
	// Sanity: contiguous from 1 to N.
	for i, m := range out {
		if m.version != i+1 {
			return nil, fmt.Errorf(
				"pgvector: non-contiguous migration versions; got %d at index %d",
				m.version, i,
			)
		}
	}
	return out, nil
}

// parseMigrationVersion extracts the leading integer prefix from a
// filename like "001_initial.sql". Returns false for files that
// don't start with digits or aren't .sql.
func parseMigrationVersion(name string) (int, bool) {
	if !strings.HasSuffix(name, ".sql") {
		return 0, false
	}
	rest, _, _ := strings.Cut(strings.TrimSuffix(name, ".sql"), "_")
	v, err := strconv.Atoi(rest)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// applyOne renders one migration against the per-deploy params and
// runs it in a transaction with an advisory lock keyed on the table.
// The lock is held only for the duration of the migration so a slow
// migration on one table doesn't block deploys on another.
func applyOne(
	ctx context.Context,
	db *sql.DB,
	table string,
	dims int,
	m migration,
) error {
	raw, err := migrationsFS.ReadFile("migrations/" + m.name)
	if err != nil {
		return fmt.Errorf("pgvector: read %s: %w", m.name, err)
	}

	tmpl, err := template.New(m.name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return fmt.Errorf("pgvector: parse %s: %w", m.name, err)
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, struct {
		Table string
		Dims  int
	}{Table: table, Dims: dims}); err != nil {
		return fmt.Errorf("pgvector: render %s: %w", m.name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgvector: begin migration %d: %w", m.version, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"SELECT pg_advisory_xact_lock($1)", advisoryLockKey(table),
	); err != nil {
		return fmt.Errorf("pgvector: migration %d lock: %w", m.version, err)
	}

	// Re-check version under the lock so two racing migrators don't
	// double-apply the same file.
	var current int
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM "+ledgerTableFor(table),
	).Scan(&current); err != nil {
		return fmt.Errorf("pgvector: re-read ledger: %w", err)
	}
	if current >= m.version {
		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, rendered.String()); err != nil {
		return fmt.Errorf("pgvector: apply %s: %w", m.name, err)
	}

	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", ledgerTableFor(table)),
		m.version,
	); err != nil {
		return fmt.Errorf("pgvector: record migration %d: %w", m.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgvector: commit migration %d: %w", m.version, err)
	}
	return nil
}

// advisoryLockKey returns a stable int64 derived from the table
// name. All processes migrating the same table coordinate on the
// same key.
func advisoryLockKey(table string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("rag-pgvector:table:"))
	_, _ = h.Write([]byte(table))
	return int64(h.Sum64())
}

