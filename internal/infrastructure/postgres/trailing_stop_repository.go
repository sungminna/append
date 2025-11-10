package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
)

type trailingStopRepository struct {
	pool *pgxpool.Pool
}

// NewTrailingStopRepository creates a new PostgreSQL trailing stop repository
func NewTrailingStopRepository(pool *pgxpool.Pool) repository.TrailingStopRepository {
	return &trailingStopRepository{pool: pool}
}

func (r *trailingStopRepository) Create(ctx context.Context, ts *model.TrailingStop) error {
	query := `
		INSERT INTO trailing_stops (id, position_id, trail_percent, highest_price, lowest_price, trigger_price, is_active, created_at, updated_at, triggered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		ts.ID, ts.PositionID, ts.TrailPercent, ts.HighestPrice, ts.LowestPrice,
		ts.TriggerPrice, ts.IsActive, ts.CreatedAt, ts.UpdatedAt, ts.TriggeredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create trailing stop: %w", err)
	}
	return nil
}

func (r *trailingStopRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TrailingStop, error) {
	query := `
		SELECT id, position_id, trail_percent, highest_price, lowest_price, trigger_price, is_active, created_at, updated_at, triggered_at
		FROM trailing_stops
		WHERE id = $1
	`
	var ts model.TrailingStop
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ts.ID, &ts.PositionID, &ts.TrailPercent, &ts.HighestPrice, &ts.LowestPrice,
		&ts.TriggerPrice, &ts.IsActive, &ts.CreatedAt, &ts.UpdatedAt, &ts.TriggeredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("trailing stop not found")
		}
		return nil, fmt.Errorf("failed to get trailing stop: %w", err)
	}
	return &ts, nil
}

func (r *trailingStopRepository) GetByPositionID(ctx context.Context, positionID uuid.UUID) (*model.TrailingStop, error) {
	query := `
		SELECT id, position_id, trail_percent, highest_price, lowest_price, trigger_price, is_active, created_at, updated_at, triggered_at
		FROM trailing_stops
		WHERE position_id = $1 AND is_active = true
		LIMIT 1
	`
	var ts model.TrailingStop
	err := r.pool.QueryRow(ctx, query, positionID).Scan(
		&ts.ID, &ts.PositionID, &ts.TrailPercent, &ts.HighestPrice, &ts.LowestPrice,
		&ts.TriggerPrice, &ts.IsActive, &ts.CreatedAt, &ts.UpdatedAt, &ts.TriggeredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active trailing stop found is not an error
		}
		return nil, fmt.Errorf("failed to get trailing stop by position: %w", err)
	}
	return &ts, nil
}

func (r *trailingStopRepository) GetActiveTrailingStops(ctx context.Context) ([]*model.TrailingStop, error) {
	query := `
		SELECT id, position_id, trail_percent, highest_price, lowest_price, trigger_price, is_active, created_at, updated_at, triggered_at
		FROM trailing_stops
		WHERE is_active = true
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active trailing stops: %w", err)
	}
	defer rows.Close()

	var trailingStops []*model.TrailingStop
	for rows.Next() {
		var ts model.TrailingStop
		err := rows.Scan(
			&ts.ID, &ts.PositionID, &ts.TrailPercent, &ts.HighestPrice, &ts.LowestPrice,
			&ts.TriggerPrice, &ts.IsActive, &ts.CreatedAt, &ts.UpdatedAt, &ts.TriggeredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trailing stop: %w", err)
		}
		trailingStops = append(trailingStops, &ts)
	}

	return trailingStops, nil
}

func (r *trailingStopRepository) Update(ctx context.Context, ts *model.TrailingStop) error {
	query := `
		UPDATE trailing_stops
		SET trail_percent = $2, highest_price = $3, lowest_price = $4, trigger_price = $5,
		    is_active = $6, updated_at = $7, triggered_at = $8
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		ts.ID, ts.TrailPercent, ts.HighestPrice, ts.LowestPrice, ts.TriggerPrice,
		ts.IsActive, ts.UpdatedAt, ts.TriggeredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update trailing stop: %w", err)
	}
	return nil
}

func (r *trailingStopRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM trailing_stops WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete trailing stop: %w", err)
	}
	return nil
}
