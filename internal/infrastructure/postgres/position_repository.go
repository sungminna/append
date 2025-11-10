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

type positionRepository struct {
	pool *pgxpool.Pool
}

// NewPositionRepository creates a new PostgreSQL position repository
func NewPositionRepository(pool *pgxpool.Pool) repository.PositionRepository {
	return &positionRepository{pool: pool}
}

func (r *positionRepository) Create(ctx context.Context, position *model.Position) error {
	query := `
		INSERT INTO positions (id, user_id, market, side, status, entry_price, quantity, initial_quantity, realized_pnl, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		position.ID, position.UserID, position.Market, position.Side, position.Status,
		position.EntryPrice, position.Quantity, position.InitialQuantity, position.RealizedPnL,
		position.CreatedAt, position.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create position: %w", err)
	}
	return nil
}

func (r *positionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Position, error) {
	query := `
		SELECT id, user_id, market, side, status, entry_price, quantity, initial_quantity, realized_pnl, created_at, updated_at, closed_at
		FROM positions
		WHERE id = $1
	`
	var position model.Position
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&position.ID, &position.UserID, &position.Market, &position.Side, &position.Status,
		&position.EntryPrice, &position.Quantity, &position.InitialQuantity, &position.RealizedPnL,
		&position.CreatedAt, &position.UpdatedAt, &position.ClosedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("position not found")
		}
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	return &position, nil
}

func (r *positionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Position, error) {
	query := `
		SELECT id, user_id, market, side, status, entry_price, quantity, initial_quantity, realized_pnl, created_at, updated_at, closed_at
		FROM positions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}
	defer rows.Close()

	return r.scanPositions(rows)
}

func (r *positionRepository) GetOpenPositions(ctx context.Context, userID uuid.UUID) ([]*model.Position, error) {
	query := `
		SELECT id, user_id, market, side, status, entry_price, quantity, initial_quantity, realized_pnl, created_at, updated_at, closed_at
		FROM positions
		WHERE user_id = $1 AND status = 'open'
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get open positions: %w", err)
	}
	defer rows.Close()

	return r.scanPositions(rows)
}

func (r *positionRepository) GetOpenPositionByMarket(ctx context.Context, userID uuid.UUID, market string) (*model.Position, error) {
	query := `
		SELECT id, user_id, market, side, status, entry_price, quantity, initial_quantity, realized_pnl, created_at, updated_at, closed_at
		FROM positions
		WHERE user_id = $1 AND market = $2 AND status = 'open'
		LIMIT 1
	`
	var position model.Position
	err := r.pool.QueryRow(ctx, query, userID, market).Scan(
		&position.ID, &position.UserID, &position.Market, &position.Side, &position.Status,
		&position.EntryPrice, &position.Quantity, &position.InitialQuantity, &position.RealizedPnL,
		&position.CreatedAt, &position.UpdatedAt, &position.ClosedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No open position found is not an error
		}
		return nil, fmt.Errorf("failed to get open position by market: %w", err)
	}
	return &position, nil
}

func (r *positionRepository) Update(ctx context.Context, position *model.Position) error {
	query := `
		UPDATE positions
		SET market = $2, side = $3, status = $4, entry_price = $5, quantity = $6,
		    initial_quantity = $7, realized_pnl = $8, updated_at = $9, closed_at = $10
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		position.ID, position.Market, position.Side, position.Status,
		position.EntryPrice, position.Quantity, position.InitialQuantity, position.RealizedPnL,
		position.UpdatedAt, position.ClosedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update position: %w", err)
	}
	return nil
}

func (r *positionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM positions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete position: %w", err)
	}
	return nil
}

func (r *positionRepository) scanPositions(rows pgx.Rows) ([]*model.Position, error) {
	var positions []*model.Position
	for rows.Next() {
		var position model.Position
		err := rows.Scan(
			&position.ID, &position.UserID, &position.Market, &position.Side, &position.Status,
			&position.EntryPrice, &position.Quantity, &position.InitialQuantity, &position.RealizedPnL,
			&position.CreatedAt, &position.UpdatedAt, &position.ClosedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, &position)
	}
	return positions, nil
}
