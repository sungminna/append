package model

import (
	"time"
)

// CandleInterval represents the time interval of a candle
type CandleInterval string

const (
	CandleInterval1m  CandleInterval = "1m"
	CandleInterval3m  CandleInterval = "3m"
	CandleInterval5m  CandleInterval = "5m"
	CandleInterval15m CandleInterval = "15m"
	CandleInterval30m CandleInterval = "30m"
	CandleInterval1h  CandleInterval = "1h"
	CandleInterval4h  CandleInterval = "4h"
	CandleInterval1d  CandleInterval = "1d"
	CandleInterval1w  CandleInterval = "1w"
	CandleInterval1M  CandleInterval = "1M"
)

// Candle represents OHLCV (Open, High, Low, Close, Volume) candlestick data
type Candle struct {
	Market          string         `json:"market"`            // e.g., "KRW-BTC"
	Interval        CandleInterval `json:"interval"`          // e.g., "1m", "5m", "1h"
	Timestamp       time.Time      `json:"timestamp"`         // Candle start time
	OpenPrice       float64        `json:"opening_price"`
	HighPrice       float64        `json:"high_price"`
	LowPrice        float64        `json:"low_price"`
	ClosePrice      float64        `json:"trade_price"`       // Last trade price
	Volume          float64        `json:"candle_acc_trade_volume"` // Accumulated trade volume
	AccTradePrice   float64        `json:"candle_acc_trade_price"`  // Accumulated trade price
	PrevClosingPrice float64       `json:"prev_closing_price,omitempty"`
	Change          string         `json:"change,omitempty"`  // RISE, EVEN, FALL
	ChangePrice     float64        `json:"change_price,omitempty"`
	ChangeRate      float64        `json:"change_rate,omitempty"`
}

// Tick represents a single trade tick
type Tick struct {
	Market           string    `json:"market"`
	TradeDateUTC     string    `json:"trade_date_utc"`
	TradeTimeUTC     string    `json:"trade_time_utc"`
	Timestamp        int64     `json:"timestamp"`
	TradePrice       float64   `json:"trade_price"`
	TradeVolume      float64   `json:"trade_volume"`
	PrevClosingPrice float64   `json:"prev_closing_price"`
	ChangePrice      float64   `json:"change_price"`
	AskBid           string    `json:"ask_bid"` // ASK or BID
	SequentialID     int64     `json:"sequential_id"`
}

// Orderbook represents the current orderbook (market depth)
type Orderbook struct {
	Market         string          `json:"market"`
	Timestamp      int64           `json:"timestamp"`
	TotalAskSize   float64         `json:"total_ask_size"`
	TotalBidSize   float64         `json:"total_bid_size"`
	OrderbookUnits []OrderbookUnit `json:"orderbook_units"`
}

// OrderbookUnit represents a single level in the orderbook
type OrderbookUnit struct {
	AskPrice float64 `json:"ask_price"`
	BidPrice float64 `json:"bid_price"`
	AskSize  float64 `json:"ask_size"`
	BidSize  float64 `json:"bid_size"`
}
