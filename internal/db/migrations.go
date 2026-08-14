package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type Migration struct {
	Version int
	Name    string
	SQL     string
	// Apply runs inside the same transaction after SQL (if any). Optional Go-side steps.
	Apply func(tx *sql.Tx) error
}

var migrations = []Migration{
	{
		Version: 1,
		Name:    "initial_schema",
		SQL: `
		CREATE TABLE IF NOT EXISTS user_configs (
			user_id TEXT PRIMARY KEY,
			daily_token_cap INTEGER NOT NULL DEFAULT 50000,
			routing_strategy TEXT NOT NULL DEFAULT 'simple',
			preferred_models TEXT NOT NULL DEFAULT '[]'
		);

		CREATE TABLE IF NOT EXISTS daily_usage (
			user_id TEXT NOT NULL,
			date_str TEXT NOT NULL,
			tokens INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, date_str)
		);

		CREATE TABLE IF NOT EXISTS session_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		);
		`,
	},
	{
		Version: 2,
		Name:    "add_model_and_tokens_to_session_messages",
		SQL: `
		ALTER TABLE session_messages ADD COLUMN model TEXT DEFAULT '';
		ALTER TABLE session_messages ADD COLUMN tokens_consumed INTEGER DEFAULT 0;
		`,
	},
	{
		Version: 3,
		Name:    "usage_events_and_session_user",
		SQL: `
		CREATE TABLE IF NOT EXISTS usage_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			tokens INTEGER NOT NULL,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_usage_events_user_time ON usage_events(user_id, created_at);
		ALTER TABLE session_messages ADD COLUMN user_id TEXT DEFAULT '';
		CREATE INDEX IF NOT EXISTS idx_session_messages_user ON session_messages(user_id, session_id);
		`,
	},
	{
		Version: 4,
		Name:    "backfill_usage_events_and_session_user",
		// Schema already added in v3; this copies legacy daily_usage / empty session user_ids.
		Apply: applyV4BackfillSQLite,
	},
}

func RunMigrations(db *sql.DB) error {
	// Create migration tracking table
	initMetaTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(initMetaTable); err != nil {
		return fmt.Errorf("failed to init schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.Version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration version %d: %w", m.Version, err)
		}

		if count > 0 {
			continue // Migration already applied
		}

		log.Printf("Applying database migration %d: %s...", m.Version, m.Name)
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin tx for migration %d: %w", m.Version, err)
		}

		if err := execMigration(tx, m); err != nil {
			_ = tx.Rollback()
			return err
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.Version, m.Name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}
		log.Printf("Successfully applied migration %d.", m.Version)
	}

	return nil
}

func execMigration(tx *sql.Tx, m Migration) error {
	if sqlText := strings.TrimSpace(m.SQL); sqlText != "" {
		if _, err := tx.Exec(sqlText); err != nil {
			return fmt.Errorf("failed to execute migration %d (%s): %w", m.Version, m.Name, err)
		}
	}
	if m.Apply != nil {
		if err := m.Apply(tx); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", m.Version, m.Name, err)
		}
	}
	return nil
}
