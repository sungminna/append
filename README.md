# Upbit Trading Platform

A comprehensive automated trading platform for Upbit cryptocurrency exchange built with Go and Gin framework.

## Features

### Market Data
- ✅ Real-time market data via REST API and WebSocket
- ✅ Historical candle data collection with rate limiting
- ✅ Multiple timeframe support (1m, 3m, 5m, 15m, 30m, 1h, 4h, 1d, 1w, 1M)
- ✅ Real-time orderbook streaming

### Trading
- Multi-user support with JWT authentication
- Multiple concurrent positions per user
- Split orders (partial buy/sell)
- Trailing stop loss
- Market and limit orders

### Data Storage
- **PostgreSQL**: User data, positions, orders, executions
- **ClickHouse**: Time-series financial data (candles, ticks, orderbook snapshots)

### API Integration
- ✅ Upbit Quotation API (market data)
- ✅ Upbit Exchange API (trading)
- ✅ Upbit WebSocket (real-time data)

## Architecture

```
├── cmd/server/           # Application entry point
├── internal/
│   ├── api/             # HTTP API layer
│   │   ├── handler/     # Request handlers
│   │   ├── middleware/  # Gin middlewares
│   │   └── router/      # Route definitions
│   ├── domain/          # Domain models
│   ├── service/         # Business logic
│   │   ├── scheduler/   # Data collection scheduler
│   │   ├── trading/     # Trading engine
│   │   └── position/    # Position management
│   └── upbit/           # Upbit API clients
│       ├── quotation/   # Market data client
│       ├── exchange/    # Trading client
│       └── websocket/   # WebSocket client
├── pkg/                 # Reusable packages
│   ├── ratelimit/       # Rate limiter
│   ├── jwt/             # JWT utilities
│   └── database/        # Database connections
└── migrations/          # Database migrations
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- ClickHouse 23+
- Docker & Docker Compose (optional)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/sungminna/upbit-trading-platform.git
cd upbit-trading-platform
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
export JWT_SECRET="your-secret-key"
export PORT="8080"
export POSTGRES_DSN="postgres://user:password@localhost:5432/upbit_trading?sslmode=disable"
export CLICKHOUSE_DSN="tcp://localhost:9000?database=upbit_trading"
```

4. Run database migrations:
```bash
# PostgreSQL
psql -U postgres -d upbit_trading -f migrations/postgres/001_init.sql

# ClickHouse
clickhouse-client --queries-file migrations/clickhouse/001_init.sql
```

5. Build and run:
```bash
go build -o bin/server ./cmd/server
./bin/server
```

### Using Docker Compose

```bash
docker-compose up -d
```

## API Endpoints

### Public Endpoints (No Authentication)

#### Get Markets
```bash
GET /api/v1/markets
```

#### Get Candles
```bash
GET /api/v1/candles/:market?interval=1m&count=100
```

Parameters:
- `interval`: 1m, 3m, 5m, 15m, 30m, 1h, 4h, 1d, 1w, 1M
- `count`: Number of candles (max 200)

#### Get Orderbook
```bash
GET /api/v1/orderbook/:market
```

#### Get Ticker
```bash
GET /api/v1/ticker?markets=KRW-BTC,KRW-ETH
```

### Protected Endpoints (Authentication Required)

#### User Management
```bash
POST /api/v1/auth/register
POST /api/v1/auth/login
GET /api/v1/users/me
```

#### Positions
```bash
GET /api/v1/positions
POST /api/v1/positions
GET /api/v1/positions/:id
DELETE /api/v1/positions/:id
```

#### Orders
```bash
POST /api/v1/orders
GET /api/v1/orders
GET /api/v1/orders/:id
DELETE /api/v1/orders/:id
```

## Testing

Run all tests:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run integration tests:
```bash
go test -v -tags=integration ./test/...
```

## Rate Limiting

The platform implements rate limiting according to Upbit's API limits:

- **Quotation API**: 30 requests/second
- **Exchange API**: 8 requests/second

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `JWT_SECRET` | JWT signing secret | - |
| `JWT_EXPIRY` | JWT token expiry | 24h |
| `POSTGRES_DSN` | PostgreSQL connection string | - |
| `CLICKHOUSE_DSN` | ClickHouse connection string | - |
| `UPBIT_ACCESS_KEY` | Upbit API access key | - |
| `UPBIT_SECRET_KEY` | Upbit API secret key | - |

## Development

### TDD Approach

This project follows Test-Driven Development (TDD):

1. Write tests first
2. Implement code to pass tests
3. Refactor while keeping tests green

### Adding New Features

1. Define domain models in `internal/domain/model/`
2. Create repository interfaces in `internal/domain/repository/`
3. Implement business logic in `internal/service/`
4. Add HTTP handlers in `internal/api/handler/`
5. Register routes in `internal/api/router/`

## Performance

- Handles 10,000+ concurrent WebSocket connections
- Sub-millisecond response times for cached data
- Efficient time-series data storage with ClickHouse
- Connection pooling for database operations

## Security

- JWT-based authentication
- Encrypted API key storage
- Rate limiting protection
- Input validation and sanitization
- CORS support

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new features
4. Implement the feature
5. Submit a pull request

## License

MIT License

## Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/)
- [Upbit API Documentation](https://docs.upbit.com/)
- [ClickHouse](https://clickhouse.com/)
- [PostgreSQL](https://www.postgresql.org/)

## Support

For issues and questions, please open an issue on GitHub.
