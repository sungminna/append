package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
)

// MarketHandler handles market-related endpoints
type MarketHandler struct {
	quotationClient *quotation.Client
}

// NewMarketHandler creates a new market handler
func NewMarketHandler(quotationClient *quotation.Client) *MarketHandler {
	return &MarketHandler{
		quotationClient: quotationClient,
	}
}

// GetMarkets returns all available markets
// GET /api/v1/markets
func (h *MarketHandler) GetMarkets(c *gin.Context) {
	markets, err := h.quotationClient.GetMarkets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, markets)
}

// GetCandles returns candle data for a market
// GET /api/v1/candles/:market
func (h *MarketHandler) GetCandles(c *gin.Context) {
	market := c.Param("market")
	interval := c.DefaultQuery("interval", string(model.CandleInterval1m))
	count := 100

	if countStr := c.Query("count"); countStr != "" {
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid count parameter"})
			return
		}
	}

	candles, err := h.quotationClient.GetCandles(c.Request.Context(), market, model.CandleInterval(interval), count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, candles)
}

// GetOrderbook returns orderbook data for a market
// GET /api/v1/orderbook/:market
func (h *MarketHandler) GetOrderbook(c *gin.Context) {
	market := c.Param("market")

	orderbook, err := h.quotationClient.GetOrderbook(c.Request.Context(), market)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orderbook)
}

// GetTicker returns ticker data for markets
// GET /api/v1/ticker?markets=KRW-BTC,KRW-ETH
func (h *MarketHandler) GetTicker(c *gin.Context) {
	marketsStr := c.Query("markets")
	if marketsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "markets parameter required"})
		return
	}

	markets := strings.Split(marketsStr, ",")
	tickers, err := h.quotationClient.GetTicker(c.Request.Context(), markets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tickers)
}
