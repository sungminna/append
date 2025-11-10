package trailing_stop

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
)

// Service handles trailing stop logic
type Service struct {
	trailingStopRepo repository.TrailingStopRepository
	positionRepo     repository.PositionRepository
	quotationClient  *quotation.Client
	tradingEngine    *trading.Engine
	mu               sync.RWMutex
	stopChan         chan struct{}
	isRunning        bool
}

// NewService creates a new trailing stop service
func NewService(
	trailingStopRepo repository.TrailingStopRepository,
	positionRepo repository.PositionRepository,
	quotationClient *quotation.Client,
	tradingEngine *trading.Engine,
) *Service {
	return &Service{
		trailingStopRepo: trailingStopRepo,
		positionRepo:     positionRepo,
		quotationClient:  quotationClient,
		tradingEngine:    tradingEngine,
		stopChan:         make(chan struct{}),
	}
}

// Start starts the trailing stop service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = true
	s.mu.Unlock()

	log.Println("Trailing stop service started")

	// Start monitoring goroutine
	go s.monitorTrailingStops(ctx)

	return nil
}

// Stop stops the trailing stop service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
	log.Println("Trailing stop service stopped")
}

// CreateTrailingStopRequest represents a request to create a trailing stop
type CreateTrailingStopRequest struct {
	PositionID   uuid.UUID `json:"position_id"`
	TrailPercent float64   `json:"trail_percent"` // e.g., 2.0 for 2%
}

// CreateTrailingStop creates a new trailing stop for a position
func (s *Service) CreateTrailingStop(ctx context.Context, userID uuid.UUID, req *CreateTrailingStopRequest) (*model.TrailingStop, error) {
	// Verify position exists and belongs to user
	position, err := s.positionRepo.GetByID(ctx, req.PositionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	if position.Status != model.PositionStatusOpen {
		return nil, fmt.Errorf("cannot create trailing stop for closed position")
	}

	// Check if trailing stop already exists
	existingTS, err := s.trailingStopRepo.GetByPositionID(ctx, req.PositionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing trailing stop: %w", err)
	}

	if existingTS != nil {
		return nil, fmt.Errorf("trailing stop already exists for this position")
	}

	// Validate trail percent
	if req.TrailPercent <= 0 || req.TrailPercent > 100 {
		return nil, fmt.Errorf("trail percent must be between 0 and 100")
	}

	// Create trailing stop
	ts := model.NewTrailingStop(req.PositionID, req.TrailPercent)

	// Get current price to initialize highest/lowest
	currentPrice, err := s.getCurrentPrice(ctx, position.Market)
	if err != nil {
		return nil, fmt.Errorf("failed to get current price: %w", err)
	}

	// Initialize price tracking based on position side
	ts.UpdatePrice(currentPrice, position.Side)

	if err := s.trailingStopRepo.Create(ctx, ts); err != nil {
		return nil, fmt.Errorf("failed to create trailing stop: %w", err)
	}

	return ts, nil
}

// GetTrailingStop retrieves a trailing stop by position ID
func (s *Service) GetTrailingStop(ctx context.Context, userID, positionID uuid.UUID) (*model.TrailingStop, error) {
	// Verify position belongs to user
	position, err := s.positionRepo.GetByID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	ts, err := s.trailingStopRepo.GetByPositionID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trailing stop: %w", err)
	}

	return ts, nil
}

// CancelTrailingStop cancels a trailing stop
func (s *Service) CancelTrailingStop(ctx context.Context, userID, trailingStopID uuid.UUID) error {
	ts, err := s.trailingStopRepo.GetByID(ctx, trailingStopID)
	if err != nil {
		return fmt.Errorf("failed to get trailing stop: %w", err)
	}

	// Verify position belongs to user
	position, err := s.positionRepo.GetByID(ctx, ts.PositionID)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return fmt.Errorf("unauthorized: position does not belong to user")
	}

	// Deactivate trailing stop
	ts.Deactivate()

	if err := s.trailingStopRepo.Update(ctx, ts); err != nil {
		return fmt.Errorf("failed to deactivate trailing stop: %w", err)
	}

	return nil
}

// monitorTrailingStops monitors all active trailing stops
func (s *Service) monitorTrailingStops(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkTrailingStops(context.Background())
		}
	}
}

// checkTrailingStops checks all active trailing stops
func (s *Service) checkTrailingStops(ctx context.Context) {
	// Get all active trailing stops
	trailingStops, err := s.trailingStopRepo.GetActiveTrailingStops(ctx)
	if err != nil {
		log.Printf("Failed to get active trailing stops: %v", err)
		return
	}

	for _, ts := range trailingStops {
		go s.checkTrailingStop(context.Background(), ts)
	}
}

// checkTrailingStop checks a single trailing stop
func (s *Service) checkTrailingStop(ctx context.Context, ts *model.TrailingStop) {
	// Get position
	position, err := s.positionRepo.GetByID(ctx, ts.PositionID)
	if err != nil {
		log.Printf("Failed to get position %s: %v", ts.PositionID, err)
		return
	}

	// Skip if position is closed
	if position.Status != model.PositionStatusOpen {
		ts.Deactivate()
		s.trailingStopRepo.Update(ctx, ts)
		return
	}

	// Get current price
	currentPrice, err := s.getCurrentPrice(ctx, position.Market)
	if err != nil {
		log.Printf("Failed to get current price for %s: %v", position.Market, err)
		return
	}

	// Update trailing stop with current price
	triggered := ts.UpdatePrice(currentPrice, position.Side)

	// Save updated trailing stop
	if err := s.trailingStopRepo.Update(ctx, ts); err != nil {
		log.Printf("Failed to update trailing stop %s: %v", ts.ID, err)
		return
	}

	// If triggered, execute exit order
	if triggered {
		log.Printf("Trailing stop triggered for position %s at price %.8f", position.ID, currentPrice)

		ts.Trigger()
		if err := s.trailingStopRepo.Update(ctx, ts); err != nil {
			log.Printf("Failed to update trailing stop status: %v", err)
		}

		// Place market sell order to close position
		orderReq := &trading.PlaceOrderRequest{
			Market:     position.Market,
			Side:       model.OrderSideAsk, // Sell to close long position
			Type:       model.OrderTypeMarket,
			Quantity:   position.Quantity,
			PositionID: &position.ID,
		}

		if position.Side == model.PositionSideShort {
			orderReq.Side = model.OrderSideBid // Buy to close short position
		}

		_, err := s.tradingEngine.PlaceOrder(ctx, position.UserID, orderReq)
		if err != nil {
			log.Printf("Failed to place trailing stop exit order: %v", err)
		} else {
			log.Printf("Trailing stop exit order placed for position %s", position.ID)
		}
	}
}

// getCurrentPrice gets the current price for a market
func (s *Service) getCurrentPrice(ctx context.Context, market string) (float64, error) {
	tickers, err := s.quotationClient.GetTicker(ctx, []string{market})
	if err != nil {
		return 0, fmt.Errorf("failed to get ticker: %w", err)
	}

	if len(tickers) == 0 {
		return 0, fmt.Errorf("no ticker data for market %s", market)
	}

	return tickers[0].TradePrice, nil
}

// UpdateTrailingPercent updates the trail percent of an active trailing stop
func (s *Service) UpdateTrailingPercent(ctx context.Context, userID, trailingStopID uuid.UUID, newPercent float64) (*model.TrailingStop, error) {
	ts, err := s.trailingStopRepo.GetByID(ctx, trailingStopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trailing stop: %w", err)
	}

	// Verify position belongs to user
	position, err := s.positionRepo.GetByID(ctx, ts.PositionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	if !ts.IsActive {
		return nil, fmt.Errorf("cannot update inactive trailing stop")
	}

	// Validate trail percent
	if newPercent <= 0 || newPercent > 100 {
		return nil, fmt.Errorf("trail percent must be between 0 and 100")
	}

	// Update trail percent
	ts.TrailPercent = newPercent
	ts.UpdatedAt = time.Now()

	// Recalculate trigger price
	if position.Side == model.PositionSideLong && ts.HighestPrice != nil {
		triggerPrice := *ts.HighestPrice * (1 - ts.TrailPercent/100)
		ts.TriggerPrice = &triggerPrice
	} else if position.Side == model.PositionSideShort && ts.LowestPrice != nil {
		triggerPrice := *ts.LowestPrice * (1 + ts.TrailPercent/100)
		ts.TriggerPrice = &triggerPrice
	}

	if err := s.trailingStopRepo.Update(ctx, ts); err != nil {
		return nil, fmt.Errorf("failed to update trailing stop: %w", err)
	}

	return ts, nil
}
