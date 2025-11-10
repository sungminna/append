package model

import (
	"time"

	"github.com/google/uuid"
)

// PositionStatus represents the status of a position
type PositionStatus string

const (
	PositionStatusOpen   PositionStatus = "open"
	PositionStatusClosed PositionStatus = "closed"
)

// PositionSide represents the side of a position (long/short)
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
)

// Position represents a trading position
type Position struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	UserID          uuid.UUID      `json:"user_id" db:"user_id"`
	Market          string         `json:"market" db:"market"`           // e.g., "KRW-BTC"
	Side            PositionSide   `json:"side" db:"side"`               // long or short
	Status          PositionStatus `json:"status" db:"status"`           // open or closed
	EntryPrice      float64        `json:"entry_price" db:"entry_price"` // Average entry price
	Quantity        float64        `json:"quantity" db:"quantity"`       // Current quantity
	InitialQuantity float64        `json:"initial_quantity" db:"initial_quantity"`
	RealizedPnL     float64        `json:"realized_pnl" db:"realized_pnl"` // Realized profit/loss
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	ClosedAt        *time.Time     `json:"closed_at,omitempty" db:"closed_at"`
}

// NewPosition creates a new position
func NewPosition(userID uuid.UUID, market string, side PositionSide, entryPrice, quantity float64) *Position {
	now := time.Now()
	return &Position{
		ID:              uuid.New(),
		UserID:          userID,
		Market:          market,
		Side:            side,
		Status:          PositionStatusOpen,
		EntryPrice:      entryPrice,
		Quantity:        quantity,
		InitialQuantity: quantity,
		RealizedPnL:     0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// CalculateUnrealizedPnL calculates unrealized profit/loss at current price
func (p *Position) CalculateUnrealizedPnL(currentPrice float64) float64 {
	if p.Side == PositionSideLong {
		return (currentPrice - p.EntryPrice) * p.Quantity
	}
	return (p.EntryPrice - currentPrice) * p.Quantity
}

// UpdateQuantity updates the position quantity and recalculates entry price
func (p *Position) UpdateQuantity(additionalQty, price float64) {
	// Recalculate average entry price
	totalValue := p.EntryPrice*p.Quantity + price*additionalQty
	p.Quantity += additionalQty
	p.EntryPrice = totalValue / p.Quantity
	p.UpdatedAt = time.Now()
}

// ReduceQuantity reduces the position quantity and updates realized PnL
func (p *Position) ReduceQuantity(qty, exitPrice float64) {
	pnl := (exitPrice - p.EntryPrice) * qty
	if p.Side == PositionSideShort {
		pnl = -pnl
	}

	p.RealizedPnL += pnl
	p.Quantity -= qty
	p.UpdatedAt = time.Now()

	if p.Quantity <= 0.00000001 { // Close position if quantity is negligible
		p.Status = PositionStatusClosed
		now := time.Now()
		p.ClosedAt = &now
	}
}
