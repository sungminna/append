package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	"github.com/sungminna/upbit-trading-platform/internal/service/position"
)

// PositionHandler handles position-related endpoints
type PositionHandler struct {
	positionService *position.Service
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(positionService *position.Service) *PositionHandler {
	return &PositionHandler{
		positionService: positionService,
	}
}

// CreatePosition creates a new position
// POST /api/v1/positions
func (h *PositionHandler) CreatePosition(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req position.CreatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.positionService.CreatePosition(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pos)
}

// GetPosition retrieves a position by ID
// GET /api/v1/positions/:id
func (h *PositionHandler) GetPosition(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	pos, err := h.positionService.GetPosition(c.Request.Context(), userID, positionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pos)
}

// GetPositions retrieves all positions for the user
// GET /api/v1/positions
func (h *PositionHandler) GetPositions(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Check if filtering for open positions only
	openOnly := c.Query("open") == "true"

	var positions interface{}
	if openOnly {
		positions, err = h.positionService.GetOpenPositions(c.Request.Context(), userID)
	} else {
		positions, err = h.positionService.GetUserPositions(c.Request.Context(), userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// ClosePositionRequest represents a request to close a position
type ClosePositionRequest struct {
	ExitPrice float64 `json:"exit_price" binding:"required"`
}

// ClosePosition closes a position
// POST /api/v1/positions/:id/close
func (h *PositionHandler) ClosePosition(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	var req ClosePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.positionService.ClosePosition(c.Request.Context(), userID, positionID, req.ExitPrice)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pos)
}

// CalculatePnL calculates PnL for a position
// GET /api/v1/positions/:id/pnl?current_price=xxx
func (h *PositionHandler) CalculatePnL(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	currentPriceStr := c.Query("current_price")
	currentPrice, err := strconv.ParseFloat(currentPriceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current_price parameter"})
		return
	}

	pnl, err := h.positionService.CalculatePositionPnL(c.Request.Context(), userID, positionID, currentPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unrealized_pnl": pnl,
		"current_price":  currentPrice,
	})
}

// GetPositionOrders retrieves all orders for a position
// GET /api/v1/positions/:id/orders
func (h *PositionHandler) GetPositionOrders(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	orders, err := h.positionService.GetPositionOrders(c.Request.Context(), userID, positionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
