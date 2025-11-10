package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	trailing_stop "github.com/sungminna/upbit-trading-platform/internal/service/trailing_stop"
)

// TrailingStopHandler handles trailing stop-related endpoints
type TrailingStopHandler struct {
	trailingStopService *trailing_stop.Service
}

// NewTrailingStopHandler creates a new trailing stop handler
func NewTrailingStopHandler(trailingStopService *trailing_stop.Service) *TrailingStopHandler {
	return &TrailingStopHandler{
		trailingStopService: trailingStopService,
	}
}

// CreateTrailingStop creates a new trailing stop
// POST /api/v1/trailing-stops
func (h *TrailingStopHandler) CreateTrailingStop(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req trailing_stop.CreateTrailingStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ts, err := h.trailingStopService.CreateTrailingStop(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ts)
}

// GetTrailingStop retrieves a trailing stop by position ID
// GET /api/v1/positions/:position_id/trailing-stop
func (h *TrailingStopHandler) GetTrailingStop(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positionID, err := uuid.Parse(c.Param("position_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	ts, err := h.trailingStopService.GetTrailingStop(c.Request.Context(), userID, positionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ts)
}

// UpdateTrailingPercentRequest represents a request to update trail percent
type UpdateTrailingPercentRequest struct {
	TrailPercent float64 `json:"trail_percent" binding:"required"`
}

// UpdateTrailingPercent updates the trail percent of a trailing stop
// PUT /api/v1/trailing-stops/:id
func (h *TrailingStopHandler) UpdateTrailingPercent(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	trailingStopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trailing stop ID"})
		return
	}

	var req UpdateTrailingPercentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ts, err := h.trailingStopService.UpdateTrailingPercent(c.Request.Context(), userID, trailingStopID, req.TrailPercent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ts)
}

// CancelTrailingStop cancels a trailing stop
// DELETE /api/v1/trailing-stops/:id
func (h *TrailingStopHandler) CancelTrailingStop(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	trailingStopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trailing stop ID"})
		return
	}

	if err := h.trailingStopService.CancelTrailingStop(c.Request.Context(), userID, trailingStopID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "trailing stop cancelled"})
}
