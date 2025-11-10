package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
)

type candleRepository struct {
	conn driver.Conn
}

// NewCandleRepository creates a new ClickHouse candle repository
func NewCandleRepository(conn driver.Conn) repository.CandleRepository {
	return &candleRepository{conn: conn}
}

func (r *candleRepository) SaveCandles(ctx context.Context, candles []model.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO candles (
			market, interval, timestamp, opening_price, high_price, low_price,
			trade_price, candle_acc_trade_volume, candle_acc_trade_price,
			prev_closing_price, change, change_price, change_rate
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, candle := range candles {
		err := batch.Append(
			candle.Market,
			string(candle.Interval),
			candle.Timestamp,
			candle.OpenPrice,
			candle.HighPrice,
			candle.LowPrice,
			candle.ClosePrice,
			candle.Volume,
			candle.AccTradePrice,
			candle.PrevClosingPrice,
			candle.Change,
			candle.ChangePrice,
			candle.ChangeRate,
		)
		if err != nil {
			return fmt.Errorf("failed to append candle: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

func (r *candleRepository) GetLatestCandle(ctx context.Context, market string, interval model.CandleInterval) (*model.Candle, error) {
	query := `
		SELECT market, interval, timestamp, opening_price, high_price, low_price,
		       trade_price, candle_acc_trade_volume, candle_acc_trade_price,
		       prev_closing_price, change, change_price, change_rate
		FROM candles
		WHERE market = ? AND interval = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var candle model.Candle
	var intervalStr string
	err := r.conn.QueryRow(ctx, query, market, string(interval)).Scan(
		&candle.Market,
		&intervalStr,
		&candle.Timestamp,
		&candle.OpenPrice,
		&candle.HighPrice,
		&candle.LowPrice,
		&candle.ClosePrice,
		&candle.Volume,
		&candle.AccTradePrice,
		&candle.PrevClosingPrice,
		&candle.Change,
		&candle.ChangePrice,
		&candle.ChangeRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest candle: %w", err)
	}

	candle.Interval = model.CandleInterval(intervalStr)
	return &candle, nil
}

func (r *candleRepository) GetCandleRange(ctx context.Context, market string, interval model.CandleInterval, from, to time.Time) ([]model.Candle, error) {
	query := `
		SELECT market, interval, timestamp, opening_price, high_price, low_price,
		       trade_price, candle_acc_trade_volume, candle_acc_trade_price,
		       prev_closing_price, change, change_price, change_rate
		FROM candles
		WHERE market = ? AND interval = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := r.conn.Query(ctx, query, market, string(interval), from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query candles: %w", err)
	}
	defer rows.Close()

	return r.scanCandles(rows)
}

func (r *candleRepository) GetCandles(ctx context.Context, market string, interval model.CandleInterval, limit int) ([]model.Candle, error) {
	query := `
		SELECT market, interval, timestamp, opening_price, high_price, low_price,
		       trade_price, candle_acc_trade_volume, candle_acc_trade_price,
		       prev_closing_price, change, change_price, change_rate
		FROM candles
		WHERE market = ? AND interval = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.conn.Query(ctx, query, market, string(interval), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query candles: %w", err)
	}
	defer rows.Close()

	return r.scanCandles(rows)
}

func (r *candleRepository) scanCandles(rows driver.Rows) ([]model.Candle, error) {
	var candles []model.Candle
	for rows.Next() {
		var candle model.Candle
		var intervalStr string
		err := rows.Scan(
			&candle.Market,
			&intervalStr,
			&candle.Timestamp,
			&candle.OpenPrice,
			&candle.HighPrice,
			&candle.LowPrice,
			&candle.ClosePrice,
			&candle.Volume,
			&candle.AccTradePrice,
			&candle.PrevClosingPrice,
			&candle.Change,
			&candle.ChangePrice,
			&candle.ChangeRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan candle: %w", err)
		}
		candle.Interval = model.CandleInterval(intervalStr)
		candles = append(candles, candle)
	}

	return candles, nil
}
