package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// TrailingStopRepository defines methods for trailing stop data access
type TrailingStopRepository interface {
	Create(ctx context.Context, ts *model.TrailingStop) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TrailingStop, error)
	GetByPositionID(ctx context.Context, positionID uuid.UUID) (*model.TrailingStop, error)
	GetActiveTrailingStops(ctx context.Context) ([]*model.TrailingStop, error)
	Update(ctx context.Context, ts *model.TrailingStop) error
	Delete(ctx context.Context, id uuid.UUID) error
}
