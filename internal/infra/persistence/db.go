package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	// Register SQLite driver.
	_ "modernc.org/sqlite"
)

// OpenSQLite opens a SQLite database and applies schema.
func OpenSQLite(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	if err := applySchema(ctx, db); err != nil {
		return nil, err
	}
	return db, nil
}

func applySchema(ctx context.Context, db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash BLOB NOT NULL,
			created_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS items (
			id TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			type TEXT NOT NULL,
			payload BLOB NOT NULL,
			deleted INTEGER NOT NULL DEFAULT 0,
			version INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY (id, owner_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_items_owner_version ON items(owner_id, version);`,
		`ALTER TABLE items ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0;`,
	}
	for i, stmt := range schema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name: deleted") {
				continue
			}
			return fmt.Errorf("schema step %d: %w", i, err)
		}
	}
	return nil
}
