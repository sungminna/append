package exchange

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/pkg/ratelimit"
)

const (
	baseURL = "https://api.upbit.com/v1"
)

// Client represents Upbit Exchange API client
type Client struct {
	accessKey   string
	secretKey   string
	httpClient  *http.Client
	rateLimiter *ratelimit.RateLimiter
}

// NewClient creates a new Exchange API client
func NewClient(accessKey, secretKey string) *Client {
	return &Client{
		accessKey: accessKey,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: ratelimit.NewRateLimiter(8), // Upbit allows 8 requests/sec for exchange API
	}
}

// Account represents user's account balance
type Account struct {
	Currency            string  `json:"currency"`
	Balance             string  `json:"balance"`
	Locked              string  `json:"locked"`
	AvgBuyPrice         string  `json:"avg_buy_price"`
	AvgBuyPriceModified bool    `json:"avg_buy_price_modified"`
	UnitCurrency        string  `json:"unit_currency"`
}

// OrderResponse represents the response from order API
type OrderResponse struct {
	UUID            string    `json:"uuid"`
	Side            string    `json:"side"`
	OrdType         string    `json:"ord_type"`
	Price           *string   `json:"price"`
	State           string    `json:"state"`
	Market          string    `json:"market"`
	CreatedAt       time.Time `json:"created_at"`
	Volume          *string   `json:"volume"`
	RemainingVolume *string   `json:"remaining_volume"`
	ReservedFee     string    `json:"reserved_fee"`
	RemainingFee    string    `json:"remaining_fee"`
	PaidFee         string    `json:"paid_fee"`
	Locked          string    `json:"locked"`
	ExecutedVolume  string    `json:"executed_volume"`
	TradesCount     int       `json:"trades_count"`
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	Market string  `json:"market"`
	Side   string  `json:"side"`
	Volume *string `json:"volume,omitempty"`
	Price  *string `json:"price,omitempty"`
	OrdType string `json:"ord_type"`
}

// GetAccounts retrieves all account balances
func (c *Client) GetAccounts(ctx context.Context) ([]Account, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	token, err := c.generateToken(nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "GET", "/accounts", nil, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("failed to decode accounts: %w", err)
	}

	return accounts, nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Create query parameters for JWT
	params := map[string]string{
		"market":   req.Market,
		"side":     req.Side,
		"ord_type": req.OrdType,
	}

	if req.Volume != nil {
		params["volume"] = *req.Volume
	}
	if req.Price != nil {
		params["price"] = *req.Price
	}

	token, err := c.generateToken(params)
	if err != nil {
		return nil, err
	}

	// Convert request to JSON
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/orders", bytes.NewReader(bodyBytes), token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &orderResp, nil
}

// GetOrder retrieves order information
func (c *Client) GetOrder(ctx context.Context, orderUUID string) (*OrderResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := map[string]string{
		"uuid": orderUUID,
	}

	token, err := c.generateToken(params)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Add("uuid", orderUUID)

	resp, err := c.doRequest(ctx, "GET", "/order?"+query.Encode(), nil, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &orderResp, nil
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, orderUUID string) (*OrderResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := map[string]string{
		"uuid": orderUUID,
	}

	token, err := c.generateToken(params)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Add("uuid", orderUUID)

	resp, err := c.doRequest(ctx, "DELETE", "/order?"+query.Encode(), nil, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &orderResp, nil
}

// GetOrders retrieves list of orders
func (c *Client) GetOrders(ctx context.Context, market string, state string) ([]OrderResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := map[string]string{
		"market": market,
		"state":  state,
	}

	token, err := c.generateToken(params)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Add("market", market)
	query.Add("state", state)

	resp, err := c.doRequest(ctx, "GET", "/orders?"+query.Encode(), nil, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orders []OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("failed to decode orders: %w", err)
	}

	return orders, nil
}

// generateToken generates JWT token for authentication
func (c *Client) generateToken(params map[string]string) (string, error) {
	claims := jwt.MapClaims{
		"access_key": c.accessKey,
		"nonce":      uuid.New().String(),
	}

	if params != nil && len(params) > 0 {
		query := url.Values{}
		for k, v := range params {
			query.Add(k, v)
		}
		queryString := query.Encode()

		hash := sha512.New()
		hash.Write([]byte(queryString))
		queryHash := hex.EncodeToString(hash.Sum(nil))

		claims["query_hash"] = queryHash
		claims["query_hash_alg"] = "SHA512"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(c.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// doRequest performs HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// ConvertOrderResponseToModel converts API response to domain model
func ConvertOrderResponseToModel(resp *OrderResponse, userID uuid.UUID) (*model.Order, error) {
	orderID, err := uuid.Parse(resp.UUID)
	if err != nil {
		return nil, fmt.Errorf("invalid order UUID: %w", err)
	}

	var side model.OrderSide
	if resp.Side == "bid" {
		side = model.OrderSideBid
	} else {
		side = model.OrderSideAsk
	}

	var orderType model.OrderType
	if resp.OrdType == "limit" {
		orderType = model.OrderTypeLimit
	} else {
		orderType = model.OrderTypeMarket
	}

	order := &model.Order{
		ID:              orderID,
		UserID:          userID,
		Market:          resp.Market,
		Side:            side,
		Type:            orderType,
		Status:          convertOrderStatus(resp.State),
		ExchangeOrderID: &resp.UUID,
		CreatedAt:       resp.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	return order, nil
}

func convertOrderStatus(state string) model.OrderStatus {
	switch state {
	case "wait":
		return model.OrderStatusSubmitted
	case "done":
		return model.OrderStatusFilled
	case "cancel":
		return model.OrderStatusCancelled
	default:
		return model.OrderStatusPending
	}
}
