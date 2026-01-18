package ledger

import (
	"testing"
	"time"
)

func TestTimeSeries_Add(t *testing.T) {
	ts := NewTimeSeries("Test")

	// Add out of order
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 300)
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 200)

	if ts.Len() != 3 {
		t.Fatalf("expected 3 points, got %d", ts.Len())
	}

	// Verify chronological order
	if ts.Points[0].Value != 100 {
		t.Errorf("first point should be 100, got %f", ts.Points[0].Value)
	}
	if ts.Points[1].Value != 200 {
		t.Errorf("second point should be 200, got %f", ts.Points[1].Value)
	}
	if ts.Points[2].Value != 300 {
		t.Errorf("third point should be 300, got %f", ts.Points[2].Value)
	}
}

func TestTimeSeries_Get_ExactMatch(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), 150)

	val, err := ts.Get(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 150 {
		t.Errorf("expected 150, got %f", val)
	}
}

func TestTimeSeries_Get_Interpolation(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 300)

	// Query for date between points - should return earlier value
	val, err := ts.Get(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100 (earlier value), got %f", val)
	}
}

func TestTimeSeries_Get_AfterLast(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)

	// Query for date after last point
	val, err := ts.Get(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100 (last value), got %f", val)
	}
}

func TestTimeSeries_Get_BeforeFirst(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 100)

	_, err := ts.Get(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for date before first point")
	}
}

func TestTimeSeries_Get_Empty(t *testing.T) {
	ts := NewTimeSeries("Test")

	_, err := ts.Get(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for empty series")
	}
}

func TestTimeSeries_First(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 300)
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)

	first, err := ts.First()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Value != 100 {
		t.Errorf("expected first value 100, got %f", first.Value)
	}
}

func TestTimeSeries_Last(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 300)

	last, err := ts.Last()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if last.Value != 300 {
		t.Errorf("expected last value 300, got %f", last.Value)
	}
}

func TestTimeSeries_FirstLast_Empty(t *testing.T) {
	ts := NewTimeSeries("Test")

	_, err := ts.First()
	if err == nil {
		t.Error("expected error for First on empty series")
	}

	_, err = ts.Last()
	if err == nil {
		t.Error("expected error for Last on empty series")
	}
}

func TestTimeSeries_ValueChange(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 150)

	change, err := ts.ValueChange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change != 50 {
		t.Errorf("expected change 50, got %f", change)
	}
}

func TestTimeSeries_PercentChange(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 150)

	pct, err := ts.PercentChange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pct != 50 {
		t.Errorf("expected 50%%, got %f", pct)
	}
}

func TestTimeSeries_PercentChange_FromZero(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 0)
	ts.Add(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 100)

	_, err := ts.PercentChange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Error("expected error for percent change from zero")
	}
}

func TestTimeSeries_Max(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 300)
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 200)

	max, err := ts.Max()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if max.Value != 300 {
		t.Errorf("expected max 300, got %f", max.Value)
	}
	if !max.Date.Equal(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected max date 2024/02/01, got %s", max.Date.Format(DateFormat))
	}
}

func TestTimeSeries_Min(t *testing.T) {
	ts := NewTimeSeries("Test")
	ts.Add(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 200)
	ts.Add(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 100)
	ts.Add(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 300)

	min, err := ts.Min()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if min.Value != 100 {
		t.Errorf("expected min 100, got %f", min.Value)
	}
}

func TestTimeSeries_MaxMin_Empty(t *testing.T) {
	ts := NewTimeSeries("Test")

	_, err := ts.Max()
	if err == nil {
		t.Error("expected error for Max on empty series")
	}

	_, err = ts.Min()
	if err == nil {
		t.Error("expected error for Min on empty series")
	}
}

func TestBuildTimeSeries_Daily(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add daily prices for a week
	for i := 0; i < 7; i++ {
		date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		price := 40000 + float64(i)*1000 // 40000, 41000, 42000, ...
		p.Prices.AddPrice("BTC", date, price)
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Daily)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pts.Value.Len() != 7 {
		t.Errorf("expected 7 value points, got %d", pts.Value.Len())
	}

	// Check first and last values
	first, _ := pts.Value.First()
	if first.Value != 40000 {
		t.Errorf("expected first value 40000, got %f", first.Value)
	}

	last, _ := pts.Value.Last()
	if last.Value != 46000 {
		t.Errorf("expected last value 46000, got %f", last.Value)
	}
}

func TestBuildTimeSeries_Monthly(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add monthly prices for 6 months
	for i := 0; i < 6; i++ {
		date := time.Date(2024, time.Month(1+i), 1, 0, 0, 0, 0, time.UTC)
		price := 40000 + float64(i)*5000
		p.Prices.AddPrice("BTC", date, price)
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Monthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pts.Value.Len() != 6 {
		t.Errorf("expected 6 value points, got %d", pts.Value.Len())
	}
}

func TestBuildTimeSeries_MultipleCommodities(t *testing.T) {
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

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("ETH", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 2000)
	p.Prices.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 45000)
	p.Prices.AddPrice("ETH", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 2500)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Monthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check holdings exist for both commodities
	if pts.Holdings["BTC"] == nil {
		t.Error("expected BTC holdings time series")
	}
	if pts.Holdings["ETH"] == nil {
		t.Error("expected ETH holdings time series")
	}

	// Check holding values
	if pts.HoldingValues["BTC"] == nil {
		t.Error("expected BTC holding values time series")
	}
	if pts.HoldingValues["ETH"] == nil {
		t.Error("expected ETH holding values time series")
	}

	// Verify BTC holding quantity stays constant
	btcHolding, _ := pts.Holdings["BTC"].Last()
	if btcHolding.Value != 1.0 {
		t.Errorf("expected BTC holding 1.0, got %f", btcHolding.Value)
	}

	// Verify BTC value increased
	btcValue, _ := pts.HoldingValues["BTC"].Last()
	if btcValue.Value != 45000 {
		t.Errorf("expected BTC value 45000, got %f", btcValue.Value)
	}
}

func TestBuildTimeSeries_WithRealizedGains(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 2.0,
		CostBasis:         80000,
		CostPerUnit:       40000,
	})

	p.Prices.AddPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 40000)
	p.Prices.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)
	p.Prices.AddPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), 55000)

	// Dispose 1 BTC in February
	p.Lots.DisposeFIFO("BTC", 1.0, time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Monthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// January: no realized gains
	janGain, _ := pts.RealizedGain.Get(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if janGain != 0 {
		t.Errorf("expected Jan realized gain 0, got %f", janGain)
	}

	// March: should show realized gain from February sale
	// Gain = 50000 - 40000 = 10000
	marGain, _ := pts.RealizedGain.Get(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if marGain != 10000 {
		t.Errorf("expected Mar realized gain 10000, got %f", marGain)
	}
}

func TestBuildTimeSeries_CostBasisAndUnrealizedGain(t *testing.T) {
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
	p.Prices.AddPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), 50000)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Monthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cost basis should stay constant
	janCost, _ := pts.CostBasis.Get(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	febCost, _ := pts.CostBasis.Get(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if janCost != 40000 || febCost != 40000 {
		t.Errorf("expected cost basis 40000, got jan=%f, feb=%f", janCost, febCost)
	}

	// Unrealized gain should increase
	janGain, _ := pts.UnrealizedGain.Get(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	febGain, _ := pts.UnrealizedGain.Get(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if janGain != 0 {
		t.Errorf("expected Jan unrealized gain 0, got %f", janGain)
	}
	if febGain != 10000 {
		t.Errorf("expected Feb unrealized gain 10000, got %f", febGain)
	}
}

func TestBuildTimeSeries_InvalidDates(t *testing.T) {
	p := NewPortfolio()

	startDate := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := p.BuildTimeSeries(startDate, endDate, Daily)
	if err == nil {
		t.Error("expected error for invalid date order")
	}
}

func TestBuildTimeSeries_Weekly(t *testing.T) {
	p := NewPortfolio()

	p.Lots.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	// Add prices for 4 weeks
	for i := 0; i < 28; i++ {
		date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		p.Prices.AddPrice("BTC", date, 40000+float64(i)*100)
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 28, 0, 0, 0, 0, time.UTC)

	pts, err := p.BuildTimeSeries(startDate, endDate, Weekly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4 weeks: Jan 1, 8, 15, 22
	if pts.Value.Len() != 4 {
		t.Errorf("expected 4 weekly points, got %d", pts.Value.Len())
	}
}
