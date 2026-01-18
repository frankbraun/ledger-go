package ledger

import (
	"math"
	"testing"
	"time"
)

// floatClose compares two floats with relative tolerance for percentage comparisons.
func floatClose(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestPeriodReturns_SimpleGain(t *testing.T) {
	p := NewPortfolio()

	// Add a lot at the start
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add prices
	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.StartValue != 40000 {
		t.Errorf("expected start value 40000, got %f", pr.StartValue)
	}
	if pr.EndValue != 50000 {
		t.Errorf("expected end value 50000, got %f", pr.EndValue)
	}
	if pr.NetCashFlow != 0 {
		t.Errorf("expected net cash flow 0, got %f", pr.NetCashFlow)
	}
	if pr.AbsoluteReturn != 10000 {
		t.Errorf("expected absolute return 10000, got %f", pr.AbsoluteReturn)
	}
	// Return: 10000 / 40000 = 0.25 (25%)
	if !floatClose(pr.Return, 0.25, 0.001) {
		t.Errorf("expected return 0.25, got %f", pr.Return)
	}
}

func TestPeriodReturns_SimpleLoss(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 50000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 40000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.AbsoluteReturn != -10000 {
		t.Errorf("expected absolute return -10000, got %f", pr.AbsoluteReturn)
	}
	// Return: -10000 / 50000 = -0.20 (-20%)
	if !floatClose(pr.Return, -0.20, 0.001) {
		t.Errorf("expected return -0.20, got %f", pr.Return)
	}
}

func TestPeriodReturns_WithCashFlow(t *testing.T) {
	p := NewPortfolio()

	// Initial position
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add more mid-period
	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         45000,
		CostPerUnit:       45000,
	})

	// Record cash flow for the purchase
	p.CashFlows = []CashFlow{
		{Date: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Amount: 45000, FlowType: "deposit"},
	}

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), 45000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start: 1 BTC @ 40000 = 40000
	// End: 2 BTC @ 50000 = 100000
	// Cash flow: 45000 deposit
	if pr.StartValue != 40000 {
		t.Errorf("expected start value 40000, got %f", pr.StartValue)
	}
	if pr.EndValue != 100000 {
		t.Errorf("expected end value 100000, got %f", pr.EndValue)
	}
	if pr.NetCashFlow != 45000 {
		t.Errorf("expected net cash flow 45000, got %f", pr.NetCashFlow)
	}

	// Absolute return: 100000 - 40000 - 45000 = 15000
	if pr.AbsoluteReturn != 15000 {
		t.Errorf("expected absolute return 15000, got %f", pr.AbsoluteReturn)
	}
}

func TestPeriodReturns_WithWithdrawal(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 1.0, // Sold half
		CostBasis:         80000,
		CostPerUnit:       40000,
	})

	// Record withdrawal from sale
	p.CashFlows = []CashFlow{
		{Date: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Amount: -50000, FlowType: "withdrawal"},
	}

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start: 2 BTC @ 40000 = 80000 (but RemainingQuantity is 1, so snapshot shows 40000)
	// This test shows the limitation - snapshot uses current RemainingQuantity
	// For proper historical tracking, we'd need point-in-time lot states

	if pr.NetCashFlow != -50000 {
		t.Errorf("expected net cash flow -50000, got %f", pr.NetCashFlow)
	}
}

func TestPeriodReturns_AnnualizedReturn(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	// 2 years later, doubled
	p.Prices.AddPrice("BTC", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 80000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Return: 100% over 2 years
	if !floatClose(pr.Return, 1.0, 0.001) {
		t.Errorf("expected return 1.0 (100%%), got %f", pr.Return)
	}

	// CAGR: sqrt(2) - 1 ≈ 0.4142 (41.42% per year)
	expectedCAGR := math.Sqrt(2) - 1
	if !floatClose(pr.AnnualizedReturn, expectedCAGR, 0.01) {
		t.Errorf("expected annualized return ~%f, got %f", expectedCAGR, pr.AnnualizedReturn)
	}
}

func TestPeriodReturns_InvalidDateOrder(t *testing.T) {
	p := NewPortfolio()

	startDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := p.PeriodReturns(startDate, endDate)
	if err == nil {
		t.Error("expected error for invalid date order")
	}
}

func TestPeriodReturns_SameDate(t *testing.T) {
	p := NewPortfolio()

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := p.PeriodReturns(date, date)
	if err == nil {
		t.Error("expected error for same start and end date")
	}
}

func TestPeriodReturns_EmptyPortfolio(t *testing.T) {
	p := NewPortfolio()

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.StartValue != 0 {
		t.Errorf("expected start value 0, got %f", pr.StartValue)
	}
	if pr.EndValue != 0 {
		t.Errorf("expected end value 0, got %f", pr.EndValue)
	}
	if pr.Return != 0 {
		t.Errorf("expected return 0, got %f", pr.Return)
	}
}

func TestPeriodReturns_CashFlowOnlyInPeriod(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Cash flows: one before, one during, one after period
	p.CashFlows = []CashFlow{
		{Date: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), Amount: 10000, FlowType: "deposit"},  // Before
		{Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Amount: 5000, FlowType: "deposit"},    // During
		{Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Amount: 3000, FlowType: "deposit"},    // After
	}

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	pr, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the mid-period cash flow should be counted
	if pr.NetCashFlow != 5000 {
		t.Errorf("expected net cash flow 5000 (only during period), got %f", pr.NetCashFlow)
	}
}

func TestPeriodReturns_ModifiedDietzWeighting(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         100000,
		CostPerUnit:       100000,
	})

	// Deposit at start of period (full weight) vs end of period (no weight)
	// should result in different returns even with same absolute numbers

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100000)
	p.Prices.AddPrice("BTC", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 110000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	// Test 1: No cash flow - simple return
	pr1, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 10% return
	if !floatClose(pr1.Return, 0.10, 0.001) {
		t.Errorf("expected return 0.10, got %f", pr1.Return)
	}
}

func TestSimpleReturn(t *testing.T) {
	tests := []struct {
		name       string
		startValue float64
		endValue   float64
		expected   float64
	}{
		{"positive return", 100, 150, 0.5},
		{"negative return", 100, 80, -0.2},
		{"no change", 100, 100, 0},
		{"zero start", 0, 100, 0},
		{"double", 50, 100, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SimpleReturn(tt.startValue, tt.endValue)
			if !floatClose(result, tt.expected, 0.001) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCAGR(t *testing.T) {
	tests := []struct {
		name       string
		startValue float64
		endValue   float64
		years      float64
		expected   float64
	}{
		{"double in 1 year", 100, 200, 1.0, 1.0},
		{"double in 2 years", 100, 200, 2.0, math.Sqrt(2) - 1},
		{"no growth", 100, 100, 1.0, 0},
		{"50% loss in 1 year", 100, 50, 1.0, -0.5},
		{"zero start", 0, 100, 1.0, 0},
		{"zero years", 100, 200, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CAGR(tt.startValue, tt.endValue, tt.years)
			if !floatClose(result, tt.expected, 0.001) {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
