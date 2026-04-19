package pricer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTokenSource struct {
	mock.Mock
}

func (m *MockTokenSource) Name() string {
	return m.Called().String(0)
}

func (m *MockTokenSource) GetTokenPricesUSD(ctx context.Context, mints []string) (map[string]float64, error) {
	args := m.Called(ctx, mints)
	return args.Get(0).(map[string]float64), args.Error(1)
}

func TestMultiPricer_GetTokenPricesUSD(t *testing.T) {
	ctx := context.Background()
	ttl := time.Minute

	t.Run("first source success", func(t *testing.T) {
		src1 := new(MockTokenSource)
		src2 := new(MockTokenSource)

		src1.On("Name").Return("src1")
		src1.On("GetTokenPricesUSD", ctx, []string{"mint1"}).Return(map[string]float64{"mint1": 1.5}, nil)

		mp := NewMultiPricer(ttl, src1, src2)
		prices, sources := mp.GetTokenPricesUSD(ctx, []string{"mint1"})

		assert.Equal(t, 1.5, prices["mint1"])
		assert.Equal(t, "src1", sources["mint1"])
		src2.AssertNotCalled(t, "GetTokenPricesUSD", mock.Anything, mock.Anything)
	})

	t.Run("fallback to second source", func(t *testing.T) {
		src1 := new(MockTokenSource)
		src2 := new(MockTokenSource)

		src1.On("Name").Return("src1")
		src1.On("GetTokenPricesUSD", ctx, []string{"mint2"}).Return(map[string]float64{}, nil) // src1 doesn't know

		src2.On("Name").Return("src2")
		src2.On("GetTokenPricesUSD", ctx, []string{"mint2"}).Return(map[string]float64{"mint2": 2.5}, nil)

		mp := NewMultiPricer(ttl, src1, src2)
		prices, sources := mp.GetTokenPricesUSD(ctx, []string{"mint2"})

		assert.Equal(t, 2.5, prices["mint2"])
		assert.Equal(t, "src2", sources["mint2"])
	})

	t.Run("error in first source fallbacks", func(t *testing.T) {
		src1 := new(MockTokenSource)
		src2 := new(MockTokenSource)

		src1.On("GetTokenPricesUSD", ctx, []string{"mint3"}).Return(map[string]float64{}, errors.New("fail"))

		src2.On("Name").Return("src2")
		src2.On("GetTokenPricesUSD", ctx, []string{"mint3"}).Return(map[string]float64{"mint3": 3.5}, nil)

		mp := NewMultiPricer(ttl, src1, src2)
		prices, sources := mp.GetTokenPricesUSD(ctx, []string{"mint3"})

		assert.Equal(t, 3.5, prices["mint3"])
		assert.Equal(t, "src2", sources["mint3"])
	})

	t.Run("caching works", func(t *testing.T) {
		src := new(MockTokenSource)
		src.On("Name").Return("src")
		src.On("GetTokenPricesUSD", ctx, []string{"mint4"}).Return(map[string]float64{"mint4": 4.5}, nil).Once()

		mp := NewMultiPricer(ttl, src)
		prices, _ := mp.GetTokenPricesUSD(ctx, []string{"mint4"})
		assert.Equal(t, 4.5, prices["mint4"])

		// Second call should use cache
		prices, _ = mp.GetTokenPricesUSD(ctx, []string{"mint4"})
		assert.Equal(t, 4.5, prices["mint4"])
		src.AssertExpectations(t)
	})
}
