package db

import (
	"log"
	"os"
	"strings"
)

// Store is the pluggable persistence surface consumed by the API layer.
type Store interface {
	UserRepository
	SessionRepository
	Close() error
}

// NewStoreFromEnv selects SQLite (default) or PostgreSQL when DATABASE_URL is set.
func NewStoreFromEnv() (Store, error) {
	if url := strings.TrimSpace(os.Getenv("DATABASE_URL")); url != "" {
		log.Printf("db: using PostgreSQL via DATABASE_URL")
		return NewPostgresRepo(url)
	}

	dbPath := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	if dbPath == "" {
		dbPath = "camper_vane.db"
	}
	log.Printf("db: using SQLite at %s", dbPath)
	return NewSQLiteRepo(dbPath)
}

// Ensure compile-time interface satisfaction.
var (
	_ Store = (*SQLiteRepo)(nil)
	_ Store = (*PostgresRepo)(nil)
	_ Store = (*MemoryRepo)(nil)
)
