package model

import (
	"time"

	"github.com/google/uuid"
)

// TrailingStop represents a trailing stop order
type TrailingStop struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	PositionID   uuid.UUID  `json:"position_id" db:"position_id"`
	TrailPercent float64    `json:"trail_percent" db:"trail_percent"` // Percentage to trail (e.g., 2.0 for 2%)
	HighestPrice *float64   `json:"highest_price,omitempty" db:"highest_price"`
	LowestPrice  *float64   `json:"lowest_price,omitempty" db:"lowest_price"`
	TriggerPrice *float64   `json:"trigger_price,omitempty" db:"trigger_price"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	TriggeredAt  *time.Time `json:"triggered_at,omitempty" db:"triggered_at"`
}

// NewTrailingStop creates a new trailing stop
func NewTrailingStop(positionID uuid.UUID, trailPercent float64) *TrailingStop {
	now := time.Now()
	return &TrailingStop{
		ID:           uuid.New(),
		PositionID:   positionID,
		TrailPercent: trailPercent,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdatePrice updates the highest/lowest price and calculates trigger price
func (ts *TrailingStop) UpdatePrice(currentPrice float64, positionSide PositionSide) bool {
	ts.UpdatedAt = time.Now()
	triggered := false

	if positionSide == PositionSideLong {
		// For long positions, track highest price and trigger on drop
		if ts.HighestPrice == nil || currentPrice > *ts.HighestPrice {
			ts.HighestPrice = &currentPrice
			triggerPrice := currentPrice * (1 - ts.TrailPercent/100)
			ts.TriggerPrice = &triggerPrice
		} else if ts.TriggerPrice != nil && currentPrice <= *ts.TriggerPrice {
			// Price dropped below trigger
			triggered = true
		}
	} else {
		// For short positions, track lowest price and trigger on rise
		if ts.LowestPrice == nil || currentPrice < *ts.LowestPrice {
			ts.LowestPrice = &currentPrice
			triggerPrice := currentPrice * (1 + ts.TrailPercent/100)
			ts.TriggerPrice = &triggerPrice
		} else if ts.TriggerPrice != nil && currentPrice >= *ts.TriggerPrice {
			// Price rose above trigger
			triggered = true
		}
	}

	return triggered
}

// Trigger marks the trailing stop as triggered
func (ts *TrailingStop) Trigger() {
	ts.IsActive = false
	now := time.Now()
	ts.TriggeredAt = &now
	ts.UpdatedAt = now
}

// Deactivate deactivates the trailing stop
func (ts *TrailingStop) Deactivate() {
	ts.IsActive = false
	ts.UpdatedAt = time.Now()
}
