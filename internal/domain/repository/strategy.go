package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// StrategyRepository defines methods for strategy data access
type StrategyRepository interface {
	Create(ctx context.Context, strategy *model.Strategy) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Strategy, error)
	GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*model.Strategy, error)
	GetActiveStrategies(ctx context.Context) ([]*model.Strategy, error)
	GetActiveStrategiesByType(ctx context.Context, strategyType model.StrategyType) ([]*model.Strategy, error)
	Update(ctx context.Context, strategy *model.Strategy) error
	Delete(ctx context.Context, id uuid.UUID) error
}
