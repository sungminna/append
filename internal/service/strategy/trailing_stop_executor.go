package strategy

import (
	"context"
	"fmt"
	"log"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// TrailingStopExecutor implements trailing stop strategy
type TrailingStopExecutor struct {
	tradingEngine *trading.Engine
}

// NewTrailingStopExecutor creates a new trailing stop executor
func NewTrailingStopExecutor(tradingEngine *trading.Engine) *TrailingStopExecutor {
	return &TrailingStopExecutor{
		tradingEngine: tradingEngine,
	}
}

func (e *TrailingStopExecutor) GetType() model.StrategyType {
	return model.StrategyTypeTrailingStop
}

func (e *TrailingStopExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.TrailingStopConfig)
	if !ok {
		return false, fmt.Errorf("invalid trailing stop config")
	}

	if config.TriggerPrice == nil {
		return false, nil
	}

	// Long position: trigger if price drops below trigger price
	// Short position: trigger if price rises above trigger price
	if position.Side == model.PositionSideLong {
		return currentPrice <= *config.TriggerPrice, nil
	}
	return currentPrice >= *config.TriggerPrice, nil
}

func (e *TrailingStopExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	log.Printf("Executing Trailing Stop for position %s at price %.8f", position.ID, currentPrice)

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
		return fmt.Errorf("failed to place trailing stop order: %w", err)
	}

	log.Printf("Trailing Stop order placed for position %s", position.ID)
	return nil
}

func (e *TrailingStopExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	config, ok := strategy.Config.(model.TrailingStopConfig)
	if !ok {
		return fmt.Errorf("invalid trailing stop config")
	}

	if position.Side == model.PositionSideLong {
		// Track highest price and update trigger
		if config.HighestPrice == nil || currentPrice > *config.HighestPrice {
			config.HighestPrice = &currentPrice
			triggerPrice := currentPrice * (1 - config.TrailPercent/100)
			config.TriggerPrice = &triggerPrice
			strategy.Config = config
			log.Printf("Updated trailing stop for position %s: highest=%.8f, trigger=%.8f",
				position.ID, currentPrice, triggerPrice)
		}
	} else {
		// Track lowest price and update trigger
		if config.LowestPrice == nil || currentPrice < *config.LowestPrice {
			config.LowestPrice = &currentPrice
			triggerPrice := currentPrice * (1 + config.TrailPercent/100)
			config.TriggerPrice = &triggerPrice
			strategy.Config = config
			log.Printf("Updated trailing stop for position %s: lowest=%.8f, trigger=%.8f",
				position.ID, currentPrice, triggerPrice)
		}
	}

	return nil
}
