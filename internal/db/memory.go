package db

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryRepo is an in-memory Store for explicit mock/unit evaluations of repository contracts.
type MemoryRepo struct {
	mu       sync.Mutex
	users    map[string]*UserConfig
	usage    map[string]int64 // key: userID|date
	sessions map[string][]SessionMessage
	seq      int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		users:    make(map[string]*UserConfig),
		usage:    make(map[string]int64),
		sessions: make(map[string][]SessionMessage),
	}
}

func (r *MemoryRepo) Close() error { return nil }

func (r *MemoryRepo) GetUserConfig(ctx context.Context, userID string) (*UserConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cfg, ok := r.users[userID]; ok {
		cp := *cfg
		cp.PreferredModels = append([]string{}, cfg.PreferredModels...)
		return &cp, nil
	}
	cfg := &UserConfig{
		UserID:          userID,
		DailyTokenCap:   50000,
		RoutingStrategy: "simple",
		PreferredModels: []string{"gemini-1.5-flash", "gpt-4o-mini", "claude-3-5-sonnet", "sonar"},
	}
	r.users[userID] = cfg
	cp := *cfg
	cp.PreferredModels = append([]string{}, cfg.PreferredModels...)
	return &cp, nil
}

func (r *MemoryRepo) UpdateUserConfig(ctx context.Context, config *UserConfig) error {
	if config == nil || config.UserID == "" {
		return fmt.Errorf("user config requires user_id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *config
	cp.PreferredModels = append([]string{}, config.PreferredModels...)
	r.users[config.UserID] = &cp
	return nil
}

func (r *MemoryRepo) GetDailyUsage(ctx context.Context, userID string, date time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.usage[usageKey(userID, date)], nil
}

func (r *MemoryRepo) IncrementDailyUsage(ctx context.Context, userID string, date time.Time, tokens int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := usageKey(userID, date)
	r.usage[key] += tokens
	return nil
}

func (r *MemoryRepo) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]SessionMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgs := r.sessions[sessionID]
	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}
	start := len(msgs) - limit
	out := make([]SessionMessage, limit)
	copy(out, msgs[start:])
	return out, nil
}

func (r *MemoryRepo) AppendToSession(ctx context.Context, sessionID string, msg SessionMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	msg.SessionID = sessionID
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	r.sessions[sessionID] = append(r.sessions[sessionID], msg)
	return nil
}

func usageKey(userID string, date time.Time) string {
	return userID + "|" + date.Format("2006-01-02")
}
