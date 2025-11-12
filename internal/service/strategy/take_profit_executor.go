package strategy

import (
	"context"
	"fmt"
	"log"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// TakeProfitExecutor implements take profit strategy
type TakeProfitExecutor struct {
	tradingEngine *trading.Engine
}

// NewTakeProfitExecutor creates a new take profit executor
func NewTakeProfitExecutor(tradingEngine *trading.Engine) *TakeProfitExecutor {
	return &TakeProfitExecutor{
		tradingEngine: tradingEngine,
	}
}

func (e *TakeProfitExecutor) GetType() model.StrategyType {
	return model.StrategyTypeTakeProfit
}

func (e *TakeProfitExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.TakeProfitConfig)
	if !ok {
		return false, fmt.Errorf("invalid take profit config")
	}

	// Long position: trigger if price rises above target
	// Short position: trigger if price falls below target
	if position.Side == model.PositionSideLong {
		return currentPrice >= config.TargetPrice, nil
	}
	return currentPrice <= config.TargetPrice, nil
}

func (e *TakeProfitExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	log.Printf("Executing Take Profit for position %s at price %.8f", position.ID, currentPrice)

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
		return fmt.Errorf("failed to place take profit order: %w", err)
	}

	log.Printf("Take Profit order placed for position %s", position.ID)
	return nil
}

func (e *TakeProfitExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	// Take profit doesn't need updates
	return nil
}
