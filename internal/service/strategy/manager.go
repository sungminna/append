package strategy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
)

// Manager manages all trading strategies
type Manager struct {
	strategyRepo    repository.StrategyRepository
	positionRepo    repository.PositionRepository
	quotationClient *quotation.Client
	executorRegistry *ExecutorRegistry
	mu              sync.RWMutex
	stopChan        chan struct{}
	isRunning       bool
}

// NewManager creates a new strategy manager
func NewManager(
	strategyRepo repository.StrategyRepository,
	positionRepo repository.PositionRepository,
	quotationClient *quotation.Client,
	executorRegistry *ExecutorRegistry,
) *Manager {
	return &Manager{
		strategyRepo:     strategyRepo,
		positionRepo:     positionRepo,
		quotationClient:  quotationClient,
		executorRegistry: executorRegistry,
		stopChan:         make(chan struct{}),
	}
}

// Start starts the strategy manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return nil
	}
	m.isRunning = true
	m.mu.Unlock()

	log.Println("Strategy Manager started")

	// Start monitoring goroutine
	go m.monitorStrategies(ctx)

	return nil
}

// Stop stops the strategy manager
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	close(m.stopChan)
	m.isRunning = false
	log.Println("Strategy Manager stopped")
}

// CreateStrategy creates a new strategy
func (m *Manager) CreateStrategy(ctx context.Context, userID uuid.UUID, positionID uuid.UUID, strategyType model.StrategyType, config model.StrategyConfig) (*model.Strategy, error) {
	// Verify position exists and belongs to user
	position, err := m.positionRepo.GetByID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	if position.Status != model.PositionStatusOpen {
		return nil, fmt.Errorf("cannot create strategy for closed position")
	}

	// Validate strategy type is supported
	if _, exists := m.executorRegistry.Get(strategyType); !exists {
		return nil, fmt.Errorf("unsupported strategy type: %s", strategyType)
	}

	// Create strategy
	strategy := model.NewStrategy(positionID, strategyType)
	strategy.Config = config

	// Initialize strategy if needed (e.g., trailing stop)
	if strategyType == model.StrategyTypeTrailingStop {
		executor, _ := m.executorRegistry.Get(strategyType)
		currentPrice, err := m.getCurrentPrice(ctx, position.Market)
		if err != nil {
			return nil, fmt.Errorf("failed to get current price: %w", err)
		}
		if err := executor.Update(ctx, strategy, position, currentPrice); err != nil {
			return nil, fmt.Errorf("failed to initialize strategy: %w", err)
		}
	}

	if err := m.strategyRepo.Create(ctx, strategy); err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}

	log.Printf("Created %s strategy %s for position %s", strategyType, strategy.ID, positionID)
	return strategy, nil
}

// GetStrategy retrieves a strategy by ID
func (m *Manager) GetStrategy(ctx context.Context, userID, strategyID uuid.UUID) (*model.Strategy, error) {
	strategy, err := m.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	// Verify position belongs to user
	position, err := m.positionRepo.GetByID(ctx, strategy.PositionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: strategy does not belong to user")
	}

	return strategy, nil
}

// GetPositionStrategies retrieves all strategies for a position
func (m *Manager) GetPositionStrategies(ctx context.Context, userID, positionID uuid.UUID) ([]*model.Strategy, error) {
	// Verify position belongs to user
	position, err := m.positionRepo.GetByID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized: position does not belong to user")
	}

	strategies, err := m.strategyRepo.GetByPositionID(ctx, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategies: %w", err)
	}

	return strategies, nil
}

// CancelStrategy cancels a strategy
func (m *Manager) CancelStrategy(ctx context.Context, userID, strategyID uuid.UUID) error {
	strategy, err := m.GetStrategy(ctx, userID, strategyID)
	if err != nil {
		return err
	}

	if !strategy.IsActive() {
		return fmt.Errorf("strategy is not active")
	}

	strategy.Cancel()

	if err := m.strategyRepo.Update(ctx, strategy); err != nil {
		return fmt.Errorf("failed to cancel strategy: %w", err)
	}

	log.Printf("Cancelled strategy %s", strategyID)
	return nil
}

// UpdateStrategyConfig updates a strategy's configuration
func (m *Manager) UpdateStrategyConfig(ctx context.Context, userID, strategyID uuid.UUID, config model.StrategyConfig) (*model.Strategy, error) {
	strategy, err := m.GetStrategy(ctx, userID, strategyID)
	if err != nil {
		return nil, err
	}

	if !strategy.IsActive() {
		return nil, fmt.Errorf("cannot update inactive strategy")
	}

	strategy.Config = config
	strategy.UpdatedAt = time.Now()

	if err := m.strategyRepo.Update(ctx, strategy); err != nil {
		return nil, fmt.Errorf("failed to update strategy: %w", err)
	}

	log.Printf("Updated strategy %s configuration", strategyID)
	return strategy, nil
}

// monitorStrategies monitors all active strategies
func (m *Manager) monitorStrategies(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkStrategies(context.Background())
		}
	}
}

// checkStrategies checks all active strategies
func (m *Manager) checkStrategies(ctx context.Context) {
	strategies, err := m.strategyRepo.GetActiveStrategies(ctx)
	if err != nil {
		log.Printf("Failed to get active strategies: %v", err)
		return
	}

	for _, strategy := range strategies {
		go m.checkStrategy(context.Background(), strategy)
	}
}

// checkStrategy checks a single strategy
func (m *Manager) checkStrategy(ctx context.Context, strategy *model.Strategy) {
	// Get executor for this strategy type
	executor, exists := m.executorRegistry.Get(strategy.Type)
	if !exists {
		log.Printf("No executor found for strategy type: %s", strategy.Type)
		return
	}

	// Get position
	position, err := m.positionRepo.GetByID(ctx, strategy.PositionID)
	if err != nil {
		log.Printf("Failed to get position %s: %v", strategy.PositionID, err)
		return
	}

	// Skip if position is closed
	if position.Status != model.PositionStatusOpen {
		strategy.Complete()
		m.strategyRepo.Update(ctx, strategy)
		return
	}

	// Get current price
	currentPrice, err := m.getCurrentPrice(ctx, position.Market)
	if err != nil {
		log.Printf("Failed to get current price for %s: %v", position.Market, err)
		return
	}

	// Update strategy state (for strategies like trailing stop)
	if err := executor.Update(ctx, strategy, position, currentPrice); err != nil {
		log.Printf("Failed to update strategy %s: %v", strategy.ID, err)
		return
	}

	// Save updated strategy
	if err := m.strategyRepo.Update(ctx, strategy); err != nil {
		log.Printf("Failed to save strategy %s: %v", strategy.ID, err)
		return
	}

	// Check if strategy should be triggered
	shouldTrigger, err := executor.Check(ctx, strategy, position, currentPrice)
	if err != nil {
		log.Printf("Failed to check strategy %s: %v", strategy.ID, err)
		return
	}

	if !shouldTrigger {
		return
	}

	// Execute strategy
	log.Printf("Strategy %s triggered for position %s", strategy.ID, position.ID)

	if err := executor.Execute(ctx, strategy, position, currentPrice); err != nil {
		log.Printf("Failed to execute strategy %s: %v", strategy.ID, err)
		return
	}

	// Mark strategy as triggered
	strategy.Trigger()
	if err := m.strategyRepo.Update(ctx, strategy); err != nil {
		log.Printf("Failed to update strategy status: %v", err)
	}

	// For scale out, check if all levels are completed
	if strategy.Type == model.StrategyTypeScaleOut {
		config, ok := strategy.Config.(model.ScaleOutConfig)
		if ok {
			allExecuted := true
			for _, level := range config.Levels {
				if !level.Executed {
					allExecuted = false
					break
				}
			}
			if allExecuted {
				strategy.Complete()
				m.strategyRepo.Update(ctx, strategy)
			}
		}
	}
}

// getCurrentPrice gets the current price for a market
func (m *Manager) getCurrentPrice(ctx context.Context, market string) (float64, error) {
	tickers, err := m.quotationClient.GetTicker(ctx, []string{market})
	if err != nil {
		return 0, fmt.Errorf("failed to get ticker: %w", err)
	}

	if len(tickers) == 0 {
		return 0, fmt.Errorf("no ticker data for market %s", market)
	}

	return tickers[0].TradePrice, nil
}
