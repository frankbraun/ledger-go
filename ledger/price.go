package ledger

import (
	"fmt"
	"sort"
	"time"
)

// PricePoint represents a market price at a specific point in time.
// Supports Ledger price directive: P 2024/01/15 BTC 42000 USD
type PricePoint struct {
	Date      time.Time // Date of the price
	Commodity string    // Asset being priced (e.g., "BTC")
	Price     float64   // Price in USD
}

// PriceHistory stores historical prices for commodities (in USD).
type PriceHistory struct {
	Prices map[string][]PricePoint // Commodity -> []PricePoint (sorted by date)
}

// NewPriceHistory creates a new empty PriceHistory.
func NewPriceHistory() *PriceHistory {
	return &PriceHistory{
		Prices: make(map[string][]PricePoint),
	}
}

// AddPrice adds a price point to the history, maintaining chronological order.
func (h *PriceHistory) AddPrice(commodity string, date time.Time, price float64) {
	point := PricePoint{
		Date:      date,
		Commodity: commodity,
		Price:     price,
	}
	prices := h.Prices[commodity]
	prices = append(prices, point)
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})
	h.Prices[commodity] = prices
}

// GetPrice returns the price of a commodity at the given date.
// If no exact match exists, returns the most recent price before that date.
// Returns an error if no price is available on or before the given date.
func (h *PriceHistory) GetPrice(commodity string, date time.Time) (float64, error) {
	prices, ok := h.Prices[commodity]
	if !ok || len(prices) == 0 {
		return 0, fmt.Errorf("no price history for commodity: %s", commodity)
	}

	// Binary search for the latest price on or before the given date
	idx := sort.Search(len(prices), func(i int) bool {
		return prices[i].Date.After(date)
	})

	// idx is the first index where Date > date, so idx-1 is the last where Date <= date
	if idx == 0 {
		return 0, fmt.Errorf("no price available for %s on or before %s", commodity, date.Format(DateFormat))
	}

	return prices[idx-1].Price, nil
}

// GetPriceExact returns the price of a commodity at the exact given date.
// Returns an error if no price exists for that exact date.
func (h *PriceHistory) GetPriceExact(commodity string, date time.Time) (float64, error) {
	prices, ok := h.Prices[commodity]
	if !ok || len(prices) == 0 {
		return 0, fmt.Errorf("no price history for commodity: %s", commodity)
	}

	// Binary search for exact date match
	idx := sort.Search(len(prices), func(i int) bool {
		return !prices[i].Date.Before(date)
	})

	if idx < len(prices) && prices[idx].Date.Equal(date) {
		return prices[idx].Price, nil
	}

	return 0, fmt.Errorf("no price for %s on %s", commodity, date.Format(DateFormat))
}

// LatestPrice returns the most recent price for a commodity.
// Returns an error if no price history exists for the commodity.
func (h *PriceHistory) LatestPrice(commodity string) (PricePoint, error) {
	prices, ok := h.Prices[commodity]
	if !ok || len(prices) == 0 {
		return PricePoint{}, fmt.Errorf("no price history for commodity: %s", commodity)
	}
	return prices[len(prices)-1], nil
}
