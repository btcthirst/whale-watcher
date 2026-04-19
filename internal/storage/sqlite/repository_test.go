package sqlite

import (
	"database/sql"
	"testing"

	"github.com/btcthirst/whale-watcher/internal/domain"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func TestRepository_InsertTransfers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	transfers := []domain.Transfer{
		{
			Signature: "sig1",
			Slot:      100,
			Type:      domain.TransferSOL,
			From:      "from1",
			To:        "to1",
			Amount:    10.5,
		},
	}

	err := repo.InsertTransfers(transfers)
	assert.NoError(t, err)

	// Verify insertion
	var count int
	err = db.QueryRow("SELECT count(*) FROM transfers").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRepository_InsertWhaleEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	events := []domain.WhaleEvent{
		{
			Signature:   "sig2",
			Slot:        101,
			Type:        "SOL",
			Token:       "So11111111111111111111111111111111111111112",
			TotalAmount: 110.0,
			TotalUSD:    16500.0,
		},
	}

	err := repo.InsertWhaleEvents(events)
	assert.NoError(t, err)

	// Verify insertion
	var count int
	err = db.QueryRow("SELECT count(*) FROM whale_events").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
