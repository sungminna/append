package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

const (
	wsURL = "wss://api.upbit.com/websocket/v1"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeTicker    MessageType = "ticker"
	MessageTypeTrade     MessageType = "trade"
	MessageTypeOrderbook MessageType = "orderbook"
)

// Client represents Upbit WebSocket client
type Client struct {
	conn        *websocket.Conn
	mu          sync.RWMutex
	handlers    map[MessageType][]MessageHandler
	isConnected bool
	reconnect   bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// MessageHandler is a callback function for WebSocket messages
type MessageHandler func(interface{}) error

// SubscribeRequest represents a WebSocket subscription request
type SubscribeRequest struct {
	Ticket string                   `json:"ticket"`
	Type   string                   `json:"type"`
	Codes  []string                 `json:"codes"`
	Format string                   `json:"format,omitempty"`
}

// TickerMessage represents a ticker WebSocket message
type TickerMessage struct {
	Type             string  `json:"type"`
	Code             string  `json:"code"`
	OpeningPrice     float64 `json:"opening_price"`
	HighPrice        float64 `json:"high_price"`
	LowPrice         float64 `json:"low_price"`
	TradePrice       float64 `json:"trade_price"`
	PrevClosingPrice float64 `json:"prev_closing_price"`
	Change           string  `json:"change"`
	ChangePrice      float64 `json:"change_price"`
	SignedChangePrice float64 `json:"signed_change_price"`
	ChangeRate       float64 `json:"change_rate"`
	SignedChangeRate  float64 `json:"signed_change_rate"`
	TradeVolume      float64 `json:"trade_volume"`
	AccTradeVolume   float64 `json:"acc_trade_volume"`
	AccTradeVolume24h float64 `json:"acc_trade_volume_24h"`
	AccTradePrice    float64 `json:"acc_trade_price"`
	AccTradePrice24h float64 `json:"acc_trade_price_24h"`
	TradeDate        string  `json:"trade_date"`
	TradeTime        string  `json:"trade_time"`
	TradeTimestamp   int64   `json:"trade_timestamp"`
	AskBid           string  `json:"ask_bid"`
	AccAskVolume     float64 `json:"acc_ask_volume"`
	AccBidVolume     float64 `json:"acc_bid_volume"`
	Highest52WeekPrice float64 `json:"highest_52_week_price"`
	Highest52WeekDate  string  `json:"highest_52_week_date"`
	Lowest52WeekPrice  float64 `json:"lowest_52_week_price"`
	Lowest52WeekDate   string  `json:"lowest_52_week_date"`
	Timestamp         int64   `json:"timestamp"`
	StreamType        string  `json:"stream_type"`
}

// TradeMessage represents a trade WebSocket message
type TradeMessage struct {
	Type              string  `json:"type"`
	Code              string  `json:"code"`
	TradePrice        float64 `json:"trade_price"`
	TradeVolume       float64 `json:"trade_volume"`
	AskBid            string  `json:"ask_bid"`
	PrevClosingPrice  float64 `json:"prev_closing_price"`
	Change            string  `json:"change"`
	ChangePrice       float64 `json:"change_price"`
	TradeDate         string  `json:"trade_date"`
	TradeTime         string  `json:"trade_time"`
	TradeTimestamp    int64   `json:"trade_timestamp"`
	Timestamp         int64   `json:"timestamp"`
	SequentialID      int64   `json:"sequential_id"`
	StreamType        string  `json:"stream_type"`
}

// OrderbookMessage represents an orderbook WebSocket message
type OrderbookMessage struct {
	Type           string                     `json:"type"`
	Code           string                     `json:"code"`
	TotalAskSize   float64                    `json:"total_ask_size"`
	TotalBidSize   float64                    `json:"total_bid_size"`
	OrderbookUnits []model.OrderbookUnit      `json:"orderbook_units"`
	Timestamp      int64                      `json:"timestamp"`
	StreamType     string                     `json:"stream_type"`
}

// NewClient creates a new WebSocket client
func NewClient() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		handlers:  make(map[MessageType][]MessageHandler),
		reconnect: true,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Connect establishes WebSocket connection
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return nil
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	// Start message reader
	go c.readMessages()

	return nil
}

// Subscribe subscribes to market data
func (c *Client) Subscribe(msgType MessageType, markets []string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("not connected")
	}

	// Send subscription request
	ticket := uuid.New().String()
	requests := []interface{}{
		map[string]string{"ticket": ticket},
		map[string]interface{}{
			"type":  string(msgType),
			"codes": markets,
		},
	}

	if err := c.conn.WriteJSON(requests); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

// OnTicker registers a handler for ticker messages
func (c *Client) OnTicker(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[MessageTypeTicker] = append(c.handlers[MessageTypeTicker], handler)
}

// OnTrade registers a handler for trade messages
func (c *Client) OnTrade(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[MessageTypeTrade] = append(c.handlers[MessageTypeTrade], handler)
}

// OnOrderbook registers a handler for orderbook messages
func (c *Client) OnOrderbook(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[MessageTypeOrderbook] = append(c.handlers[MessageTypeOrderbook], handler)
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reconnect = false
	c.cancel()

	if c.conn != nil {
		c.isConnected = false
		return c.conn.Close()
	}

	return nil
}

// readMessages reads and processes incoming WebSocket messages
func (c *Client) readMessages() {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		c.mu.Unlock()

		if c.reconnect {
			time.Sleep(5 * time.Second)
			if err := c.Connect(); err == nil {
				go c.readMessages()
			}
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			c.handleMessage(message)
		}
	}
}

// handleMessage processes a single message
func (c *Client) handleMessage(data []byte) {
	var msgType struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &msgType); err != nil {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	switch MessageType(msgType.Type) {
	case MessageTypeTicker:
		var msg TickerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		for _, handler := range c.handlers[MessageTypeTicker] {
			handler(msg)
		}

	case MessageTypeTrade:
		var msg TradeMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		for _, handler := range c.handlers[MessageTypeTrade] {
			handler(msg)
		}

	case MessageTypeOrderbook:
		var msg OrderbookMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		for _, handler := range c.handlers[MessageTypeOrderbook] {
			handler(msg)
		}
	}
}
