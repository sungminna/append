package quotation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/pkg/ratelimit"
)

const (
	baseURL = "https://api.upbit.com/v1"
)

// Client represents Upbit Quotation API client
type Client struct {
	httpClient  *http.Client
	rateLimiter *ratelimit.RateLimiter
}

// NewClient creates a new Quotation API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: ratelimit.NewRateLimiter(30), // Upbit allows 30 requests/sec for quotation API
	}
}

// Market represents a trading market
type Market struct {
	Market        string `json:"market"`
	KoreanName    string `json:"korean_name"`
	EnglishName   string `json:"english_name"`
	MarketWarning string `json:"market_warning,omitempty"`
}

// Ticker represents current price information
type Ticker struct {
	Market           string  `json:"market"`
	TradeDate        string  `json:"trade_date"`
	TradeTime        string  `json:"trade_time"`
	TradeDateKST     string  `json:"trade_date_kst"`
	TradeTimeKST     string  `json:"trade_time_kst"`
	TradeTimestamp   int64   `json:"trade_timestamp"`
	OpeningPrice     float64 `json:"opening_price"`
	HighPrice        float64 `json:"high_price"`
	LowPrice         float64 `json:"low_price"`
	TradePrice       float64 `json:"trade_price"`
	PrevClosingPrice float64 `json:"prev_closing_price"`
	Change           string  `json:"change"`
	ChangePrice      float64 `json:"change_price"`
	ChangeRate       float64 `json:"change_rate"`
	SignedChangePrice float64 `json:"signed_change_price"`
	SignedChangeRate  float64 `json:"signed_change_rate"`
	TradeVolume      float64 `json:"trade_volume"`
	AccTradePrice    float64 `json:"acc_trade_price"`
	AccTradePrice24h float64 `json:"acc_trade_price_24h"`
	AccTradeVolume   float64 `json:"acc_trade_volume"`
	AccTradeVolume24h float64 `json:"acc_trade_volume_24h"`
	Highest52WeekPrice float64 `json:"highest_52_week_price"`
	Highest52WeekDate  string  `json:"highest_52_week_date"`
	Lowest52WeekPrice  float64 `json:"lowest_52_week_price"`
	Lowest52WeekDate   string  `json:"lowest_52_week_date"`
	Timestamp         int64   `json:"timestamp"`
}

// GetMarkets retrieves all available markets
func (c *Client) GetMarkets(ctx context.Context) ([]Market, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "GET", "/market/all", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var markets []Market
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("failed to decode markets: %w", err)
	}

	return markets, nil
}

// GetCandles retrieves candle data
func (c *Client) GetCandles(ctx context.Context, market string, interval model.CandleInterval, count int) ([]model.Candle, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	endpoint := c.getCandleEndpoint(interval)
	params := url.Values{}
	params.Add("market", market)
	params.Add("count", fmt.Sprintf("%d", count))

	resp, err := c.doRequest(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var candles []model.Candle
	if err := json.NewDecoder(resp.Body).Decode(&candles); err != nil {
		return nil, fmt.Errorf("failed to decode candles: %w", err)
	}

	// Set market and interval for each candle
	for i := range candles {
		candles[i].Market = market
		candles[i].Interval = interval
	}

	return candles, nil
}

// GetCandleRange retrieves candles within a time range
func (c *Client) GetCandleRange(ctx context.Context, market string, interval model.CandleInterval, from, to time.Time) ([]model.Candle, error) {
	var allCandles []model.Candle
	currentTo := to
	maxCount := 200 // Upbit's max count per request

	for {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		endpoint := c.getCandleEndpoint(interval)
		params := url.Values{}
		params.Add("market", market)
		params.Add("to", currentTo.UTC().Format("2006-01-02T15:04:05"))
		params.Add("count", fmt.Sprintf("%d", maxCount))

		resp, err := c.doRequest(ctx, "GET", endpoint+"?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}

		var candles []model.Candle
		if err := json.NewDecoder(resp.Body).Decode(&candles); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode candles: %w", err)
		}
		resp.Body.Close()

		if len(candles) == 0 {
			break
		}

		// Set market and interval
		for i := range candles {
			candles[i].Market = market
			candles[i].Interval = interval
		}

		// Filter candles within range and add to result
		for _, candle := range candles {
			if candle.Timestamp.Before(from) {
				return allCandles, nil
			}
			if candle.Timestamp.After(from) && candle.Timestamp.Before(to) {
				allCandles = append(allCandles, candle)
			}
		}

		// Update currentTo for next iteration
		lastCandle := candles[len(candles)-1]
		if lastCandle.Timestamp.Before(from) {
			break
		}
		currentTo = lastCandle.Timestamp

		// Prevent infinite loop
		if len(candles) < maxCount {
			break
		}
	}

	return allCandles, nil
}

// GetOrderbook retrieves current orderbook
func (c *Client) GetOrderbook(ctx context.Context, market string) (*model.Orderbook, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("markets", market)

	resp, err := c.doRequest(ctx, "GET", "/orderbook?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orderbooks []model.Orderbook
	if err := json.NewDecoder(resp.Body).Decode(&orderbooks); err != nil {
		return nil, fmt.Errorf("failed to decode orderbook: %w", err)
	}

	if len(orderbooks) == 0 {
		return nil, fmt.Errorf("no orderbook data for market %s", market)
	}

	return &orderbooks[0], nil
}

// GetTicker retrieves ticker information for markets
func (c *Client) GetTicker(ctx context.Context, markets []string) ([]Ticker, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	for _, market := range markets {
		params.Add("markets", market)
	}

	resp, err := c.doRequest(ctx, "GET", "/ticker?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tickers []Ticker
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("failed to decode ticker: %w", err)
	}

	return tickers, nil
}

// doRequest performs HTTP request with error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// getCandleEndpoint returns the appropriate endpoint for candle interval
func (c *Client) getCandleEndpoint(interval model.CandleInterval) string {
	switch interval {
	case model.CandleInterval1m:
		return "/candles/minutes/1"
	case model.CandleInterval3m:
		return "/candles/minutes/3"
	case model.CandleInterval5m:
		return "/candles/minutes/5"
	case model.CandleInterval15m:
		return "/candles/minutes/15"
	case model.CandleInterval30m:
		return "/candles/minutes/30"
	case model.CandleInterval1h:
		return "/candles/minutes/60"
	case model.CandleInterval4h:
		return "/candles/minutes/240"
	case model.CandleInterval1d:
		return "/candles/days"
	case model.CandleInterval1w:
		return "/candles/weeks"
	case model.CandleInterval1M:
		return "/candles/months"
	default:
		return "/candles/minutes/1"
	}
}
