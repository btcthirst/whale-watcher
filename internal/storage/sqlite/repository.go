package sqlite

import (
	"database/sql"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// InsertTransfers — batch insert
func (r *Repository) InsertTransfers(transfers []domain.Transfer) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO transfers(signature, slot, type, from_addr, to_addr, mint, amount)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range transfers {
		_, err := stmt.Exec(
			t.Signature,
			t.Slot,
			t.Type,
			t.From,
			t.To,
			t.Mint,
			t.Amount,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertWhaleEvents — batch insert
func (r *Repository) InsertWhaleEvents(events []domain.WhaleEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO whale_events(signature, slot, type, token, total_amount, total_usd)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		_, err := stmt.Exec(
			e.Signature,
			e.Slot,
			e.Type,
			e.Token,
			e.TotalAmount,
			e.TotalUSD,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
