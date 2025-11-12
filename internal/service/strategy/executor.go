package strategy

import (
	"context"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// Executor defines the interface for strategy execution
// OCP: Open for extension, closed for modification
type Executor interface {
	// Check evaluates the strategy against current market conditions
	// Returns true if the strategy should be triggered
	Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error)

	// Execute performs the action when strategy is triggered
	Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error

	// Update updates the strategy state based on current market conditions
	Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error

	// GetType returns the strategy type this executor handles
	GetType() model.StrategyType
}

// ExecutorRegistry manages strategy executors
type ExecutorRegistry struct {
	executors map[model.StrategyType]Executor
}

// NewExecutorRegistry creates a new executor registry
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[model.StrategyType]Executor),
	}
}

// Register registers a strategy executor
func (r *ExecutorRegistry) Register(executor Executor) {
	r.executors[executor.GetType()] = executor
}

// Get retrieves an executor for a strategy type
func (r *ExecutorRegistry) Get(strategyType model.StrategyType) (Executor, bool) {
	executor, exists := r.executors[strategyType]
	return executor, exists
}

// GetAll returns all registered executors
func (r *ExecutorRegistry) GetAll() map[model.StrategyType]Executor {
	return r.executors
}
