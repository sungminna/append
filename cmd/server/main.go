package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sungminna/upbit-trading-platform/internal/api/router"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	chrepo "github.com/sungminna/upbit-trading-platform/internal/infrastructure/clickhouse"
	pgrepo "github.com/sungminna/upbit-trading-platform/internal/infrastructure/postgres"
	"github.com/sungminna/upbit-trading-platform/internal/service/auth"
	"github.com/sungminna/upbit-trading-platform/internal/service/position"
	"github.com/sungminna/upbit-trading-platform/internal/service/scheduler"
	"github.com/sungminna/upbit-trading-platform/internal/service/strategy"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
	trailing_stop "github.com/sungminna/upbit-trading-platform/internal/service/trailing_stop"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
	"github.com/sungminna/upbit-trading-platform/pkg/database/clickhouse"
	"github.com/sungminna/upbit-trading-platform/pkg/database/postgres"
	jwtpkg "github.com/sungminna/upbit-trading-platform/pkg/jwt"
)

func main() {
	ctx := context.Background()

	// Configuration from environment variables
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-this-in-production")
	port := getEnv("PORT", "8080")
	postgresDSN := getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/upbit_trading?sslmode=disable")
	clickhouseAddr := getEnv("CLICKHOUSE_ADDR", "localhost:9000")
	clickhouseDB := getEnv("CLICKHOUSE_DB", "upbit_trading")

	log.Println("Starting Upbit Trading Platform...")

	// Initialize PostgreSQL connection
	log.Println("Connecting to PostgreSQL...")
	pgPool, err := postgres.NewPool(ctx, &postgres.Config{
		DSN: postgresDSN,
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close(pgPool)
	log.Println("PostgreSQL connected successfully")

	// Initialize ClickHouse connection
	log.Println("Connecting to ClickHouse...")
	chConn, err := clickhouse.NewConn(ctx, &clickhouse.Config{
		Addr:     strings.Split(clickhouseAddr, ","),
		Database: clickhouseDB,
		Username: getEnv("CLICKHOUSE_USER", "default"),
		Password: getEnv("CLICKHOUSE_PASSWORD", ""),
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer clickhouse.Close(chConn)
	log.Println("ClickHouse connected successfully")

	// Initialize repositories
	userRepo := pgrepo.NewUserRepository(pgPool)
	userAPIKeyRepo := pgrepo.NewUserAPIKeyRepository(pgPool)
	positionRepo := pgrepo.NewPositionRepository(pgPool)
	orderRepo := pgrepo.NewOrderRepository(pgPool)
	executionRepo := pgrepo.NewOrderExecutionRepository(pgPool)
	trailingStopRepo := pgrepo.NewTrailingStopRepository(pgPool)
	strategyRepo := pgrepo.NewStrategyRepository(pgPool)
	candleRepo := chrepo.NewCandleRepository(chConn)

	// Initialize JWT manager
	jwtManager := jwtpkg.NewManager(jwtSecret, 24*time.Hour)

	// Initialize Upbit clients
	quotationClient := quotation.NewClient()

	// Initialize services
	authService := auth.NewService(userRepo, userAPIKeyRepo, jwtManager)
	positionService := position.NewService(positionRepo, orderRepo)
	tradingEngine := trading.NewEngine(orderRepo, executionRepo, positionRepo, userAPIKeyRepo)
	trailingStopService := trailing_stop.NewService(trailingStopRepo, positionRepo, quotationClient, tradingEngine)

	// Initialize strategy executors (OCP pattern)
	executorRegistry := strategy.NewExecutorRegistry()
	executorRegistry.Register(strategy.NewStopLossExecutor(tradingEngine))
	executorRegistry.Register(strategy.NewTakeProfitExecutor(tradingEngine))
	executorRegistry.Register(strategy.NewTrailingStopExecutor(tradingEngine))
	executorRegistry.Register(strategy.NewOCOExecutor(tradingEngine, strategyRepo))
	executorRegistry.Register(strategy.NewScaleOutExecutor(tradingEngine))
	executorRegistry.Register(strategy.NewTimeBasedExitExecutor(tradingEngine))

	// Initialize strategy manager
	strategyManager := strategy.NewManager(strategyRepo, positionRepo, quotationClient, executorRegistry)

	// Start trading engine
	log.Println("Starting trading engine...")
	if err := tradingEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start trading engine: %v", err)
	}

	// Start trailing stop service
	log.Println("Starting trailing stop service...")
	if err := trailingStopService.Start(ctx); err != nil {
		log.Fatalf("Failed to start trailing stop service: %v", err)
	}

	// Start strategy manager
	log.Println("Starting strategy manager...")
	if err := strategyManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start strategy manager: %v", err)
	}

	// Start candle collection scheduler (optional - for demonstration)
	markets := strings.Split(getEnv("MARKETS", "KRW-BTC,KRW-ETH"), ",")
	if len(markets) > 0 && markets[0] != "" {
		log.Printf("Starting candle collector for markets: %v", markets)
		candleCollector := scheduler.NewCandleCollector(
			quotationClient,
			candleRepo,
			markets,
			model.CandleInterval1m,
		)
		if err := candleCollector.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start candle collector: %v", err)
		}
		defer candleCollector.Stop()
	}

	// Setup router
	r := router.Setup(&router.Config{
		JWTSecret:           jwtSecret,
		JWTExpiry:           24 * time.Hour,
		QuotationClient:     quotationClient,
		AuthService:         authService,
		PositionService:     positionService,
		TradingEngine:       tradingEngine,
		TrailingStopService: trailingStopService,
		StrategyManager:     strategyManager,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %s", port)
		log.Printf("API endpoints available at http://localhost:%s/api/v1", port)
		log.Printf("Health check: http://localhost:%s/health", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Stop services
	log.Println("Stopping trading engine...")
	tradingEngine.Stop()

	log.Println("Stopping trailing stop service...")
	trailingStopService.Stop()

	log.Println("Stopping strategy manager...")
	strategyManager.Stop()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
