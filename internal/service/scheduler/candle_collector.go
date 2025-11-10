package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/upbit/quotation"
)

// CandleCollector collects candle data from Upbit API
type CandleCollector struct {
	quotationClient *quotation.Client
	markets         []string
	interval        model.CandleInterval
	storage         CandleStorage
	mu              sync.RWMutex
	isRunning       bool
	stopChan        chan struct{}
}

// CandleStorage is an interface for storing candle data
type CandleStorage interface {
	SaveCandles(ctx context.Context, candles []model.Candle) error
	GetLatestCandle(ctx context.Context, market string, interval model.CandleInterval) (*model.Candle, error)
}

// NewCandleCollector creates a new candle collector
func NewCandleCollector(
	quotationClient *quotation.Client,
	storage CandleStorage,
	markets []string,
	interval model.CandleInterval,
) *CandleCollector {
	return &CandleCollector{
		quotationClient: quotationClient,
		markets:         markets,
		interval:        interval,
		storage:         storage,
		stopChan:        make(chan struct{}),
	}
}

// Start starts the candle collector
func (cc *CandleCollector) Start(ctx context.Context) error {
	cc.mu.Lock()
	if cc.isRunning {
		cc.mu.Unlock()
		return nil
	}
	cc.isRunning = true
	cc.mu.Unlock()

	// Collect historical data on startup
	log.Println("Collecting historical candle data...")
	if err := cc.collectHistoricalData(ctx); err != nil {
		log.Printf("Error collecting historical data: %v", err)
	}

	// Start periodic collection
	go cc.runPeriodic(ctx)

	return nil
}

// Stop stops the candle collector
func (cc *CandleCollector) Stop() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if !cc.isRunning {
		return
	}

	close(cc.stopChan)
	cc.isRunning = false
}

// collectHistoricalData collects historical candle data
func (cc *CandleCollector) collectHistoricalData(ctx context.Context) error {
	// Collect last 30 days of data
	to := time.Now()
	from := to.Add(-30 * 24 * time.Hour)

	for _, market := range cc.markets {
		log.Printf("Collecting historical data for %s...", market)

		candles, err := cc.quotationClient.GetCandleRange(ctx, market, cc.interval, from, to)
		if err != nil {
			log.Printf("Error collecting historical data for %s: %v", market, err)
			continue
		}

		if len(candles) > 0 {
			if err := cc.storage.SaveCandles(ctx, candles); err != nil {
				log.Printf("Error saving candles for %s: %v", market, err)
			} else {
				log.Printf("Saved %d candles for %s", len(candles), market)
			}
		}

		// Rate limiting - small delay between markets
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// runPeriodic runs periodic candle collection
func (cc *CandleCollector) runPeriodic(ctx context.Context) {
	ticker := time.NewTicker(cc.getCollectionInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cc.stopChan:
			return
		case <-ticker.C:
			cc.collectLatestCandles(ctx)
		}
	}
}

// collectLatestCandles collects the latest candles for all markets
func (cc *CandleCollector) collectLatestCandles(ctx context.Context) {
	for _, market := range cc.markets {
		candles, err := cc.quotationClient.GetCandles(ctx, market, cc.interval, 1)
		if err != nil {
			log.Printf("Error collecting candle for %s: %v", market, err)
			continue
		}

		if len(candles) > 0 {
			if err := cc.storage.SaveCandles(ctx, candles); err != nil {
				log.Printf("Error saving candle for %s: %v", market, err)
			}
		}
	}
}

// getCollectionInterval returns the collection interval based on candle interval
func (cc *CandleCollector) getCollectionInterval() time.Duration {
	switch cc.interval {
	case model.CandleInterval1m:
		return 1 * time.Minute
	case model.CandleInterval3m:
		return 3 * time.Minute
	case model.CandleInterval5m:
		return 5 * time.Minute
	case model.CandleInterval15m:
		return 15 * time.Minute
	case model.CandleInterval30m:
		return 30 * time.Minute
	case model.CandleInterval1h:
		return 1 * time.Hour
	case model.CandleInterval4h:
		return 4 * time.Hour
	case model.CandleInterval1d:
		return 24 * time.Hour
	default:
		return 1 * time.Minute
	}
}
