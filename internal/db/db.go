package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type UserConfig struct {
	UserID          string   `json:"user_id"`
	DailyTokenCap   int64    `json:"daily_token_cap"`
	RoutingStrategy string   `json:"routing_strategy"`
	PreferredModels []string `json:"preferred_models"`
}

type SessionMessage struct {
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type UserRepository interface {
	GetUserConfig(ctx context.Context, userID string) (*UserConfig, error)
	UpdateUserConfig(ctx context.Context, config *UserConfig) error
	GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error)
	IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error
}

type SessionRepository interface {
	GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error)
	AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error
}

type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(dbPath string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	repo := &SQLiteRepo{db: db}
	if err := repo.initTables(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepo) initTables() error {
	query := `
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
	`
	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepo) GetUserConfig(ctx context.Context, userID string) (*UserConfig, error) {
	query := `SELECT user_id, daily_token_cap, routing_strategy, preferred_models FROM user_configs WHERE user_id = ?`
	row := r.db.QueryRowContext(ctx, query, userID)

	var cfg UserConfig
	var modelsJSON string
	err := row.Scan(&cfg.UserID, &cfg.DailyTokenCap, &cfg.RoutingStrategy, &modelsJSON)
	if err == sql.ErrNoRows {
		// Provision default profile on first fetch
		defaultCfg := &UserConfig{
			UserID:          userID,
			DailyTokenCap:   50000,
			RoutingStrategy: "simple",
			PreferredModels: []string{"gemini-1.5-flash", "gpt-4o-mini", "claude-3-5-sonnet"},
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

func (r *SQLiteRepo) UpdateUserConfig(ctx context.Context, config *UserConfig) error {
	modelsJSON, err := json.Marshal(config.PreferredModels)
	if err != nil {
		return fmt.Errorf("failed to marshal preferred models: %w", err)
	}

	query := `
	INSERT INTO user_configs (user_id, daily_token_cap, routing_strategy, preferred_models)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET
		daily_token_cap = excluded.daily_token_cap,
		routing_strategy = excluded.routing_strategy,
		preferred_models = excluded.preferred_models
	`
	_, err = r.db.ExecContext(ctx, query, config.UserID, config.DailyTokenCap, config.RoutingStrategy, string(modelsJSON))
	if err != nil {
		return fmt.Errorf("failed to upsert user config: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT tokens FROM daily_usage WHERE user_id = ? AND date_str = ?`
	var tokens int64
	err := r.db.QueryRowContext(ctx, query, userID, dateStr).Scan(&tokens)
	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query daily usage: %w", err)
	}
	return tokens, nil
}

func (r *SQLiteRepo) IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error {
	dateStr := date.Format("2006-01-02")
	query := `
	INSERT INTO daily_usage (user_id, date_str, tokens)
	VALUES (?, ?, ?)
	ON CONFLICT(user_id, date_str) DO UPDATE SET
		tokens = tokens + excluded.tokens
	`
	_, err := r.db.ExecContext(ctx, query, userID, dateStr, tokens)
	if err != nil {
		return fmt.Errorf("failed to increment daily usage: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error) {
	query := `SELECT session_id, role, content, timestamp FROM session_messages WHERE session_id = ? ORDER BY id DESC LIMIT ?`
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
	// Reverse to return chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *SQLiteRepo) AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error {
	query := `INSERT INTO session_messages (session_id, role, content, timestamp) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, sessionID, msg.Role, msg.Content, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to append session message: %w", err)
	}
	return nil
}
