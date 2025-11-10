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

type orderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository creates a new PostgreSQL order repository
func NewOrderRepository(pool *pgxpool.Pool) repository.OrderRepository {
	return &orderRepository{pool: pool}
}

func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
	query := `
		INSERT INTO orders (id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		order.ID, order.UserID, order.PositionID, order.Market, order.Side, order.Type,
		order.Price, order.Quantity, order.ExecutedQuantity, order.Status, order.ExchangeOrderID,
		order.CreatedAt, order.UpdatedAt, order.SubmittedAt, order.FilledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	query := `
		SELECT id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at
		FROM orders
		WHERE id = $1
	`
	var order model.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.PositionID, &order.Market, &order.Side, &order.Type,
		&order.Price, &order.Quantity, &order.ExecutedQuantity, &order.Status, &order.ExchangeOrderID,
		&order.CreatedAt, &order.UpdatedAt, &order.SubmittedAt, &order.FilledAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *orderRepository) GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at
		FROM orders
		WHERE position_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by position: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *orderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*model.Order, error) {
	query := `
		SELECT id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at
		FROM orders
		WHERE exchange_order_id = $1
	`
	var order model.Order
	err := r.pool.QueryRow(ctx, query, exchangeOrderID).Scan(
		&order.ID, &order.UserID, &order.PositionID, &order.Market, &order.Side, &order.Type,
		&order.Price, &order.Quantity, &order.ExecutedQuantity, &order.Status, &order.ExchangeOrderID,
		&order.CreatedAt, &order.UpdatedAt, &order.SubmittedAt, &order.FilledAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order by exchange ID: %w", err)
	}
	return &order, nil
}

func (r *orderRepository) GetPendingOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, position_id, market, side, order_type, price, quantity, executed_quantity, status, exchange_order_id, created_at, updated_at, submitted_at, filled_at
		FROM orders
		WHERE user_id = $1 AND status IN ('pending', 'submitted', 'partial')
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *orderRepository) Update(ctx context.Context, order *model.Order) error {
	query := `
		UPDATE orders
		SET position_id = $2, market = $3, side = $4, order_type = $5, price = $6,
		    quantity = $7, executed_quantity = $8, status = $9, exchange_order_id = $10,
		    updated_at = $11, submitted_at = $12, filled_at = $13
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		order.ID, order.PositionID, order.Market, order.Side, order.Type,
		order.Price, order.Quantity, order.ExecutedQuantity, order.Status, order.ExchangeOrderID,
		order.UpdatedAt, order.SubmittedAt, order.FilledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}

func (r *orderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	return nil
}

func (r *orderRepository) scanOrders(rows pgx.Rows) ([]*model.Order, error) {
	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.PositionID, &order.Market, &order.Side, &order.Type,
			&order.Price, &order.Quantity, &order.ExecutedQuantity, &order.Status, &order.ExchangeOrderID,
			&order.CreatedAt, &order.UpdatedAt, &order.SubmittedAt, &order.FilledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, &order)
	}
	return orders, nil
}

type orderExecutionRepository struct {
	pool *pgxpool.Pool
}

// NewOrderExecutionRepository creates a new PostgreSQL order execution repository
func NewOrderExecutionRepository(pool *pgxpool.Pool) repository.OrderExecutionRepository {
	return &orderExecutionRepository{pool: pool}
}

func (r *orderExecutionRepository) Create(ctx context.Context, execution *model.OrderExecution) error {
	query := `
		INSERT INTO order_executions (id, order_id, price, quantity, fee, total, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		execution.ID, execution.OrderID, execution.Price, execution.Quantity,
		execution.Fee, execution.Total, execution.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create order execution: %w", err)
	}
	return nil
}

func (r *orderExecutionRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.OrderExecution, error) {
	query := `
		SELECT id, order_id, price, quantity, fee, total, created_at
		FROM order_executions
		WHERE order_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.OrderExecution
	for rows.Next() {
		var execution model.OrderExecution
		err := rows.Scan(
			&execution.ID, &execution.OrderID, &execution.Price, &execution.Quantity,
			&execution.Fee, &execution.Total, &execution.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order execution: %w", err)
		}
		executions = append(executions, &execution)
	}

	return executions, nil
}
