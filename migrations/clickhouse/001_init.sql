-- ClickHouse Schema for Time-Series Financial Data

-- Candles table (OHLCV data)
CREATE TABLE IF NOT EXISTS candles (
    market String,
    interval String,
    timestamp DateTime,
    opening_price Float64,
    high_price Float64,
    low_price Float64,
    trade_price Float64,
    candle_acc_trade_volume Float64,
    candle_acc_trade_price Float64,
    prev_closing_price Float64,
    change String,
    change_price Float64,
    change_rate Float64,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (market, interval, timestamp)
SETTINGS index_granularity = 8192;

-- Ticks table (trade data)
CREATE TABLE IF NOT EXISTS ticks (
    market String,
    trade_date_utc String,
    trade_time_utc String,
    timestamp Int64,
    trade_price Float64,
    trade_volume Float64,
    prev_closing_price Float64,
    change_price Float64,
    ask_bid String,
    sequential_id Int64,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(toDateTime(timestamp / 1000))
ORDER BY (market, timestamp, sequential_id)
SETTINGS index_granularity = 8192;

-- Orderbook snapshots table
CREATE TABLE IF NOT EXISTS orderbook_snapshots (
    market String,
    timestamp Int64,
    total_ask_size Float64,
    total_bid_size Float64,
    orderbook_units String, -- JSON string of orderbook units
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(toDateTime(timestamp / 1000))
ORDER BY (market, timestamp)
SETTINGS index_granularity = 8192;

-- Real-time ticker data (for monitoring)
CREATE TABLE IF NOT EXISTS tickers (
    market String,
    trade_price Float64,
    opening_price Float64,
    high_price Float64,
    low_price Float64,
    prev_closing_price Float64,
    change String,
    change_price Float64,
    change_rate Float64,
    trade_volume Float64,
    acc_trade_volume Float64,
    acc_trade_price Float64,
    timestamp Int64,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(toDateTime(timestamp / 1000))
ORDER BY (market, timestamp)
SETTINGS index_granularity = 8192;

-- Materialized view for candle aggregation (1-minute candles)
CREATE MATERIALIZED VIEW IF NOT EXISTS candles_1m_mv
ENGINE = MergeTree()
PARTITION BY toYYYYMM(candle_time)
ORDER BY (market, candle_time)
AS SELECT
    market,
    toStartOfMinute(toDateTime(timestamp / 1000)) as candle_time,
    argMin(trade_price, timestamp) as opening_price,
    max(trade_price) as high_price,
    min(trade_price) as low_price,
    argMax(trade_price, timestamp) as trade_price,
    sum(trade_volume) as volume,
    sum(trade_price * trade_volume) as acc_trade_price
FROM ticks
GROUP BY market, candle_time;
