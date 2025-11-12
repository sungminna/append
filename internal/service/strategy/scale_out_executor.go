package strategy

import (
	"context"
	"fmt"
	"log"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// ScaleOutExecutor implements scale out strategy (분할 청산)
// 여러 가격 레벨에서 순차적으로 포지션을 청산
type ScaleOutExecutor struct {
	tradingEngine *trading.Engine
}

// NewScaleOutExecutor creates a new scale out executor
func NewScaleOutExecutor(tradingEngine *trading.Engine) *ScaleOutExecutor {
	return &ScaleOutExecutor{
		tradingEngine: tradingEngine,
	}
}

func (e *ScaleOutExecutor) GetType() model.StrategyType {
	return model.StrategyTypeScaleOut
}

func (e *ScaleOutExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.ScaleOutConfig)
	if !ok {
		return false, fmt.Errorf("invalid scale out config")
	}

	// Check if any level should be triggered
	for i := range config.Levels {
		if config.Levels[i].Executed {
			continue
		}

		// Long position: trigger when price reaches or exceeds level
		// Short position: trigger when price reaches or falls below level
		if position.Side == model.PositionSideLong {
			if currentPrice >= config.Levels[i].Price {
				return true, nil
			}
		} else {
			if currentPrice <= config.Levels[i].Price {
				return true, nil
			}
		}
	}

	return false, nil
}

func (e *ScaleOutExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	config, ok := strategy.Config.(model.ScaleOutConfig)
	if !ok {
		return fmt.Errorf("invalid scale out config")
	}

	// Find which level(s) to execute
	for i := range config.Levels {
		if config.Levels[i].Executed {
			continue
		}

		shouldExecute := false
		if position.Side == model.PositionSideLong {
			shouldExecute = currentPrice >= config.Levels[i].Price
		} else {
			shouldExecute = currentPrice <= config.Levels[i].Price
		}

		if !shouldExecute {
			continue
		}

		// Calculate quantity to sell based on percentage
		quantity := position.Quantity * (config.Levels[i].Percentage / 100)

		log.Printf("Executing Scale Out level %d for position %s: %.2f%% at price %.8f",
			i+1, position.ID, config.Levels[i].Percentage, currentPrice)

		// Determine order side (opposite of position side)
		orderSide := model.OrderSideAsk
		if position.Side == model.PositionSideShort {
			orderSide = model.OrderSideBid
		}

		// Place market order to partially close position
		orderReq := &trading.PlaceOrderRequest{
			Market:     position.Market,
			Side:       orderSide,
			Type:       model.OrderTypeMarket,
			Quantity:   quantity,
			PositionID: &position.ID,
		}

		_, err := e.tradingEngine.PlaceOrder(ctx, position.UserID, orderReq)
		if err != nil {
			log.Printf("Failed to place scale out order for level %d: %v", i+1, err)
			continue
		}

		// Mark level as executed
		config.Levels[i].Executed = true
		strategy.Config = config

		log.Printf("Scale Out order placed for position %s, level %d", position.ID, i+1)
	}

	// Check if all levels are executed
	allExecuted := true
	for _, level := range config.Levels {
		if !level.Executed {
			allExecuted = false
			break
		}
	}

	// If all levels executed, mark strategy as completed
	if allExecuted {
		log.Printf("All Scale Out levels executed for position %s", position.ID)
	}

	return nil
}

func (e *ScaleOutExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	// Scale out doesn't need continuous updates
	return nil
}
