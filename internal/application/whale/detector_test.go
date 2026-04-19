package whale

import (
	"context"
	"testing"

	"github.com/btcthirst/whale-watcher/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSOLPricer is a mock of SOLPricer interface
type MockSOLPricer struct {
	mock.Mock
}

func (m *MockSOLPricer) GetSOLPriceUSD(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

// MockTokenPricer is a mock of TokenPricer interface
type MockTokenPricer struct {
	mock.Mock
}

func (m *MockTokenPricer) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, map[string]string) {
	args := m.Called(ctx, mints)
	return args.Get(0).(map[string]float64), args.Get(1).(map[string]string)
}

func TestDetector_Detect(t *testing.T) {
	ctx := context.Background()
	solThreshold := 100.0
	usdThreshold := 50000.0
	tokenAmountThreshold := 1000000.0

	t.Run("SOL transfer above amount threshold", func(t *testing.T) {
		mockSOL := new(MockSOLPricer)
		mockToken := new(MockTokenPricer)

		mockSOL.On("GetSOLPriceUSD", ctx).Return(150.0, nil)

		detector := NewDetector(mockSOL, mockToken, solThreshold, usdThreshold, tokenAmountThreshold)

		transfers := []domain.Transfer{
			{
				Signature: "sig1",
				Slot:      100,
				Mint:      "So11111111111111111111111111111111111111112",
				Type:      domain.TransferSOL,
				Amount:    110.0, // Above 100 SOL
			},
		}

		events, err := detector.Detect(ctx, transfers)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "SOL", events[0].Type)
		assert.Equal(t, 110.0, events[0].TotalAmount)
		assert.Equal(t, 110.0*150.0, events[0].TotalUSD)
	})

	t.Run("SOL transfer above USD threshold", func(t *testing.T) {
		mockSOL := new(MockSOLPricer)
		mockToken := new(MockTokenPricer)

		mockSOL.On("GetSOLPriceUSD", ctx).Return(600.0, nil) // High SOL price

		detector := NewDetector(mockSOL, mockToken, solThreshold, usdThreshold, tokenAmountThreshold)

		transfers := []domain.Transfer{
			{
				Signature: "sig2",
				Slot:      101,
				Mint:      "So11111111111111111111111111111111111111112",
				Type:      domain.TransferSOL,
				Amount:    90.0, // Below 100 SOL, but 90*600 = 54000 > 50000 USD
			},
		}

		events, err := detector.Detect(ctx, transfers)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, 54000.0, events[0].TotalUSD)
	})

	t.Run("TOKEN transfer with known price above USD threshold", func(t *testing.T) {
		mockSOL := new(MockSOLPricer)
		mockToken := new(MockTokenPricer)

		mockSOL.On("GetSOLPriceUSD", ctx).Return(150.0, nil)
		mockToken.On("GetTokenPricesUSD", ctx, []string{"token1"}).Return(
			map[string]float64{"token1": 10.0},
			map[string]string{"token1": "jupiter"},
		)

		detector := NewDetector(mockSOL, mockToken, solThreshold, usdThreshold, tokenAmountThreshold)

		transfers := []domain.Transfer{
			{
				Signature: "sig3",
				Slot:      102,
				Mint:      "token1",
				Type:      domain.TransferToken,
				Amount:    6000.0, // 6000 * 10 = 60000 > 50000 USD
			},
		}

		events, err := detector.Detect(ctx, transfers)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "TOKEN", events[0].Type)
		assert.Equal(t, 60000.0, events[0].TotalUSD)
		assert.Equal(t, "jupiter", events[0].PriceSource)
	})

	t.Run("TOKEN transfer with unknown price above amount threshold", func(t *testing.T) {
		mockSOL := new(MockSOLPricer)
		mockToken := new(MockTokenPricer)

		mockSOL.On("GetSOLPriceUSD", ctx).Return(150.0, nil)
		mockToken.On("GetTokenPricesUSD", ctx, []string{"token2"}).Return(
			map[string]float64{},
			map[string]string{},
		)

		detector := NewDetector(mockSOL, mockToken, solThreshold, usdThreshold, tokenAmountThreshold)

		transfers := []domain.Transfer{
			{
				Signature: "sig4",
				Slot:      103,
				Mint:      "token2",
				Type:      domain.TransferToken,
				Amount:    1100000.0, // Above 1,000,000
			},
		}

		events, err := detector.Detect(ctx, transfers)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "TOKEN", events[0].Type)
		assert.Equal(t, "fallback", events[0].PriceSource)
	})

	t.Run("Aggregation of multiple transfers in one transaction", func(t *testing.T) {
		mockSOL := new(MockSOLPricer)
		mockToken := new(MockTokenPricer)

		mockSOL.On("GetSOLPriceUSD", ctx).Return(150.0, nil)

		detector := NewDetector(mockSOL, mockToken, solThreshold, usdThreshold, tokenAmountThreshold)

		transfers := []domain.Transfer{
			{
				Signature: "sig5",
				Slot:      104,
				Mint:      "So11111111111111111111111111111111111111112",
				Type:      domain.TransferSOL,
				Amount:    60.0,
			},
			{
				Signature: "sig5",
				Slot:      104,
				Mint:      "So11111111111111111111111111111111111111112",
				Type:      domain.TransferSOL,
				Amount:    50.0,
			},
		} // Total 110 SOL > 100 threshold

		events, err := detector.Detect(ctx, transfers)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, 110.0, events[0].TotalAmount)
	})
}
