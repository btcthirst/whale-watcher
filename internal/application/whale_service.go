package application

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type WhaleService struct {
	logger     *zap.Logger
	config     *domain.WhaleDomain
	rpc        domain.RPCClient
	pricer     domain.PriceOracle
	tokenCache map[solana.PublicKey]*TokenMeta
	cacheMu    sync.RWMutex
}

type TokenMeta struct {
	Symbol   string
	Decimals uint8
}

func NewWhaleService(cfg *domain.WhaleDomain, rpc domain.RPCClient, pricer domain.PriceOracle, logger *zap.Logger) *WhaleService {
	return &WhaleService{
		logger:     logger,
		config:     cfg,
		rpc:        rpc,
		pricer:     pricer,
		tokenCache: make(map[solana.PublicKey]*TokenMeta),
	}
}

func (s *WhaleService) ProcessTransaction(ctx context.Context, sig solana.Signature) (*domain.WhaleAlert, error) {
	tx, err := s.rpc.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding: solana.EncodingJSONParsed,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch transaction: %w", err)
	}
	if tx == nil || tx.Transaction == nil || tx.Meta == nil {
		return nil, nil
	}

	parsedTx, err := tx.Transaction.GetParsedTransaction()
	if err != nil {
		return nil, fmt.Errorf("parse transaction: %w", err)
	}

	var alerts []*domain.WhaleAlert
	commitment := string(tx.ConfirmationStatus)

	for _, inst := range parsedTx.Message.Instructions {
		alert, err := s.analyzeInstruction(ctx, inst, tx.Slot, commitment)
		if err != nil {
			s.logger.Debug("instruction analysis skipped", zap.Error(err))
			continue
		}
		if alert != nil {
			alerts = append(alerts, alert)
		}
	}

	if len(alerts) == 0 {
		return nil, nil
	}

	largest := alerts[0]
	for _, a := range alerts {
		if a.AmountLamports > largest.AmountLamports || (a.TokenMint != nil && a.USDValue > largest.USDValue) {
			largest = a
		}
	}
	return largest, nil
}

func (s *WhaleService) analyzeInstruction(ctx context.Context, inst solana.ParsedInstruction, slot uint64, commitment string) (*domain.WhaleAlert, error) {
	if s.config.ExcludedPrograms[inst.ProgramId] {
		return nil, nil
	}

	if inst.ProgramId.Equals(token.ProgramID) {
		return s.analyzeTokenTransfer(ctx, inst, slot, commitment)
	}
	if inst.ProgramId.Equals(solana.SystemProgramID) {
		return s.analyzeSOLTransfer(inst, slot, commitment)
	}
	return nil, nil
}

func (s *WhaleService) analyzeSOLTransfer(inst solana.ParsedInstruction, slot uint64, commitment string) (*domain.WhaleAlert, error) {
	if inst.Name != "transfer" {
		return nil, nil
	}

	src, ok := inst.Info["source"].(string)
	if !ok {
		return nil, fmt.Errorf("missing source")
	}
	dst, ok := inst.Info["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("missing destination")
	}
	lam, ok := inst.Info["lamports"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing amount")
	}

	lamports := uint64(lam)
	if lamports < s.config.SOLThresholdLamports {
		return nil, nil
	}

	usd := 0.0
	if price, err := s.pricer.GetSOLPriceUSD(context.Background()); err == nil {
		usd = (float64(lamports) / solana.LAMPORTS_PER_SOL) * price
	}

	return &domain.WhaleAlert{
		ID:             generateID(),
		Signature:      inst.Signature.String(),
		Timestamp:      time.Now(),
		From:           solana.MustPublicKeyFromBase58(src),
		To:             solana.MustPublicKeyFromBase58(dst),
		AmountLamports: lamports,
		AmountSOL:      float64(lamports) / solana.LAMPORTS_PER_SOL,
		USDValue:       usd,
		ProgramID:      inst.ProgramId,
		Slot:           slot,
		Commitment:     commitment,
		Metadata:       inst.Info,
	}, nil
}

func (s *WhaleService) analyzeTokenTransfer(ctx context.Context, inst solana.ParsedInstruction, slot uint64, commitment string) (*domain.WhaleAlert, error) {
	if inst.Name != "transfer" && inst.Name != "transferChecked" {
		return nil, nil
	}

	mintStr, ok := inst.Info["mint"].(string)
	if !ok {
		return nil, fmt.Errorf("missing mint")
	}
	mint := solana.MustPublicKeyFromBase58(mintStr)

	amountStr := fmt.Sprintf("%v", inst.Info["amount"])
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	meta := s.getTokenMeta(ctx, mint)
	humanAmount := float64(amount) / math.Pow10(int(meta.Decimals))

	usd := 0.0
	if price, err := s.pricer.GetTokenPriceUSD(ctx, mint); err == nil {
		usd = price * humanAmount
	}

	if s.config.USDThreshold > 0 && usd < s.config.USDThreshold {
		return nil, nil
	}

	return &domain.WhaleAlert{
		ID:             generateID(),
		Signature:      inst.Signature.String(),
		Timestamp:      time.Now(),
		From:           solana.MustPublicKeyFromBase58(inst.Info["source"].(string)),
		To:             solana.MustPublicKeyFromBase58(inst.Info["destination"].(string)),
		AmountLamports: amount,
		USDValue:       usd,
		TokenMint:      &mint,
		TokenSymbol:    meta.Symbol,
		ProgramID:      inst.ProgramId,
		Slot:           slot,
		Commitment:     commitment,
		Metadata:       inst.Info,
	}, nil
}

func (s *WhaleService) getTokenMeta(_ context.Context, mint solana.PublicKey) *TokenMeta {
	s.cacheMu.RLock()
	if m, ok := s.tokenCache[mint]; ok {
		s.cacheMu.RUnlock()
		return m
	}
	s.cacheMu.RUnlock()

	meta := &TokenMeta{Symbol: "UNKNOWN", Decimals: 9}
	s.cacheMu.Lock()
	s.tokenCache[mint] = meta
	s.cacheMu.Unlock()
	return meta
}

func generateID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
