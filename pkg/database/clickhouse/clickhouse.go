package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Config holds ClickHouse configuration
type Config struct {
	Addr     []string
	Database string
	Username string
	Password string
	Debug    bool
	TLS      *tls.Config
}

// NewConn creates a new ClickHouse connection
func NewConn(ctx context.Context, cfg *Config) (driver.Conn, error) {
	options := &clickhouse.Options{
		Addr: cfg.Addr,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Debug: cfg.Debug,
		Debugf: func(format string, v ...interface{}) {
			// Custom debug logger if needed
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      time.Second * 10,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	}

	if cfg.TLS != nil {
		options.TLS = cfg.TLS
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Ping to verify connection
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return conn, nil
}

// Close closes the ClickHouse connection
func Close(conn driver.Conn) error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}
