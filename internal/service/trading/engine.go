package trading

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/exchange"
)

// Engine handles order execution and state management
type Engine struct {
	orderRepo          repository.OrderRepository
	executionRepo      repository.OrderExecutionRepository
	positionRepo       repository.PositionRepository
	userAPIKeyRepo     repository.UserAPIKeyRepository
	exchangeClients    map[uuid.UUID]*exchange.Client // userID -> exchange client
	mu                 sync.RWMutex
	orderStateMu       sync.RWMutex // Separate mutex for order state management
	monitoringOrders   map[uuid.UUID]bool
	stopChan           chan struct{}
	isRunning          bool
}

// NewEngine creates a new trading engine
func NewEngine(
	orderRepo repository.OrderRepository,
	executionRepo repository.OrderExecutionRepository,
	positionRepo repository.PositionRepository,
	userAPIKeyRepo repository.UserAPIKeyRepository,
) *Engine {
	return &Engine{
		orderRepo:        orderRepo,
		executionRepo:    executionRepo,
		positionRepo:     positionRepo,
		userAPIKeyRepo:   userAPIKeyRepo,
		exchangeClients:  make(map[uuid.UUID]*exchange.Client),
		monitoringOrders: make(map[uuid.UUID]bool),
		stopChan:         make(chan struct{}),
	}
}

// Start starts the trading engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil
	}
	e.isRunning = true
	e.mu.Unlock()

	log.Println("Trading engine started")

	// Start order monitoring goroutine
	go e.monitorOrders(ctx)

	return nil
}

// Stop stops the trading engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return
	}

	close(e.stopChan)
	e.isRunning = false
	log.Println("Trading engine stopped")
}

// PlaceOrderRequest represents a request to place an order
type PlaceOrderRequest struct {
	Market     string           `json:"market"`
	Side       model.OrderSide  `json:"side"`
	Type       model.OrderType  `json:"type"`
	Price      *float64         `json:"price,omitempty"`
	Quantity   float64          `json:"quantity"`
	PositionID *uuid.UUID       `json:"position_id,omitempty"`
	SplitCount int              `json:"split_count,omitempty"` // Number of splits for order
}

// PlaceOrder places a new order
func (e *Engine) PlaceOrder(ctx context.Context, userID uuid.UUID, req *PlaceOrderRequest) ([]*model.Order, error) {
	// Validate request
	if req.SplitCount < 1 {
		req.SplitCount = 1
	}

	// Create orders (split if requested)
	var orders []*model.Order
	quantityPerOrder := req.Quantity / float64(req.SplitCount)

	for i := 0; i < req.SplitCount; i++ {
		order := model.NewOrder(userID, req.Market, req.Side, req.Type, quantityPerOrder, req.Price)
		order.PositionID = req.PositionID

		// Save order to database
		if err := e.orderRepo.Create(ctx, order); err != nil {
			return nil, fmt.Errorf("failed to create order: %w", err)
		}

		orders = append(orders, order)
	}

	// Execute orders asynchronously
	for _, order := range orders {
		go e.executeOrder(context.Background(), order)
	}

	return orders, nil
}

// executeOrder executes a single order
func (e *Engine) executeOrder(ctx context.Context, order *model.Order) {
	log.Printf("Executing order %s for user %s", order.ID, order.UserID)

	// Get exchange client for user
	client, err := e.getExchangeClient(ctx, order.UserID)
	if err != nil {
		e.updateOrderStatus(ctx, order, model.OrderStatusFailed)
		log.Printf("Failed to get exchange client for user %s: %v", order.UserID, err)
		return
	}

	// Prepare order request
	var priceStr, volumeStr *string
	if order.Price != nil {
		price := fmt.Sprintf("%.8f", *order.Price)
		priceStr = &price
	}
	volume := fmt.Sprintf("%.8f", order.Quantity)
	volumeStr = &volume

	ordType := "limit"
	if order.Type == model.OrderTypeMarket {
		ordType = "market"
	}

	orderReq := exchange.OrderRequest{
		Market:  order.Market,
		Side:    string(order.Side),
		Volume:  volumeStr,
		Price:   priceStr,
		OrdType: ordType,
	}

	// Submit order to exchange
	resp, err := client.PlaceOrder(ctx, orderReq)
	if err != nil {
		e.updateOrderStatus(ctx, order, model.OrderStatusFailed)
		log.Printf("Failed to place order %s on exchange: %v", order.ID, err)
		return
	}

	// Update order with exchange order ID
	order.ExchangeOrderID = &resp.UUID
	order.Status = model.OrderStatusSubmitted
	now := time.Now()
	order.SubmittedAt = &now

	if err := e.orderRepo.Update(ctx, order); err != nil {
		log.Printf("Failed to update order %s: %v", order.ID, err)
	}

	// Start monitoring this order
	e.startMonitoringOrder(ctx, order.ID)
}

// monitorOrders monitors submitted orders for status updates
func (e *Engine) monitorOrders(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.checkPendingOrders(context.Background())
		}
	}
}

// checkPendingOrders checks all pending orders for updates
func (e *Engine) checkPendingOrders(ctx context.Context) {
	e.orderStateMu.RLock()
	orderIDs := make([]uuid.UUID, 0, len(e.monitoringOrders))
	for orderID := range e.monitoringOrders {
		orderIDs = append(orderIDs, orderID)
	}
	e.orderStateMu.RUnlock()

	for _, orderID := range orderIDs {
		go e.checkOrderStatus(ctx, orderID)
	}
}

// checkOrderStatus checks the status of a single order
func (e *Engine) checkOrderStatus(ctx context.Context, orderID uuid.UUID) {
	order, err := e.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		log.Printf("Failed to get order %s: %v", orderID, err)
		return
	}

	if order.ExchangeOrderID == nil {
		return
	}

	// Get exchange client
	client, err := e.getExchangeClient(ctx, order.UserID)
	if err != nil {
		log.Printf("Failed to get exchange client for order %s: %v", orderID, err)
		return
	}

	// Get order status from exchange
	resp, err := client.GetOrder(ctx, *order.ExchangeOrderID)
	if err != nil {
		log.Printf("Failed to get order status from exchange for %s: %v", orderID, err)
		return
	}

	// Update order status based on exchange response
	e.processOrderUpdate(ctx, order, resp)
}

// processOrderUpdate processes an order update from the exchange
func (e *Engine) processOrderUpdate(ctx context.Context, order *model.Order, resp *exchange.OrderResponse) {
	e.orderStateMu.Lock()
	defer e.orderStateMu.Unlock()

	previousStatus := order.Status

	// Update order status based on exchange state
	switch resp.State {
	case "wait":
		order.Status = model.OrderStatusSubmitted
	case "watch":
		order.Status = model.OrderStatusSubmitted
	case "done":
		order.Status = model.OrderStatusFilled
		now := time.Now()
		order.FilledAt = &now
		// Stop monitoring filled orders
		delete(e.monitoringOrders, order.ID)
	case "cancel":
		order.Status = model.OrderStatusCancelled
		delete(e.monitoringOrders, order.ID)
	}

	// Check if order is partially filled
	if resp.ExecutedVolume != "" {
		var executedQty float64
		fmt.Sscanf(resp.ExecutedVolume, "%f", &executedQty)

		if executedQty > order.ExecutedQuantity {
			// New execution detected
			newExecutedQty := executedQty - order.ExecutedQuantity
			order.UpdateExecution(newExecutedQty)

			// Create execution record
			var avgPrice float64
			if resp.Price != nil {
				fmt.Sscanf(*resp.Price, "%f", &avgPrice)
			}

			execution := model.NewOrderExecution(order.ID, avgPrice, newExecutedQty, 0)
			if err := e.executionRepo.Create(ctx, execution); err != nil {
				log.Printf("Failed to create order execution: %v", err)
			}

			// Update position if this order is associated with a position
			if order.PositionID != nil {
				e.updatePosition(ctx, order, newExecutedQty, avgPrice)
			}
		}
	}

	// Save updated order
	if err := e.orderRepo.Update(ctx, order); err != nil {
		log.Printf("Failed to update order %s: %v", order.ID, err)
	}

	// Log status changes
	if previousStatus != order.Status {
		log.Printf("Order %s status changed: %s -> %s", order.ID, previousStatus, order.Status)
	}
}

// updatePosition updates the associated position based on order execution
func (e *Engine) updatePosition(ctx context.Context, order *model.Order, qty, price float64) {
	position, err := e.positionRepo.GetByID(ctx, *order.PositionID)
	if err != nil {
		log.Printf("Failed to get position %s: %v", *order.PositionID, err)
		return
	}

	if order.Side == model.OrderSideBid {
		// Buy order - increase position
		position.UpdateQuantity(qty, price)
	} else {
		// Sell order - reduce position
		position.ReduceQuantity(qty, price)
	}

	if err := e.positionRepo.Update(ctx, position); err != nil {
		log.Printf("Failed to update position %s: %v", position.ID, err)
	}
}

// CancelOrder cancels an existing order
func (e *Engine) CancelOrder(ctx context.Context, userID, orderID uuid.UUID) error {
	order, err := e.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != userID {
		return fmt.Errorf("unauthorized: order does not belong to user")
	}

	if !order.IsPending() {
		return fmt.Errorf("order cannot be cancelled (status: %s)", order.Status)
	}

	if order.ExchangeOrderID == nil {
		return fmt.Errorf("order not yet submitted to exchange")
	}

	// Get exchange client
	client, err := e.getExchangeClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get exchange client: %w", err)
	}

	// Cancel order on exchange
	_, err = client.CancelOrder(ctx, *order.ExchangeOrderID)
	if err != nil {
		return fmt.Errorf("failed to cancel order on exchange: %w", err)
	}

	// Update order status
	e.updateOrderStatus(ctx, order, model.OrderStatusCancelled)

	// Stop monitoring
	e.orderStateMu.Lock()
	delete(e.monitoringOrders, order.ID)
	e.orderStateMu.Unlock()

	return nil
}

// GetOrder retrieves an order
func (e *Engine) GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*model.Order, error) {
	order, err := e.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != userID {
		return nil, fmt.Errorf("unauthorized: order does not belong to user")
	}

	return order, nil
}

// GetUserOrders retrieves all orders for a user
func (e *Engine) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	orders, err := e.orderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	return orders, nil
}

// getExchangeClient gets or creates an exchange client for a user
func (e *Engine) getExchangeClient(ctx context.Context, userID uuid.UUID) (*exchange.Client, error) {
	e.mu.RLock()
	client, exists := e.exchangeClients[userID]
	e.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Get user's API key
	apiKey, err := e.userAPIKeyRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user API key: %w", err)
	}

	// Create new exchange client
	client = exchange.NewClient(apiKey.AccessKey, apiKey.SecretKey)

	e.mu.Lock()
	e.exchangeClients[userID] = client
	e.mu.Unlock()

	return client, nil
}

// startMonitoringOrder starts monitoring an order
func (e *Engine) startMonitoringOrder(ctx context.Context, orderID uuid.UUID) {
	e.orderStateMu.Lock()
	e.monitoringOrders[orderID] = true
	e.orderStateMu.Unlock()
}

// updateOrderStatus updates an order's status
func (e *Engine) updateOrderStatus(ctx context.Context, order *model.Order, status model.OrderStatus) {
	order.Status = status
	order.UpdatedAt = time.Now()

	if err := e.orderRepo.Update(ctx, order); err != nil {
		log.Printf("Failed to update order status: %v", err)
	}
}
