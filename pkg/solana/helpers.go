// Package solana provides Solana-related utilities.
package solana

import (
	"crypto/rand"

	"github.com/gagliardetto/solana-go"
)

// NewTransactionID генерує унікальний ID для алерту
func NewTransactionID() string {
	var b [32]byte
	rand.Read(b[:])
	return solana.PublicKey(b[:]).String()
}
