package strategy

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// TimeBasedExitExecutor implements time-based exit strategy
// 특정 시간이 되면 자동으로 포지션 청산
type TimeBasedExitExecutor struct {
	tradingEngine *trading.Engine
}

// NewTimeBasedExitExecutor creates a new time-based exit executor
func NewTimeBasedExitExecutor(tradingEngine *trading.Engine) *TimeBasedExitExecutor {
	return &TimeBasedExitExecutor{
		tradingEngine: tradingEngine,
	}
}

func (e *TimeBasedExitExecutor) GetType() model.StrategyType {
	return model.StrategyTypeTimeBasedExit
}

func (e *TimeBasedExitExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.TimeBasedExitConfig)
	if !ok {
		return false, fmt.Errorf("invalid time based exit config")
	}

	// Trigger if current time is past exit time
	return time.Now().After(config.ExitTime) || time.Now().Equal(config.ExitTime), nil
}

func (e *TimeBasedExitExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	config, ok := strategy.Config.(model.TimeBasedExitConfig)
	if !ok {
		return fmt.Errorf("invalid time based exit config")
	}

	log.Printf("Executing Time-Based Exit for position %s at scheduled time %s",
		position.ID, config.ExitTime.Format(time.RFC3339))

	// Determine order side (opposite of position side)
	orderSide := model.OrderSideAsk
	if position.Side == model.PositionSideShort {
		orderSide = model.OrderSideBid
	}

	// Place market order to close position
	orderReq := &trading.PlaceOrderRequest{
		Market:     position.Market,
		Side:       orderSide,
		Type:       model.OrderTypeMarket,
		Quantity:   position.Quantity,
		PositionID: &position.ID,
	}

	_, err := e.tradingEngine.PlaceOrder(ctx, position.UserID, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place time-based exit order: %w", err)
	}

	log.Printf("Time-Based Exit order placed for position %s", position.ID)
	return nil
}

func (e *TimeBasedExitExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	// Time-based exit doesn't need updates
	return nil
}
