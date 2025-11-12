package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
)

type strategyRepository struct {
	pool *pgxpool.Pool
}

// NewStrategyRepository creates a new PostgreSQL strategy repository
func NewStrategyRepository(pool *pgxpool.Pool) repository.StrategyRepository {
	return &strategyRepository{pool: pool}
}

func (r *strategyRepository) Create(ctx context.Context, strategy *model.Strategy) error {
	configJSON, err := json.Marshal(strategy.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO strategies (id, position_id, strategy_type, status, config, created_at, updated_at, triggered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = r.pool.Exec(ctx, query,
		strategy.ID, strategy.PositionID, strategy.Type, strategy.Status,
		configJSON, strategy.CreatedAt, strategy.UpdatedAt, strategy.TriggeredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create strategy: %w", err)
	}
	return nil
}

func (r *strategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Strategy, error) {
	query := `
		SELECT id, position_id, strategy_type, status, config, created_at, updated_at, triggered_at
		FROM strategies
		WHERE id = $1
	`
	var strategy model.Strategy
	var configJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&strategy.ID, &strategy.PositionID, &strategy.Type, &strategy.Status,
		&configJSON, &strategy.CreatedAt, &strategy.UpdatedAt, &strategy.TriggeredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("strategy not found")
		}
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	// Unmarshal config based on strategy type
	if err := r.unmarshalConfig(&strategy, configJSON); err != nil {
		return nil, err
	}

	return &strategy, nil
}

func (r *strategyRepository) GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*model.Strategy, error) {
	query := `
		SELECT id, position_id, strategy_type, status, config, created_at, updated_at, triggered_at
		FROM strategies
		WHERE position_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategies: %w", err)
	}
	defer rows.Close()

	return r.scanStrategies(rows)
}

func (r *strategyRepository) GetActiveStrategies(ctx context.Context) ([]*model.Strategy, error) {
	query := `
		SELECT id, position_id, strategy_type, status, config, created_at, updated_at, triggered_at
		FROM strategies
		WHERE status = 'active'
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active strategies: %w", err)
	}
	defer rows.Close()

	return r.scanStrategies(rows)
}

func (r *strategyRepository) GetActiveStrategiesByType(ctx context.Context, strategyType model.StrategyType) ([]*model.Strategy, error) {
	query := `
		SELECT id, position_id, strategy_type, status, config, created_at, updated_at, triggered_at
		FROM strategies
		WHERE status = 'active' AND strategy_type = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, strategyType)
	if err != nil {
		return nil, fmt.Errorf("failed to get active strategies by type: %w", err)
	}
	defer rows.Close()

	return r.scanStrategies(rows)
}

func (r *strategyRepository) Update(ctx context.Context, strategy *model.Strategy) error {
	configJSON, err := json.Marshal(strategy.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE strategies
		SET status = $2, config = $3, updated_at = $4, triggered_at = $5
		WHERE id = $1
	`
	_, err = r.pool.Exec(ctx, query,
		strategy.ID, strategy.Status, configJSON, strategy.UpdatedAt, strategy.TriggeredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update strategy: %w", err)
	}
	return nil
}

func (r *strategyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM strategies WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete strategy: %w", err)
	}
	return nil
}

func (r *strategyRepository) scanStrategies(rows pgx.Rows) ([]*model.Strategy, error) {
	var strategies []*model.Strategy
	for rows.Next() {
		var strategy model.Strategy
		var configJSON []byte
		err := rows.Scan(
			&strategy.ID, &strategy.PositionID, &strategy.Type, &strategy.Status,
			&configJSON, &strategy.CreatedAt, &strategy.UpdatedAt, &strategy.TriggeredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		// Unmarshal config based on strategy type
		if err := r.unmarshalConfig(&strategy, configJSON); err != nil {
			return nil, err
		}

		strategies = append(strategies, &strategy)
	}
	return strategies, nil
}

func (r *strategyRepository) unmarshalConfig(strategy *model.Strategy, configJSON []byte) error {
	switch strategy.Type {
	case model.StrategyTypeStopLoss:
		var config model.StopLossConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal stop loss config: %w", err)
		}
		strategy.Config = config
	case model.StrategyTypeTakeProfit:
		var config model.TakeProfitConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal take profit config: %w", err)
		}
		strategy.Config = config
	case model.StrategyTypeTrailingStop:
		var config model.TrailingStopConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal trailing stop config: %w", err)
		}
		strategy.Config = config
	case model.StrategyTypeOCO:
		var config model.OCOConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal OCO config: %w", err)
		}
		strategy.Config = config
	case model.StrategyTypeScaleOut:
		var config model.ScaleOutConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal scale out config: %w", err)
		}
		strategy.Config = config
	case model.StrategyTypeTimeBasedExit:
		var config model.TimeBasedExitConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return fmt.Errorf("failed to unmarshal time based exit config: %w", err)
		}
		strategy.Config = config
	default:
		return fmt.Errorf("unknown strategy type: %s", strategy.Type)
	}
	return nil
}
