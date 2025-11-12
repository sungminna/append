package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/strategy"
)

// StrategyHandler handles strategy-related endpoints
type StrategyHandler struct {
	strategyManager *strategy.Manager
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(strategyManager *strategy.Manager) *StrategyHandler {
	return &StrategyHandler{
		strategyManager: strategyManager,
	}
}

// CreateStrategyRequest represents a request to create a strategy
type CreateStrategyRequest struct {
	PositionID   uuid.UUID                `json:"position_id" binding:"required"`
	StrategyType model.StrategyType       `json:"strategy_type" binding:"required"`
	Config       json.RawMessage          `json:"config" binding:"required"`
}

// CreateStrategy creates a new trading strategy
// POST /api/v1/strategies
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req CreateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse config based on strategy type
	var config model.StrategyConfig
	switch req.StrategyType {
	case model.StrategyTypeStopLoss:
		var cfg model.StopLossConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stop loss config"})
			return
		}
		config = cfg

	case model.StrategyTypeTakeProfit:
		var cfg model.TakeProfitConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid take profit config"})
			return
		}
		config = cfg

	case model.StrategyTypeTrailingStop:
		var cfg model.TrailingStopConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trailing stop config"})
			return
		}
		config = cfg

	case model.StrategyTypeOCO:
		var cfg model.OCOConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OCO config"})
			return
		}
		config = cfg

	case model.StrategyTypeScaleOut:
		var cfg model.ScaleOutConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scale out config"})
			return
		}
		config = cfg

	case model.StrategyTypeTimeBasedExit:
		var cfg model.TimeBasedExitConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time based exit config"})
			return
		}
		config = cfg

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported strategy type"})
		return
	}

	strat, err := h.strategyManager.CreateStrategy(c.Request.Context(), userID, req.PositionID, req.StrategyType, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, strat)
}

// GetStrategy retrieves a strategy by ID
// GET /api/v1/strategies/:id
func (h *StrategyHandler) GetStrategy(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	strategyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy ID"})
		return
	}

	strat, err := h.strategyManager.GetStrategy(c.Request.Context(), userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strat)
}

// GetPositionStrategies retrieves all strategies for a position
// GET /api/v1/positions/:position_id/strategies
func (h *StrategyHandler) GetPositionStrategies(c *gin.Context) {
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

	strategies, err := h.strategyManager.GetPositionStrategies(c.Request.Context(), userID, positionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategies)
}

// UpdateStrategyConfigRequest represents a request to update strategy config
type UpdateStrategyConfigRequest struct {
	Config json.RawMessage `json:"config" binding:"required"`
}

// UpdateStrategyConfig updates a strategy's configuration
// PUT /api/v1/strategies/:id
func (h *StrategyHandler) UpdateStrategyConfig(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	strategyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy ID"})
		return
	}

	// First get the strategy to know its type
	existingStrategy, err := h.strategyManager.GetStrategy(c.Request.Context(), userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req UpdateStrategyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse config based on strategy type
	var config model.StrategyConfig
	switch existingStrategy.Type {
	case model.StrategyTypeStopLoss:
		var cfg model.StopLossConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stop loss config"})
			return
		}
		config = cfg

	case model.StrategyTypeTakeProfit:
		var cfg model.TakeProfitConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid take profit config"})
			return
		}
		config = cfg

	case model.StrategyTypeTrailingStop:
		var cfg model.TrailingStopConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trailing stop config"})
			return
		}
		config = cfg

	case model.StrategyTypeOCO:
		var cfg model.OCOConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OCO config"})
			return
		}
		config = cfg

	case model.StrategyTypeScaleOut:
		var cfg model.ScaleOutConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scale out config"})
			return
		}
		config = cfg

	case model.StrategyTypeTimeBasedExit:
		var cfg model.TimeBasedExitConfig
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time based exit config"})
			return
		}
		config = cfg

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported strategy type"})
		return
	}

	strat, err := h.strategyManager.UpdateStrategyConfig(c.Request.Context(), userID, strategyID, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strat)
}

// CancelStrategy cancels a strategy
// DELETE /api/v1/strategies/:id
func (h *StrategyHandler) CancelStrategy(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	strategyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy ID"})
		return
	}

	if err := h.strategyManager.CancelStrategy(c.Request.Context(), userID, strategyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "strategy cancelled"})
}
