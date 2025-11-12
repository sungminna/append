package strategy

import (
	"context"
	"fmt"
	"log"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
	"github.com/sungminna/upbit-trading-platform/internal/service/trading"
)

// OCOExecutor implements OCO (One Cancels Other) strategy
// 손절과 익절을 동시에 설정, 하나가 체결되면 나머지 자동 취소
type OCOExecutor struct {
	tradingEngine *trading.Engine
	strategyRepo  repository.StrategyRepository
}

// NewOCOExecutor creates a new OCO executor
func NewOCOExecutor(tradingEngine *trading.Engine, strategyRepo repository.StrategyRepository) *OCOExecutor {
	return &OCOExecutor{
		tradingEngine: tradingEngine,
		strategyRepo:  strategyRepo,
	}
}

func (e *OCOExecutor) GetType() model.StrategyType {
	return model.StrategyTypeOCO
}

func (e *OCOExecutor) Check(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) (bool, error) {
	config, ok := strategy.Config.(model.OCOConfig)
	if !ok {
		return false, fmt.Errorf("invalid OCO config")
	}

	if position.Side == model.PositionSideLong {
		// Trigger if price hits stop loss OR take profit
		return currentPrice <= config.StopPrice || currentPrice >= config.TargetPrice, nil
	}
	// Short position
	return currentPrice >= config.StopPrice || currentPrice <= config.TargetPrice, nil
}

func (e *OCOExecutor) Execute(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	config, ok := strategy.Config.(model.OCOConfig)
	if !ok {
		return fmt.Errorf("invalid OCO config")
	}

	var triggeredType string
	if position.Side == model.PositionSideLong {
		if currentPrice <= config.StopPrice {
			triggeredType = "Stop Loss"
		} else {
			triggeredType = "Take Profit"
		}
	} else {
		if currentPrice >= config.StopPrice {
			triggeredType = "Stop Loss"
		} else {
			triggeredType = "Take Profit"
		}
	}

	log.Printf("Executing OCO %s for position %s at price %.8f", triggeredType, position.ID, currentPrice)

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
		return fmt.Errorf("failed to place OCO order: %w", err)
	}

	log.Printf("OCO %s order placed for position %s", triggeredType, position.ID)
	return nil
}

func (e *OCOExecutor) Update(ctx context.Context, strategy *model.Strategy, position *model.Position, currentPrice float64) error {
	// OCO doesn't need updates
	return nil
}
