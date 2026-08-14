package db

import (
	"database/sql"
	"fmt"
	"log"
)

// applyV4Backfill restores pre-v3 data that migration v3 introduced without copying:
//
//  1. usage_events: one row per daily_usage row (tokens > 0) that has no existing
//     usage_events on that calendar date for the user. created_at is set to
//     UTC noon on date_str (approximation — daily_usage has no intra-day times).
//  2. session_messages.user_id: when empty and exactly one user_configs row exists,
//     assign that user_id. Multiple users → leave empty and log a warning.
//
// Idempotent: skips dates that already have events; only updates empty user_id.
func applyV4Backfill(tx *sql.Tx, dialect string) error {
	if err := backfillUsageEventsFromDailyUsage(tx, dialect); err != nil {
		return err
	}
	return backfillSessionMessageUserIDs(tx, dialect)
}

func backfillUsageEventsFromDailyUsage(tx *sql.Tx, dialect string) error {
	var q string
	switch dialect {
	case "postgres":
		// UTC noon on date_str; skip days that already have any usage_events.
		q = `
		INSERT INTO usage_events (user_id, tokens, created_at)
		SELECT d.user_id, d.tokens, (d.date_str || 'T12:00:00Z')::timestamptz
		FROM daily_usage d
		WHERE d.tokens > 0
		  AND NOT EXISTS (
		    SELECT 1 FROM usage_events e
		    WHERE e.user_id = d.user_id
		      AND (e.created_at AT TIME ZONE 'UTC')::date = d.date_str::date
		  )`
	default: // sqlite
		// Store RFC3339 UTC noon so created_at matches Go driver formatting used by IncrementDailyUsage.
		q = `
		INSERT INTO usage_events (user_id, tokens, created_at)
		SELECT d.user_id, d.tokens, d.date_str || 'T12:00:00Z'
		FROM daily_usage d
		WHERE d.tokens > 0
		  AND NOT EXISTS (
		    SELECT 1 FROM usage_events e
		    WHERE e.user_id = d.user_id
		      AND date(e.created_at) = d.date_str
		  )`
	}
	res, err := tx.Exec(q)
	if err != nil {
		return fmt.Errorf("v4 backfill usage_events: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n > 0 {
		log.Printf("migration v4: backfilled %d usage_events from daily_usage (created_at = UTC noon)", n)
	}
	return nil
}

func backfillSessionMessageUserIDs(tx *sql.Tx, dialect string) error {
	rows, err := tx.Query(`SELECT user_id FROM user_configs ORDER BY user_id`)
	if err != nil {
		return fmt.Errorf("v4 list user_configs: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("v4 scan user_id: %w", err)
		}
		users = append(users, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("v4 iterate user_configs: %w", err)
	}

	emptyPred := `(user_id IS NULL OR user_id = '')`
	var emptyCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM session_messages WHERE ` + emptyPred).Scan(&emptyCount); err != nil {
		return fmt.Errorf("v4 count empty session user_id: %w", err)
	}
	if emptyCount == 0 {
		return nil
	}

	switch len(users) {
	case 0:
		log.Printf("migration v4: %d session_messages have empty user_id but no user_configs; leaving unassigned", emptyCount)
		return nil
	case 1:
		var q string
		if dialect == "postgres" {
			q = `UPDATE session_messages SET user_id = $1 WHERE user_id IS NULL OR user_id = ''`
		} else {
			q = `UPDATE session_messages SET user_id = ? WHERE user_id IS NULL OR user_id = ''`
		}
		res, err := tx.Exec(q, users[0])
		if err != nil {
			return fmt.Errorf("v4 backfill session_messages.user_id: %w", err)
		}
		if n, err := res.RowsAffected(); err == nil {
			log.Printf("migration v4: assigned user_id %q to %d session_messages", users[0], n)
		}
		return nil
	default:
		log.Printf("migration v4: WARNING: %d session_messages have empty user_id and %d users in user_configs; leaving unassigned (do not invent ownership)", emptyCount, len(users))
		return nil
	}
}

func applyV4BackfillSQLite(tx *sql.Tx) error {
	return applyV4Backfill(tx, "sqlite")
}

func applyV4BackfillPostgres(tx *sql.Tx) error {
	return applyV4Backfill(tx, "postgres")
}
