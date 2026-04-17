package domain

import (
	"context"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// AlertChannel абстрагує спосіб доставки сповіщення
type AlertChannel interface {
	Send(alert *WhaleAlert) error
	Close() error
}

// Storage визначає контракт для збереження історії алертів
type Storage interface {
	Save(alert *WhaleAlert) error
	GetRecent(limit int, since time.Time) ([]*WhaleAlert, error)
	Close() error
}

// RPCClient покриває HTTP/RPC запити до ноди Solana
type RPCClient interface {
	GetTransaction(ctx context.Context, sig solana.Signature, opts *rpc.GetTransactionOpts) (*rpc.GetTransactionResult, error)
	GetTokenAccountsByOwner(ctx context.Context, owner solana.PublicKey, opts *rpc.GetTokenAccountsByOwnerOpts) (*rpc.GetTokenAccountsByOwnerResult, error)
	GetLatestBlockhash(ctx context.Context, commitment rpc.CommitmentType) (*rpc.GetLatestBlockhashResult, error)
	GetAccountInfo(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error)
}

// PriceOracle інтерфейс для отримання цін токенів у USD
type PriceOracle interface {
	GetSOLPriceUSD(ctx context.Context) (float64, error)
	GetTokenPriceUSD(ctx context.Context, mint solana.PublicKey) (float64, error)
}
