package ledger

import (
	"testing"
	"time"
)

func TestAddPrice(t *testing.T) {
	h := NewPriceHistory()

	// Add prices out of order
	h.AddPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 60000)
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	prices := h.Prices["BTC"]
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	// Verify chronological order
	if prices[0].Price != 40000 {
		t.Errorf("first price should be 40000, got %f", prices[0].Price)
	}
	if prices[1].Price != 50000 {
		t.Errorf("second price should be 50000, got %f", prices[1].Price)
	}
	if prices[2].Price != 60000 {
		t.Errorf("third price should be 60000, got %f", prices[2].Price)
	}
}

func TestGetPrice_ExactMatch(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 42000)

	price, err := h.GetPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 42000 {
		t.Errorf("expected 42000, got %f", price)
	}
}

func TestGetPrice_UsesEarlierPrice(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	// Query for a date between the two prices
	price, err := h.GetPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 40000 {
		t.Errorf("expected 40000 (earlier price), got %f", price)
	}
}

func TestGetPrice_UsesLatestBeforeDate(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)
	h.AddPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 60000)

	// Query for a date after all prices
	price, err := h.GetPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 60000 {
		t.Errorf("expected 60000 (latest price), got %f", price)
	}
}

func TestGetPrice_NoHistoryError(t *testing.T) {
	h := NewPriceHistory()

	_, err := h.GetPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for unknown commodity")
	}
}

func TestGetPrice_DateBeforeFirstPriceError(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	_, err := h.GetPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for date before first price")
	}
}

func TestGetPriceExact_Match(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 42000)

	price, err := h.GetPriceExact("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 42000 {
		t.Errorf("expected 42000, got %f", price)
	}
}

func TestGetPriceExact_NoMatch(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	_, err := h.GetPriceExact("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for non-exact date")
	}
}

func TestGetPriceExact_NoHistory(t *testing.T) {
	h := NewPriceHistory()

	_, err := h.GetPriceExact("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for unknown commodity")
	}
}

func TestLatestPrice(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 60000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	point, err := h.LatestPrice("BTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if point.Price != 60000 {
		t.Errorf("expected 60000, got %f", point.Price)
	}
	if !point.Date.Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected date 2024/03/01, got %s", point.Date.Format(DateFormat))
	}
}

func TestLatestPrice_NoHistory(t *testing.T) {
	h := NewPriceHistory()

	_, err := h.LatestPrice("BTC")
	if err == nil {
		t.Error("expected error for unknown commodity")
	}
}

func TestGetPrice_MultipleCommodities(t *testing.T) {
	h := NewPriceHistory()
	h.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	h.AddPrice("ETH", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 2000)
	h.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 45000)
	h.AddPrice("ETH", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 2500)

	btcPrice, err := h.GetPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if btcPrice != 45000 {
		t.Errorf("expected BTC 45000, got %f", btcPrice)
	}

	ethPrice, err := h.GetPrice("ETH", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ethPrice != 2500 {
		t.Errorf("expected ETH 2500, got %f", ethPrice)
	}
}
