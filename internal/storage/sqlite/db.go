// Package sqlite provides storage implementations for the whale watcher.
package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func NewDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// важливо для перформансу
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")
	_, _ = db.Exec("PRAGMA cache_size=100000;")
	_, _ = db.Exec("PRAGMA temp_store=MEMORY;")

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS transfers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		signature TEXT,
		slot INTEGER,
		type TEXT,
		from_addr TEXT,
		to_addr TEXT,
		mint TEXT,
		amount REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS whale_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		signature TEXT,
		slot INTEGER,
		type TEXT,
		token TEXT,
		total_amount REAL,
		total_usd REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_transfers_sig ON transfers(signature);
	CREATE INDEX IF NOT EXISTS idx_whale_sig ON whale_events(signature);
	CREATE INDEX IF NOT EXISTS idx_whale_usd ON whale_events(total_usd);
	`

	_, err := db.Exec(schema)
	return err
}
