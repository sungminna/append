package model

import (
	"time"

	"github.com/google/uuid"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBid OrderSide = "bid" // Buy order
	OrderSideAsk OrderSide = "ask" // Sell order
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusSubmitted OrderStatus = "submitted" // Submitted to exchange
	OrderStatusPartial   OrderStatus = "partial"   // Partially filled
	OrderStatusFilled    OrderStatus = "filled"    // Completely filled
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusFailed    OrderStatus = "failed"
)

// Order represents a trading order
type Order struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	UserID           uuid.UUID   `json:"user_id" db:"user_id"`
	PositionID       *uuid.UUID  `json:"position_id,omitempty" db:"position_id"`
	Market           string      `json:"market" db:"market"`           // e.g., "KRW-BTC"
	Side             OrderSide   `json:"side" db:"side"`               // bid or ask
	Type             OrderType   `json:"type" db:"order_type"`         // limit or market
	Price            *float64    `json:"price,omitempty" db:"price"`   // Null for market orders
	Quantity         float64     `json:"quantity" db:"quantity"`       // Original quantity
	ExecutedQuantity float64     `json:"executed_quantity" db:"executed_quantity"`
	Status           OrderStatus `json:"status" db:"status"`
	ExchangeOrderID  *string     `json:"exchange_order_id,omitempty" db:"exchange_order_id"` // Upbit order UUID
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
	SubmittedAt      *time.Time  `json:"submitted_at,omitempty" db:"submitted_at"`
	FilledAt         *time.Time  `json:"filled_at,omitempty" db:"filled_at"`
}

// NewOrder creates a new order
func NewOrder(userID uuid.UUID, market string, side OrderSide, orderType OrderType, quantity float64, price *float64) *Order {
	now := time.Now()
	return &Order{
		ID:               uuid.New(),
		UserID:           userID,
		Market:           market,
		Side:             side,
		Type:             orderType,
		Price:            price,
		Quantity:         quantity,
		ExecutedQuantity: 0,
		Status:           OrderStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// IsComplete checks if the order is completely filled
func (o *Order) IsComplete() bool {
	return o.Status == OrderStatusFilled
}

// IsPending checks if the order is still pending or submitted
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusSubmitted
}

// UpdateExecution updates the order with execution information
func (o *Order) UpdateExecution(executedQty float64) {
	o.ExecutedQuantity += executedQty
	o.UpdatedAt = time.Now()

	if o.ExecutedQuantity >= o.Quantity {
		o.Status = OrderStatusFilled
		now := time.Now()
		o.FilledAt = &now
	} else if o.ExecutedQuantity > 0 {
		o.Status = OrderStatusPartial
	}
}

// OrderExecution represents a single execution (fill) of an order
type OrderExecution struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Price     float64   `json:"price" db:"price"`
	Quantity  float64   `json:"quantity" db:"quantity"`
	Fee       float64   `json:"fee" db:"fee"`
	Total     float64   `json:"total" db:"total"` // Price * Quantity
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NewOrderExecution creates a new order execution record
func NewOrderExecution(orderID uuid.UUID, price, quantity, fee float64) *OrderExecution {
	return &OrderExecution{
		ID:        uuid.New(),
		OrderID:   orderID,
		Price:     price,
		Quantity:  quantity,
		Fee:       fee,
		Total:     price * quantity,
		CreatedAt: time.Now(),
	}
}
