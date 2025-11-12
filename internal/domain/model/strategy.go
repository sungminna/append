package model

import (
	"time"

	"github.com/google/uuid"
)

// StrategyType represents the type of trading strategy
type StrategyType string

const (
	StrategyTypeStopLoss      StrategyType = "stop_loss"
	StrategyTypeTakeProfit    StrategyType = "take_profit"
	StrategyTypeTrailingStop  StrategyType = "trailing_stop"
	StrategyTypeOCO           StrategyType = "oco"           // One Cancels Other
	StrategyTypeScaleOut      StrategyType = "scale_out"    // 분할 청산
	StrategyTypeTimeBasedExit StrategyType = "time_based_exit"
)

// StrategyStatus represents the status of a strategy
type StrategyStatus string

const (
	StrategyStatusActive    StrategyStatus = "active"
	StrategyStatusTriggered StrategyStatus = "triggered"
	StrategyStatusCancelled StrategyStatus = "cancelled"
	StrategyStatusCompleted StrategyStatus = "completed"
)

// Strategy represents a trading strategy attached to a position
type Strategy struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	PositionID uuid.UUID      `json:"position_id" db:"position_id"`
	Type       StrategyType   `json:"type" db:"strategy_type"`
	Status     StrategyStatus `json:"status" db:"status"`
	Config     StrategyConfig `json:"config" db:"config"` // Strategy-specific configuration
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
	TriggeredAt *time.Time    `json:"triggered_at,omitempty" db:"triggered_at"`
}

// StrategyConfig holds the configuration for each strategy type
type StrategyConfig interface{}

// StopLossConfig represents stop loss configuration
type StopLossConfig struct {
	StopPrice float64 `json:"stop_price"` // 손절 가격
}

// TakeProfitConfig represents take profit configuration
type TakeProfitConfig struct {
	TargetPrice float64 `json:"target_price"` // 목표 가격
}

// TrailingStopConfig represents trailing stop configuration
type TrailingStopConfig struct {
	TrailPercent float64  `json:"trail_percent"` // 트레일링 퍼센트
	HighestPrice *float64 `json:"highest_price,omitempty"`
	LowestPrice  *float64 `json:"lowest_price,omitempty"`
	TriggerPrice *float64 `json:"trigger_price,omitempty"`
}

// OCOConfig represents OCO (One Cancels Other) configuration
// 손절과 익절을 동시에 설정, 하나가 체결되면 나머지 취소
type OCOConfig struct {
	StopPrice   float64 `json:"stop_price"`   // 손절 가격
	TargetPrice float64 `json:"target_price"` // 익절 가격
}

// ScaleOutConfig represents scale out configuration
// 여러 단계로 나누어 청산
type ScaleOutConfig struct {
	Levels []ScaleOutLevel `json:"levels"` // 청산 레벨들
}

type ScaleOutLevel struct {
	Price      float64 `json:"price"`       // 청산 가격
	Percentage float64 `json:"percentage"`  // 청산 비율 (0-100)
	Executed   bool    `json:"executed"`    // 실행 여부
}

// TimeBasedExitConfig represents time-based exit configuration
// 특정 시간이 되면 자동 청산
type TimeBasedExitConfig struct {
	ExitTime time.Time `json:"exit_time"` // 청산 시간
}

// NewStrategy creates a new strategy
func NewStrategy(positionID uuid.UUID, strategyType StrategyType) *Strategy {
	now := time.Now()
	return &Strategy{
		ID:         uuid.New(),
		PositionID: positionID,
		Type:       strategyType,
		Status:     StrategyStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Trigger marks the strategy as triggered
func (s *Strategy) Trigger() {
	s.Status = StrategyStatusTriggered
	now := time.Now()
	s.TriggeredAt = &now
	s.UpdatedAt = now
}

// Cancel marks the strategy as cancelled
func (s *Strategy) Cancel() {
	s.Status = StrategyStatusCancelled
	s.UpdatedAt = time.Now()
}

// Complete marks the strategy as completed
func (s *Strategy) Complete() {
	s.Status = StrategyStatusCompleted
	s.UpdatedAt = time.Now()
}

// IsActive checks if the strategy is active
func (s *Strategy) IsActive() bool {
	return s.Status == StrategyStatusActive
}
