package ledger

import (
	"math"
	"testing"
	"time"
)

// floatEquals compares two floats with a small epsilon for floating point comparison.
func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestSnapshot_SingleLot(t *testing.T) {
	p := NewPortfolio()

	// Add a lot
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add price
	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshot.Holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(snapshot.Holdings))
	}

	btc := snapshot.Holdings["BTC"]
	if btc == nil {
		t.Fatal("expected BTC holding")
	}

	if btc.TotalQuantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", btc.TotalQuantity)
	}
	if btc.TotalCostBasis != 40000 {
		t.Errorf("expected cost basis 40000, got %f", btc.TotalCostBasis)
	}
	if btc.MarketValue != 60000 {
		t.Errorf("expected market value 60000, got %f", btc.MarketValue)
	}
	if btc.UnrealizedGain != 20000 {
		t.Errorf("expected unrealized gain 20000, got %f", btc.UnrealizedGain)
	}
	if !floatEquals(btc.UnrealizedGainPercent, 50.0) {
		t.Errorf("expected unrealized gain percent 50.0, got %f", btc.UnrealizedGainPercent)
	}
}

func TestSnapshot_MultipleLots(t *testing.T) {
	p := NewPortfolio()

	// Add multiple lots of the same commodity
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	if btc.TotalQuantity != 2.0 {
		t.Errorf("expected quantity 2.0, got %f", btc.TotalQuantity)
	}
	if btc.TotalCostBasis != 90000 {
		t.Errorf("expected cost basis 90000, got %f", btc.TotalCostBasis)
	}
	if btc.MarketValue != 120000 {
		t.Errorf("expected market value 120000, got %f", btc.MarketValue)
	}
	if btc.LotCount != 2 {
		t.Errorf("expected 2 lots, got %d", btc.LotCount)
	}
	// Weighted average: 90000 / 2 = 45000
	if btc.WeightedAverageCost != 45000 {
		t.Errorf("expected weighted average cost 45000, got %f", btc.WeightedAverageCost)
	}
}

func TestSnapshot_MultipleCommodities(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	p.Lots.AddLot(&Lot{
		Commodity:         "ETH",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  10.0,
		RemainingQuantity: 10.0,
		CostBasis:         20000,
		CostPerUnit:       2000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)
	p.Prices.AddPrice("ETH", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 3000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshot.Holdings) != 2 {
		t.Fatalf("expected 2 holdings, got %d", len(snapshot.Holdings))
	}

	// BTC: 1 * 60000 = 60000
	// ETH: 10 * 3000 = 30000
	// Total: 90000
	if snapshot.TotalMarketValue != 90000 {
		t.Errorf("expected total market value 90000, got %f", snapshot.TotalMarketValue)
	}

	// Cost: 40000 + 20000 = 60000
	if snapshot.TotalCostBasis != 60000 {
		t.Errorf("expected total cost basis 60000, got %f", snapshot.TotalCostBasis)
	}

	// Gain: 90000 - 60000 = 30000
	if snapshot.TotalUnrealizedGain != 30000 {
		t.Errorf("expected total unrealized gain 30000, got %f", snapshot.TotalUnrealizedGain)
	}

	// Allocations: BTC = 60000/90000 = 66.67%, ETH = 30000/90000 = 33.33%
	if !floatEquals(snapshot.Allocations["BTC"], 66.67) {
		t.Errorf("expected BTC allocation ~66.67%%, got %f", snapshot.Allocations["BTC"])
	}
	if !floatEquals(snapshot.Allocations["ETH"], 33.33) {
		t.Errorf("expected ETH allocation ~33.33%%, got %f", snapshot.Allocations["ETH"])
	}
}

func TestSnapshot_ExcludesFutureLots(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	// Future lot - should be excluded
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 45000)

	snapshot, err := p.Snapshot(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	if btc.TotalQuantity != 1.0 {
		t.Errorf("expected quantity 1.0 (excluding future lot), got %f", btc.TotalQuantity)
	}
	if btc.LotCount != 1 {
		t.Errorf("expected 1 lot, got %d", btc.LotCount)
	}
}

func TestSnapshot_ExcludesDisposedLots(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 0, // Fully disposed
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	if btc.TotalQuantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", btc.TotalQuantity)
	}
	if btc.TotalCostBasis != 50000 {
		t.Errorf("expected cost basis 50000, got %f", btc.TotalCostBasis)
	}
	if btc.LotCount != 1 {
		t.Errorf("expected 1 lot (excluding disposed), got %d", btc.LotCount)
	}
}

func TestSnapshot_PartiallyDisposedLot(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 0.5, // Partially disposed
		CostBasis:         80000,
		CostPerUnit:       40000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	if btc.TotalQuantity != 0.5 {
		t.Errorf("expected quantity 0.5, got %f", btc.TotalQuantity)
	}
	// Cost basis for remaining: 0.5 * 40000 = 20000
	if btc.TotalCostBasis != 20000 {
		t.Errorf("expected cost basis 20000, got %f", btc.TotalCostBasis)
	}
	// Market value: 0.5 * 60000 = 30000
	if btc.MarketValue != 30000 {
		t.Errorf("expected market value 30000, got %f", btc.MarketValue)
	}
}

func TestSnapshot_NoPriceError(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// No price added
	_, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for missing price")
	}
}

func TestSnapshot_EmptyPortfolio(t *testing.T) {
	p := NewPortfolio()

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshot.Holdings) != 0 {
		t.Errorf("expected 0 holdings, got %d", len(snapshot.Holdings))
	}
	if snapshot.TotalMarketValue != 0 {
		t.Errorf("expected total market value 0, got %f", snapshot.TotalMarketValue)
	}
}

func TestSnapshot_CashFlows(t *testing.T) {
	p := NewPortfolio()

	p.CashFlows = []CashFlow{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 10000, FlowType: "deposit"},
		{Date: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 5000, FlowType: "deposit"},
		{Date: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), Amount: -2000, FlowType: "withdrawal"},
		{Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Amount: 3000, FlowType: "deposit"}, // After snapshot date
	}

	// Snapshot on April 1 - should include first 3 cash flows
	snapshot, err := p.Snapshot(time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 10000 + 5000 - 2000 = 13000
	if snapshot.CashFlowToDate != 13000 {
		t.Errorf("expected cash flow to date 13000, got %f", snapshot.CashFlowToDate)
	}
}

func TestSnapshot_CachesResult(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cached := p.Snapshots["2024/06/01"]
	if cached != snapshot {
		t.Error("snapshot should be cached")
	}
}

func TestSnapshot_FirstAndLastAcquisition(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 60000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	expectedFirst := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedLast := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	if !btc.FirstAcquisition.Equal(expectedFirst) {
		t.Errorf("expected first acquisition %s, got %s", expectedFirst.Format(DateFormat), btc.FirstAcquisition.Format(DateFormat))
	}
	if !btc.LastAcquisition.Equal(expectedLast) {
		t.Errorf("expected last acquisition %s, got %s", expectedLast.Format(DateFormat), btc.LastAcquisition.Format(DateFormat))
	}
}

func TestSnapshot_UnrealizedLoss(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         60000,
		CostPerUnit:       60000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 40000)

	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	btc := snapshot.Holdings["BTC"]
	// Loss: 40000 - 60000 = -20000
	if btc.UnrealizedGain != -20000 {
		t.Errorf("expected unrealized gain -20000, got %f", btc.UnrealizedGain)
	}
	// Percent: -20000 / 60000 * 100 = -33.33%
	if !floatEquals(btc.UnrealizedGainPercent, -33.33) {
		t.Errorf("expected unrealized gain percent ~-33.33%%, got %f", btc.UnrealizedGainPercent)
	}
}
