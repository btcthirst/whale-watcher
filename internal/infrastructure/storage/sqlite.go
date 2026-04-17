// Package storage provides storage implementations for the whale watcher.
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	_ "github.com/mattn/go-sqlite3"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type SQLiteStorage struct {
	db            *sql.DB
	retentionDays int
}

func NewSQLite(path string, retentionDays int) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS whale_alerts (
			id TEXT PRIMARY KEY,
			signature TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			from_addr TEXT NOT NULL,
			to_addr TEXT NOT NULL,
			amount_lamports INTEGER NOT NULL,
			amount_sol REAL NOT NULL,
			usd_value REAL DEFAULT 0,
			token_mint TEXT,
			token_symbol TEXT,
			program_id TEXT,
			slot INTEGER NOT NULL,
			commitment TEXT
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}

	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_ts ON whale_alerts(timestamp);")

	return &SQLiteStorage{db: db, retentionDays: retentionDays}, nil
}

func (s *SQLiteStorage) Save(alert *domain.WhaleAlert) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO whale_alerts
		(id, signature, timestamp, from_addr, to_addr, amount_lamports, amount_sol, usd_value, token_mint, token_symbol, program_id, slot, commitment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, alert.ID, alert.Signature, alert.Timestamp, alert.From.String(), alert.To.String(),
		alert.AmountLamports, alert.AmountSOL, alert.USDValue,
		optionalPK(alert.TokenMint), alert.TokenSymbol, alert.ProgramID.String(),
		alert.Slot, alert.Commitment)
	return err
}

func (s *SQLiteStorage) GetRecent(limit int, since time.Time) ([]*domain.WhaleAlert, error) {
	rows, err := s.db.Query(`
		SELECT id, signature, timestamp, from_addr, to_addr, amount_lamports, amount_sol, usd_value, token_mint, token_symbol, program_id, slot, commitment
		FROM whale_alerts WHERE timestamp > ? ORDER BY timestamp DESC LIMIT ?
	`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*domain.WhaleAlert
	for rows.Next() {
		var a domain.WhaleAlert
		var tmStr, fromStr, toStr, progStr string
		err := rows.Scan(&a.ID, &a.Signature, &a.Timestamp, &fromStr, &toStr,
			&a.AmountLamports, &a.AmountSOL, &a.USDValue, &tmStr, &a.TokenSymbol, &progStr, &a.Slot, &a.Commitment)
		if err != nil {
			return nil, err
		}

		a.From = solana.MustPublicKeyFromBase58(fromStr)
		a.To = solana.MustPublicKeyFromBase58(toStr)
		a.ProgramID = solana.MustPublicKeyFromBase58(progStr)
		if tmStr != "" {
			pk := solana.MustPublicKeyFromBase58(tmStr)
			a.TokenMint = &pk
		}
		alerts = append(alerts, &a)
	}
	return alerts, rows.Err()
}

func (s *SQLiteStorage) Close() error {
	// Clean old records
	if s.retentionDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
		_, _ = s.db.Exec("DELETE FROM whale_alerts WHERE timestamp < ?", cutoff)
	}
	return s.db.Close()
}

func optionalPK(pk *solana.PublicKey) string {
	if pk == nil {
		return ""
	}
	return pk.String()
}
