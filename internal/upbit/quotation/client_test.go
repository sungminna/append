package quotation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
}

func TestClient_GetMarkets(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	markets, err := client.GetMarkets(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, markets)

	// Check that we have KRW markets
	hasKRWMarket := false
	for _, market := range markets {
		if len(market.Market) > 4 && market.Market[:3] == "KRW" {
			hasKRWMarket = true
			break
		}
	}
	assert.True(t, hasKRWMarket, "Should have at least one KRW market")
}

func TestClient_GetCandles(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	candles, err := client.GetCandles(ctx, "KRW-BTC", model.CandleInterval1m, 10)

	require.NoError(t, err)
	assert.NotEmpty(t, candles)
	assert.LessOrEqual(t, len(candles), 10)

	// Verify candle data structure
	for _, candle := range candles {
		assert.Equal(t, "KRW-BTC", candle.Market)
		assert.Equal(t, model.CandleInterval1m, candle.Interval)
		assert.Greater(t, candle.ClosePrice, 0.0)
		assert.Greater(t, candle.HighPrice, 0.0)
		assert.Greater(t, candle.LowPrice, 0.0)
		assert.Greater(t, candle.OpenPrice, 0.0)
		assert.GreaterOrEqual(t, candle.Volume, 0.0)
	}
}

func TestClient_GetCandleRange(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	to := time.Now()
	from := to.Add(-1 * time.Hour) // Get 1 hour of data

	candles, err := client.GetCandleRange(ctx, "KRW-BTC", model.CandleInterval1m, from, to)

	require.NoError(t, err)
	assert.NotEmpty(t, candles)

	// Verify candles are within the time range
	for _, candle := range candles {
		assert.True(t, candle.Timestamp.After(from) || candle.Timestamp.Equal(from))
		assert.True(t, candle.Timestamp.Before(to) || candle.Timestamp.Equal(to))
	}
}

func TestClient_GetOrderbook(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	orderbook, err := client.GetOrderbook(ctx, "KRW-BTC")

	require.NoError(t, err)
	assert.NotNil(t, orderbook)
	assert.Equal(t, "KRW-BTC", orderbook.Market)
	assert.NotEmpty(t, orderbook.OrderbookUnits)

	// Verify orderbook structure
	for _, unit := range orderbook.OrderbookUnits {
		assert.Greater(t, unit.AskPrice, 0.0)
		assert.Greater(t, unit.BidPrice, 0.0)
		assert.GreaterOrEqual(t, unit.AskSize, 0.0)
		assert.GreaterOrEqual(t, unit.BidSize, 0.0)
	}
}

func TestClient_GetTicker(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	tickers, err := client.GetTicker(ctx, []string{"KRW-BTC", "KRW-ETH"})

	require.NoError(t, err)
	assert.Len(t, tickers, 2)

	for _, ticker := range tickers {
		assert.NotEmpty(t, ticker.Market)
		assert.Greater(t, ticker.TradePrice, 0.0)
	}
}

// Integration test - requires actual API
func TestClient_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := NewClient()
	ctx := context.Background()

	// Make multiple requests rapidly
	for i := 0; i < 5; i++ {
		_, err := client.GetMarkets(ctx)
		assert.NoError(t, err)
	}
}
