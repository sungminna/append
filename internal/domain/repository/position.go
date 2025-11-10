package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// PositionRepository defines methods for position data access
type PositionRepository interface {
	Create(ctx context.Context, position *model.Position) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Position, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Position, error)
	GetOpenPositions(ctx context.Context, userID uuid.UUID) ([]*model.Position, error)
	GetOpenPositionByMarket(ctx context.Context, userID uuid.UUID, market string) (*model.Position, error)
	Update(ctx context.Context, position *model.Position) error
	Delete(ctx context.Context, id uuid.UUID) error
}
