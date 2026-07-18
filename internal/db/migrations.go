package db

import (
	"database/sql"
	"fmt"
	"log"
)

type Migration struct {
	Version int
	Name    string
	SQL     string
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

		if _, err := tx.Exec(m.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %w", m.Version, m.Name, err)
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
