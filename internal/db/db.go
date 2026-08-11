package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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
	UserID    string    `json:"user_id,omitempty"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type SessionSummary struct {
	SessionID    string    `json:"session_id"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	Preview      string    `json:"preview"`
}

type UserRepository interface {
	GetUserConfig(ctx context.Context, userID string) (*UserConfig, error)
	UpdateUserConfig(ctx context.Context, config *UserConfig) error
	GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error)
	GetUsageSince(ctx context.Context, userID string, since time.Time) (int64, error)
	IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error
}

type SessionRepository interface {
	GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error)
	AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error
	ListSessions(ctx context.Context, userID string, limit int) ([]SessionSummary, error)
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
	if err := RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

func NewSQLiteRepoFromEnv() (*SQLiteRepo, error) {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "camper_vane.db"
	}
	return NewSQLiteRepo(dbPath)
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
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO usage_events (user_id, tokens, created_at) VALUES (?, ?, ?)`,
		userID, tokens, date.UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to record usage event: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) GetUsageSince(ctx context.Context, userID string, since time.Time) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens), 0) FROM usage_events WHERE user_id = ? AND created_at >= ?`,
		userID, since.UTC(),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to query usage window: %w", err)
	}
	return total.Int64, nil
}

func (r *SQLiteRepo) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error) {
	query := `SELECT session_id, COALESCE(user_id, ''), role, content, timestamp FROM session_messages WHERE session_id = ? ORDER BY id DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query session history: %w", err)
	}
	defer rows.Close()

	var msgs []SessionMessage
	for rows.Next() {
		var m SessionMessage
		if err := rows.Scan(&m.SessionID, &m.UserID, &m.Role, &m.Content, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *SQLiteRepo) AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error {
	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	query := `INSERT INTO session_messages (session_id, user_id, role, content, timestamp) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, sessionID, msg.UserID, msg.Role, msg.Content, ts)
	if err != nil {
		return fmt.Errorf("failed to append session message: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) ListSessions(ctx context.Context, userID string, limit int) ([]SessionSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
	SELECT session_id,
	       MAX(timestamp) AS updated_at,
	       COUNT(*) AS message_count,
	       COALESCE((
	         SELECT content FROM session_messages sm2
	         WHERE sm2.session_id = sm.session_id AND sm2.user_id = sm.user_id
	         ORDER BY sm2.id ASC LIMIT 1
	       ), '') AS preview
	FROM session_messages sm
	WHERE user_id = ?
	GROUP BY session_id
	ORDER BY updated_at DESC
	LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var out []SessionSummary
	for rows.Next() {
		var s SessionSummary
		var updatedAt string
		if err := rows.Scan(&s.SessionID, &updatedAt, &s.MessageCount, &s.Preview); err != nil {
			return nil, err
		}
		if ts, err := parseFlexibleTime(updatedAt); err == nil {
			s.UpdatedAt = ts
		}
		if len(s.Preview) > 80 {
			s.Preview = s.Preview[:80] + "…"
		}
		out = append(out, s)
	}
	return out, nil
}

func parseFlexibleTime(v string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999+00:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, v); err == nil {
			return ts, nil
		}
	}
	return time.Parse(time.RFC3339, v)
}
