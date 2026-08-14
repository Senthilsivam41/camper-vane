package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// seedPreV3SQLite builds a DB that looks like a pre-v3 install (schema through v2 only).
func seedPreV3SQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO schema_migrations (version, name) VALUES (1, 'initial_schema'), (2, 'add_model_and_tokens_to_session_messages');

		CREATE TABLE user_configs (
			user_id TEXT PRIMARY KEY,
			daily_token_cap INTEGER NOT NULL DEFAULT 50000,
			routing_strategy TEXT NOT NULL DEFAULT 'simple',
			preferred_models TEXT NOT NULL DEFAULT '[]'
		);
		CREATE TABLE daily_usage (
			user_id TEXT NOT NULL,
			date_str TEXT NOT NULL,
			tokens INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, date_str)
		);
		CREATE TABLE session_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			model TEXT DEFAULT '',
			tokens_consumed INTEGER DEFAULT 0
		);
	`)
	if err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	return db
}

func TestMigrationV4BackfillSingleUser(t *testing.T) {
	raw := seedPreV3SQLite(t)

	_, err := raw.Exec(`
		INSERT INTO user_configs (user_id, daily_token_cap, routing_strategy, preferred_models)
		VALUES ('developer-1', 50000, 'simple', '[]');
		INSERT INTO daily_usage (user_id, date_str, tokens) VALUES
			('developer-1', '2026-08-10', 1200),
			('developer-1', '2026-08-11', 3400),
			('developer-1', '2026-08-12', 0);
		INSERT INTO session_messages (session_id, role, content, timestamp) VALUES
			('default-session', 'user', 'hello', '2026-08-10 09:00:00'),
			('default-session', 'assistant', 'hi', '2026-08-10 09:00:01');
	`)
	if err != nil {
		t.Fatalf("seed data: %v", err)
	}

	if err := RunMigrations(raw); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	var maxVer int
	if err := raw.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&maxVer); err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if maxVer != 4 {
		t.Fatalf("expected schema version 4, got %d", maxVer)
	}

	var eventCount int
	var eventSum int64
	if err := raw.QueryRow(`SELECT COUNT(*), COALESCE(SUM(tokens), 0) FROM usage_events`).Scan(&eventCount, &eventSum); err != nil {
		t.Fatalf("usage_events: %v", err)
	}
	if eventCount != 2 {
		t.Fatalf("expected 2 backfilled events (skip tokens=0), got %d", eventCount)
	}
	if eventSum != 4600 {
		t.Fatalf("expected token sum 4600, got %d", eventSum)
	}

	var noonStamp string
	if err := raw.QueryRow(`SELECT created_at FROM usage_events WHERE tokens = 1200`).Scan(&noonStamp); err != nil {
		t.Fatalf("created_at: %v", err)
	}
	ts, err := time.Parse(time.RFC3339, noonStamp)
	if err != nil {
		// SQLite may return "2006-01-02 15:04:05" depending on driver/storage.
		ts, err = time.Parse("2006-01-02 15:04:05", noonStamp)
		if err != nil {
			t.Fatalf("parse created_at %q: %v", noonStamp, err)
		}
		ts = ts.UTC()
	}
	if !ts.Equal(time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected UTC noon 2026-08-10, got %v (raw %q)", ts, noonStamp)
	}

	var emptyUserMsgs int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM session_messages WHERE user_id IS NULL OR user_id = ''`).Scan(&emptyUserMsgs); err != nil {
		t.Fatalf("empty user_id count: %v", err)
	}
	if emptyUserMsgs != 0 {
		t.Fatalf("expected all session_messages assigned, %d still empty", emptyUserMsgs)
	}

	var assigned string
	if err := raw.QueryRow(`SELECT DISTINCT user_id FROM session_messages`).Scan(&assigned); err != nil {
		t.Fatalf("assigned user: %v", err)
	}
	if assigned != "developer-1" {
		t.Fatalf("expected developer-1, got %q", assigned)
	}

	// Rolling window via repo API should see backfilled tokens.
	repo := &SQLiteRepo{db: raw}
	since := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	got, err := repo.GetUsageSince(context.Background(), "developer-1", since)
	if err != nil || got != 4600 {
		t.Fatalf("GetUsageSince: got %d err=%v want 4600", got, err)
	}
	sessions, err := repo.ListSessions(context.Background(), "developer-1", 10)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("ListSessions: got %d err=%v", len(sessions), err)
	}
}

func TestMigrationV4IdempotentWhenAlreadyPopulated(t *testing.T) {
	raw := seedPreV3SQLite(t)
	_, err := raw.Exec(`
		INSERT INTO user_configs (user_id) VALUES ('solo');
		INSERT INTO daily_usage (user_id, date_str, tokens) VALUES ('solo', '2026-08-01', 500);
		INSERT INTO session_messages (session_id, role, content, timestamp) VALUES ('s1', 'user', 'x', '2026-08-01 10:00:00');
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := RunMigrations(raw); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	// Simulate re-running apply logic after version recorded: delete version 4 marker and re-run.
	_, err = raw.Exec(`DELETE FROM schema_migrations WHERE version = 4`)
	if err != nil {
		t.Fatalf("delete v4 marker: %v", err)
	}
	if err := RunMigrations(raw); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var eventCount int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected still 1 usage_event after re-run, got %d", eventCount)
	}
}

func TestMigrationV4MultiUserLeavesEmptySessionUserID(t *testing.T) {
	raw := seedPreV3SQLite(t)
	_, err := raw.Exec(`
		INSERT INTO user_configs (user_id) VALUES ('alice'), ('bob');
		INSERT INTO session_messages (session_id, role, content, timestamp) VALUES
			('orphan', 'user', 'who owns me?', '2026-08-01 10:00:00');
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := RunMigrations(raw); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	var uid string
	if err := raw.QueryRow(`SELECT COALESCE(user_id, '') FROM session_messages WHERE session_id = 'orphan'`).Scan(&uid); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if uid != "" {
		t.Fatalf("expected empty user_id with multiple users, got %q", uid)
	}

	repo := &SQLiteRepo{db: raw}
	for _, u := range []string{"alice", "bob"} {
		sessions, err := repo.ListSessions(context.Background(), u, 10)
		if err != nil {
			t.Fatalf("ListSessions %s: %v", u, err)
		}
		if len(sessions) != 0 {
			t.Fatalf("expected orphan session hidden from %s, got %+v", u, sessions)
		}
	}
}

func TestMigrationV4SkipsDaysWithExistingEvents(t *testing.T) {
	raw := seedPreV3SQLite(t)
	_, err := raw.Exec(`
		INSERT INTO user_configs (user_id) VALUES ('u1');
		INSERT INTO daily_usage (user_id, date_str, tokens) VALUES
			('u1', '2026-08-01', 1000),
			('u1', '2026-08-02', 2000);
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Apply v3 schema only, then insert a post-v3 event for Aug 1 before v4.
	if err := RunMigrations(raw); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Reset to post-v3 / pre-v4 with a real event already present for Aug 1.
	_, err = raw.Exec(`
		DELETE FROM schema_migrations WHERE version = 4;
		DELETE FROM usage_events;
		INSERT INTO usage_events (user_id, tokens, created_at) VALUES ('u1', 50, '2026-08-01 15:30:00');
	`)
	if err != nil {
		t.Fatalf("prep mixed state: %v", err)
	}

	if err := RunMigrations(raw); err != nil {
		t.Fatalf("v4 migrate: %v", err)
	}

	var count int
	var sum int64
	if err := raw.QueryRow(`SELECT COUNT(*), COALESCE(SUM(tokens),0) FROM usage_events`).Scan(&count, &sum); err != nil {
		t.Fatalf("query: %v", err)
	}
	// Aug 1 skipped (already has event); Aug 2 backfilled with 2000.
	if count != 2 || sum != 2050 {
		t.Fatalf("expected 2 events totaling 2050, got count=%d sum=%d", count, sum)
	}
}

func TestFreshDBNeedsNoBackfill(t *testing.T) {
	repo, err := NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteRepo: %v", err)
	}
	defer repo.Close()

	var eventCount int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&eventCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("fresh DB should have 0 usage_events, got %d", eventCount)
	}
	var maxVer int
	if err := repo.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&maxVer); err != nil {
		t.Fatalf("version: %v", err)
	}
	if maxVer != 4 {
		t.Fatalf("expected version 4 on fresh DB, got %d", maxVer)
	}
}
