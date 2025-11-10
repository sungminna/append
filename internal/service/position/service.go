package position

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
)

// Service handles position management
type Service struct {
	positionRepo repository.PositionRepository
	orderRepo    repository.OrderRepository
}

// NewService creates a new position service
func NewService(
	positionRepo repository.PositionRepository,
	orderRepo repository.OrderRepository,
) *Service {
	return &Service{
		positionRepo: positionRepo,
		orderRepo:    orderRepo,
	}
}

// CreatePositionRequest represents a request to create a position
type CreatePositionRequest struct {
	Market     string             `json:"market"`
	Side       model.PositionSide `json:"side"`
	EntryPrice float64            `json:"entry_price"`
	Quantity   float64            `json:"quantity"`
}

// CreatePosition creates a new position
func (s *Service) CreatePosition(ctx context.Context, userID uuid.UUID, req *CreatePositionRequest) (*model.Position, error) {
	// Check if there's already an open position for this market
	existingPosition, err := s.positionRepo.GetOpenPositionByMarket(ctx, userID, req.Market)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing position: %w", err)
	}

	if existingPosition != nil {
		return nil, fmt.Errorf("there is already an open position for market %s", req.Market)
	}

	// Create new position
	position := model.NewPosition(userID, req.Market, req.Side, req.EntryPrice, req.Quantity)
	if err := s.positionRepo.Create(ctx, position); err != nil {
		return nil, fmt.Errorf("failed to create position: %w", err)
	}

	return position, nil
}

// GetPosition retrieves a position by ID
func (s *Service) GetPosition(ctx context.Context, userID, positionID uuid.UUID) (*model.Position, error) {
	position, err := s.positionRepo.GetByID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	return position, nil
}

// GetUserPositions retrieves all positions for a user
func (s *Service) GetUserPositions(ctx context.Context, userID uuid.UUID) ([]*model.Position, error) {
	positions, err := s.positionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user positions: %w", err)
	}

	return positions, nil
}

// GetOpenPositions retrieves all open positions for a user
func (s *Service) GetOpenPositions(ctx context.Context, userID uuid.UUID) ([]*model.Position, error) {
	positions, err := s.positionRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get open positions: %w", err)
	}

	return positions, nil
}

// IncreasePosition increases the quantity of an existing position
func (s *Service) IncreasePosition(ctx context.Context, userID, positionID uuid.UUID, additionalQty, price float64) (*model.Position, error) {
	position, err := s.GetPosition(ctx, userID, positionID)
	if err != nil {
		return nil, err
	}

	if position.Status != model.PositionStatusOpen {
		return nil, fmt.Errorf("cannot increase a closed position")
	}

	// Update position quantity and recalculate entry price
	position.UpdateQuantity(additionalQty, price)

	if err := s.positionRepo.Update(ctx, position); err != nil {
		return nil, fmt.Errorf("failed to update position: %w", err)
	}

	return position, nil
}

// ReducePosition reduces the quantity of a position
func (s *Service) ReducePosition(ctx context.Context, userID, positionID uuid.UUID, qty, exitPrice float64) (*model.Position, error) {
	position, err := s.GetPosition(ctx, userID, positionID)
	if err != nil {
		return nil, err
	}

	if position.Status != model.PositionStatusOpen {
		return nil, fmt.Errorf("cannot reduce a closed position")
	}

	if qty > position.Quantity {
		return nil, fmt.Errorf("reduction quantity (%f) exceeds position quantity (%f)", qty, position.Quantity)
	}

	// Reduce position and update realized PnL
	position.ReduceQuantity(qty, exitPrice)

	if err := s.positionRepo.Update(ctx, position); err != nil {
		return nil, fmt.Errorf("failed to update position: %w", err)
	}

	return position, nil
}

// ClosePosition closes a position
func (s *Service) ClosePosition(ctx context.Context, userID, positionID uuid.UUID, exitPrice float64) (*model.Position, error) {
	position, err := s.GetPosition(ctx, userID, positionID)
	if err != nil {
		return nil, err
	}

	if position.Status != model.PositionStatusOpen {
		return nil, fmt.Errorf("position is already closed")
	}

	// Close entire position
	position.ReduceQuantity(position.Quantity, exitPrice)

	if err := s.positionRepo.Update(ctx, position); err != nil {
		return nil, fmt.Errorf("failed to update position: %w", err)
	}

	return position, nil
}

// CalculatePositionPnL calculates unrealized PnL for a position at current price
func (s *Service) CalculatePositionPnL(ctx context.Context, userID, positionID uuid.UUID, currentPrice float64) (float64, error) {
	position, err := s.GetPosition(ctx, userID, positionID)
	if err != nil {
		return 0, err
	}

	unrealizedPnL := position.CalculateUnrealizedPnL(currentPrice)
	return unrealizedPnL, nil
}

// GetPositionOrders retrieves all orders associated with a position
func (s *Service) GetPositionOrders(ctx context.Context, userID, positionID uuid.UUID) ([]*model.Order, error) {
	// Verify position belongs to user
	_, err := s.GetPosition(ctx, userID, positionID)
	if err != nil {
		return nil, err
	}

	orders, err := s.orderRepo.GetByPositionID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position orders: %w", err)
	}

	return orders, nil
}
