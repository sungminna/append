package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sungminna/upbit-trading-platform/internal/api/handler"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
	jwtpkg "github.com/sungminna/upbit-trading-platform/pkg/jwt"
)

// Config holds router configuration
type Config struct {
	JWTSecret      string
	JWTExpiry      time.Duration
	QuotationClient *quotation.Client
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

	// Public API endpoints (no authentication required)
	publicAPI := r.Group("/api/v1")
	{
		// Market data endpoints
		marketHandler := handler.NewMarketHandler(cfg.QuotationClient)
		publicAPI.GET("/markets", marketHandler.GetMarkets)
		publicAPI.GET("/candles/:market", marketHandler.GetCandles)
		publicAPI.GET("/orderbook/:market", marketHandler.GetOrderbook)
		publicAPI.GET("/ticker", marketHandler.GetTicker)
	}

	// Protected API endpoints (authentication required)
	protectedAPI := r.Group("/api/v1")
	protectedAPI.Use(middleware.AuthMiddleware(jwtManager))
	{
		// User endpoints would go here
		// Position endpoints would go here
		// Order endpoints would go here
	}

	return r
}
