package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sungminna/upbit-trading-platform/internal/api/handler"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	"github.com/sungminna/upbit-trading-platform/internal/service/auth"
	"github.com/sungminna/upbit-trading-platform/internal/service/position"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
	trailing_stop "github.com/sungminna/upbit-trading-platform/internal/service/trailing_stop"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
	jwtpkg "github.com/sungminna/upbit-trading-platform/pkg/jwt"
)

// Config holds router configuration
type Config struct {
	JWTSecret           string
	JWTExpiry           time.Duration
	QuotationClient     *quotation.Client
	AuthService         *auth.Service
	PositionService     *position.Service
	TradingEngine       *trading.Engine
	TrailingStopService *trailing_stop.Service
}

// Setup sets up the Gin router
func Setup(cfg *Config) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// JWT manager
	jwtManager := jwtpkg.NewManager(cfg.JWTSecret, cfg.JWTExpiry)

	// Initialize handlers
	marketHandler := handler.NewMarketHandler(cfg.QuotationClient)
	userHandler := handler.NewUserHandler(cfg.AuthService)
	positionHandler := handler.NewPositionHandler(cfg.PositionService)
	orderHandler := handler.NewOrderHandler(cfg.TradingEngine)
	trailingStopHandler := handler.NewTrailingStopHandler(cfg.TrailingStopService)

	// Public API endpoints (no authentication required)
	publicAPI := r.Group("/api/v1")
	{
		// Market data endpoints
		publicAPI.GET("/markets", marketHandler.GetMarkets)
		publicAPI.GET("/candles/:market", marketHandler.GetCandles)
		publicAPI.GET("/orderbook/:market", marketHandler.GetOrderbook)
		publicAPI.GET("/ticker", marketHandler.GetTicker)

		// Auth endpoints
		publicAPI.POST("/auth/register", userHandler.Register)
		publicAPI.POST("/auth/login", userHandler.Login)
	}

	// Protected API endpoints (authentication required)
	protectedAPI := r.Group("/api/v1")
	protectedAPI.Use(middleware.AuthMiddleware(jwtManager))
	{
		// User endpoints
		protectedAPI.GET("/users/me", userHandler.GetMe)
		protectedAPI.POST("/users/api-keys", userHandler.AddAPIKey)
		protectedAPI.GET("/users/api-keys/active", userHandler.GetActiveAPIKey)
		protectedAPI.DELETE("/users/api-keys/:id", userHandler.DeactivateAPIKey)

		// Position endpoints
		protectedAPI.POST("/positions", positionHandler.CreatePosition)
		protectedAPI.GET("/positions", positionHandler.GetPositions)
		protectedAPI.GET("/positions/:id", positionHandler.GetPosition)
		protectedAPI.POST("/positions/:id/close", positionHandler.ClosePosition)
		protectedAPI.GET("/positions/:id/pnl", positionHandler.CalculatePnL)
		protectedAPI.GET("/positions/:id/orders", positionHandler.GetPositionOrders)

		// Order endpoints
		protectedAPI.POST("/orders", orderHandler.PlaceOrder)
		protectedAPI.GET("/orders", orderHandler.GetOrders)
		protectedAPI.GET("/orders/:id", orderHandler.GetOrder)
		protectedAPI.DELETE("/orders/:id", orderHandler.CancelOrder)

		// Trailing stop endpoints
		protectedAPI.POST("/trailing-stops", trailingStopHandler.CreateTrailingStop)
		protectedAPI.GET("/positions/:position_id/trailing-stop", trailingStopHandler.GetTrailingStop)
		protectedAPI.PUT("/trailing-stops/:id", trailingStopHandler.UpdateTrailingPercent)
		protectedAPI.DELETE("/trailing-stops/:id", trailingStopHandler.CancelTrailingStop)
	}

	return r
}
