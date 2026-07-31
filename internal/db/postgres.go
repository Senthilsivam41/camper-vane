package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(databaseURL string) (*PostgresRepo, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	repo := &PostgresRepo{db: db}
	if err := runPostgresMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run postgres migrations: %w", err)
	}
	return repo, nil
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}

func (r *PostgresRepo) GetUserConfig(ctx context.Context, userID string) (*UserConfig, error) {
	query := `SELECT user_id, daily_token_cap, routing_strategy, preferred_models FROM user_configs WHERE user_id = $1`
	row := r.db.QueryRowContext(ctx, query, userID)

	var cfg UserConfig
	var modelsJSON string
	err := row.Scan(&cfg.UserID, &cfg.DailyTokenCap, &cfg.RoutingStrategy, &modelsJSON)
	if err == sql.ErrNoRows {
		defaultCfg := &UserConfig{
			UserID:          userID,
			DailyTokenCap:   50000,
			RoutingStrategy: "simple",
			PreferredModels: []string{"gemini-1.5-flash", "gpt-4o-mini", "claude-3-5-sonnet", "sonar"},
		}
		if err := r.UpdateUserConfig(ctx, defaultCfg); err != nil {
			return nil, fmt.Errorf("failed to provision default user config: %w", err)
		}
		return defaultCfg, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user config: %w", err)
	}

	if err := json.Unmarshal([]byte(modelsJSON), &cfg.PreferredModels); err != nil {
		cfg.PreferredModels = []string{}
	}
	return &cfg, nil
}

func (r *PostgresRepo) UpdateUserConfig(ctx context.Context, config *UserConfig) error {
	modelsJSON, err := json.Marshal(config.PreferredModels)
	if err != nil {
		return fmt.Errorf("failed to marshal preferred models: %w", err)
	}

	query := `
	INSERT INTO user_configs (user_id, daily_token_cap, routing_strategy, preferred_models)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id) DO UPDATE SET
		daily_token_cap = EXCLUDED.daily_token_cap,
		routing_strategy = EXCLUDED.routing_strategy,
		preferred_models = EXCLUDED.preferred_models
	`
	_, err = r.db.ExecContext(ctx, query, config.UserID, config.DailyTokenCap, config.RoutingStrategy, string(modelsJSON))
	if err != nil {
		return fmt.Errorf("failed to upsert user config: %w", err)
	}
	return nil
}

func (r *PostgresRepo) GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT tokens FROM daily_usage WHERE user_id = $1 AND date_str = $2`
	var tokens int64
	err := r.db.QueryRowContext(ctx, query, userID, dateStr).Scan(&tokens)
	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query daily usage: %w", err)
	}
	return tokens, nil
}

func (r *PostgresRepo) IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error {
	dateStr := date.Format("2006-01-02")
	query := `
	INSERT INTO daily_usage (user_id, date_str, tokens)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id, date_str) DO UPDATE SET
		tokens = daily_usage.tokens + EXCLUDED.tokens
	`
	_, err := r.db.ExecContext(ctx, query, userID, dateStr, tokens)
	if err != nil {
		return fmt.Errorf("failed to increment daily usage: %w", err)
	}
	return nil
}

func (r *PostgresRepo) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error) {
	query := `SELECT session_id, role, content, timestamp FROM session_messages WHERE session_id = $1 ORDER BY id DESC LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query session history: %w", err)
	}
	defer rows.Close()

	var msgs []SessionMessage
	for rows.Next() {
		var m SessionMessage
		if err := rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *PostgresRepo) AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error {
	query := `INSERT INTO session_messages (session_id, role, content, timestamp) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, sessionID, msg.Role, msg.Content, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to append session message: %w", err)
	}
	return nil
}

func runPostgresMigrations(db *sql.DB) error {
	initMetaTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	if _, err := db.Exec(initMetaTable); err != nil {
		return fmt.Errorf("failed to init schema_migrations table: %w", err)
	}

	pgMigrations := []Migration{
		{
			Version: 1,
			Name:    "initial_schema",
			SQL: `
			CREATE TABLE IF NOT EXISTS user_configs (
				user_id TEXT PRIMARY KEY,
				daily_token_cap BIGINT NOT NULL DEFAULT 50000,
				routing_strategy TEXT NOT NULL DEFAULT 'simple',
				preferred_models TEXT NOT NULL DEFAULT '[]'
			);

			CREATE TABLE IF NOT EXISTS daily_usage (
				user_id TEXT NOT NULL,
				date_str TEXT NOT NULL,
				tokens BIGINT NOT NULL DEFAULT 0,
				PRIMARY KEY (user_id, date_str)
			);

			CREATE TABLE IF NOT EXISTS session_messages (
				id BIGSERIAL PRIMARY KEY,
				session_id TEXT NOT NULL,
				role TEXT NOT NULL,
				content TEXT NOT NULL,
				timestamp TIMESTAMPTZ NOT NULL
			);
			`,
		},
		{
			Version: 2,
			Name:    "add_model_and_tokens_to_session_messages",
			SQL: `
			ALTER TABLE session_messages ADD COLUMN IF NOT EXISTS model TEXT DEFAULT '';
			ALTER TABLE session_messages ADD COLUMN IF NOT EXISTS tokens_consumed BIGINT DEFAULT 0;
			`,
		},
	}

	for _, m := range pgMigrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", m.Version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration version %d: %w", m.Version, err)
		}
		if count > 0 {
			continue
		}

		log.Printf("Applying postgres migration %d: %s...", m.Version, m.Name)
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin tx for migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %w", m.Version, m.Name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}
		log.Printf("Successfully applied postgres migration %d.", m.Version)
	}

	return nil
}
