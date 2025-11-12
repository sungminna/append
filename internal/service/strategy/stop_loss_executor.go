package strategy

import (
	"context"
	"fmt"
	"log"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// StopLossExecutor implements stop loss strategy
type StopLossExecutor struct {
	tradingEngine *trading.Engine
}

// NewStopLossExecutor creates a new stop loss executor
func NewStopLossExecutor(tradingEngine *trading.Engine) *StopLossExecutor {
	return &StopLossExecutor{
		tradingEngine: tradingEngine,
	}
}

func (e *StopLossExecutor) GetType() model.StrategyType {
	return model.StrategyTypeStopLoss
}

func (e *StopLossExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.StopLossConfig)
	if !ok {
		return false, fmt.Errorf("invalid stop loss config")
	}

	// Long position: trigger if price falls below stop price
	// Short position: trigger if price rises above stop price
	if position.Side == model.PositionSideLong {
		return currentPrice <= config.StopPrice, nil
	}
	return currentPrice >= config.StopPrice, nil
}

func (e *StopLossExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	log.Printf("Executing Stop Loss for position %s at price %.8f", position.ID, currentPrice)

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
		return fmt.Errorf("failed to place stop loss order: %w", err)
	}

	log.Printf("Stop Loss order placed for position %s", position.ID)
	return nil
}

func (e *StopLossExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	// Stop loss doesn't need updates
	return nil
}
