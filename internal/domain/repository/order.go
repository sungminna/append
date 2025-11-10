package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// OrderRepository defines methods for order data access
type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
	GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*model.Order, error)
	GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*model.Order, error)
	GetPendingOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
	Update(ctx context.Context, order *model.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// OrderExecutionRepository defines methods for order execution data access
type OrderExecutionRepository interface {
	Create(ctx context.Context, execution *model.OrderExecution) error
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.OrderExecution, error)
}
