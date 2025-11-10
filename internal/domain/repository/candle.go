package repository

import (
	"context"
	"time"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// CandleRepository defines methods for candle data access (ClickHouse)
type CandleRepository interface {
	SaveCandles(ctx context.Context, candles []model.Candle) error
	GetLatestCandle(ctx context.Context, market string, interval model.CandleInterval) (*model.Candle, error)
	GetCandleRange(ctx context.Context, market string, interval model.CandleInterval, from, to time.Time) ([]model.Candle, error)
	GetCandles(ctx context.Context, market string, interval model.CandleInterval, limit int) ([]model.Candle, error)
}
