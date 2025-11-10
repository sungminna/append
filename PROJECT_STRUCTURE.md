# Upbit Trading Platform Architecture

## Project Structure

```
.
├── cmd/
│   └── server/              # Application entry point
│       └── main.go
├── internal/
│   ├── api/                 # HTTP API layer
│   │   ├── handler/         # HTTP handlers
│   │   ├── middleware/      # Gin middlewares (auth, logging, etc.)
│   │   └── router/          # Route definitions
│   ├── domain/              # Domain models and interfaces
│   │   ├── model/           # Domain entities
│   │   └── repository/      # Repository interfaces
│   ├── service/             # Business logic layer
│   │   ├── auth/            # Authentication service
│   │   ├── trading/         # Trading execution engine
│   │   ├── position/        # Position management
│   │   └── scheduler/       # Candle collection scheduler
│   └── upbit/               # Upbit API clients
│       ├── quotation/       # Quotation API client
│       ├── exchange/        # Exchange API client
│       └── websocket/       # WebSocket client
├── pkg/                     # Reusable packages
│   ├── database/
│   │   ├── postgres/        # PostgreSQL connection
│   │   └── clickhouse/      # ClickHouse connection
│   ├── ratelimit/           # Rate limiter
│   └── jwt/                 # JWT utilities
├── config/                  # Configuration files
├── migrations/              # Database migrations
│   ├── postgres/
│   └── clickhouse/
└── test/                    # Integration tests

```

## Technology Stack

- **Framework**: Gin (HTTP router)
- **Language**: Go 1.21+
- **Databases**:
  - PostgreSQL: User data, positions, orders
  - ClickHouse: Time-series financial data (candles, ticks)
- **Authentication**: JWT
- **APIs**: Upbit Quotation, Exchange, WebSocket

## Key Features

### 1. Rate-Limited Data Collection
- Historical candle data collection with proper rate limiting
- Real-time candle updates via scheduler
- Parallel market data collection

### 2. User Management
- JWT-based authentication
- Multi-user support
- User-specific API key management for Upbit

### 3. Position Management
- Multiple concurrent positions per user
- Position lifecycle tracking
- PnL calculation

### 4. Advanced Trading Features
- Split orders (partial buy/sell)
- Trailing stop loss
- Various order types support
- Order execution engine

### 5. Real-time Market Data
- WebSocket integration for real-time orderbook
- Real-time price updates
- Market depth visualization

## Database Schema

### PostgreSQL (Transactional Data)
- users
- user_api_keys
- positions
- orders
- order_executions

### ClickHouse (Time-Series Data)
- candles (OHLCV data)
- ticks (trade data)
- orderbook_snapshots

## API Rate Limits (Upbit)

- Quotation API: 30 requests/sec (public)
- Exchange API: 8 requests/sec (private)
- Proper rate limiting implementation required

## Testing Strategy

- TDD approach for all components
- Unit tests for business logic
- Integration tests for API endpoints
- Mock Upbit API for testing
