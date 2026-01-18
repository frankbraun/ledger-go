package ledger

import (
	"fmt"
	"math"
	"time"
)

// PeriodReturn represents performance over a specific time period.
type PeriodReturn struct {
	StartDate        time.Time // Beginning of period
	EndDate          time.Time // End of period
	StartValue       float64   // Portfolio value at start (USD)
	EndValue         float64   // Portfolio value at end (USD)
	NetCashFlow      float64   // Net deposits minus withdrawals during period (USD)
	AbsoluteReturn   float64   // EndValue - StartValue - NetCashFlow
	Return           float64   // MWR/IRR: actual investor experience
	AnnualizedReturn float64   // CAGR
}

// AssetPerformance tracks performance for a single commodity.
type AssetPerformance struct {
	Commodity      string       // Asset type (e.g., "BTC", "AAPL")
	Period         PeriodReturn // Performance metrics for the period
	RealizedGain   float64      // Net realized gains/losses (USD)
	UnrealizedGain float64      // Current unrealized gain (USD)
	TotalReturn    float64      // RealizedGain + UnrealizedGain
	Contribution   float64      // Contribution to portfolio return
}

// PerformanceReport aggregates all metrics.
type PerformanceReport struct {
	GeneratedAt      time.Time                     // When report was generated
	ReportPeriod     PeriodReturn                  // Overall portfolio performance
	AssetPerformance map[string]*AssetPerformance  // Commodity -> performance
}

// NewPerformanceReport creates a new empty PerformanceReport.
func NewPerformanceReport() *PerformanceReport {
	return &PerformanceReport{
		GeneratedAt:      time.Now(),
		AssetPerformance: make(map[string]*AssetPerformance),
	}
}

// PeriodReturns calculates performance metrics between two dates.
// Uses the Modified Dietz method to calculate money-weighted returns.
func (p *Portfolio) PeriodReturns(startDate, endDate time.Time) (*PeriodReturn, error) {
	if !startDate.Before(endDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	// Get portfolio snapshots at start and end
	startSnapshot, err := p.Snapshot(startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get start snapshot: %w", err)
	}

	endSnapshot, err := p.Snapshot(endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get end snapshot: %w", err)
	}

	pr := &PeriodReturn{
		StartDate:  startDate,
		EndDate:    endDate,
		StartValue: startSnapshot.TotalMarketValue,
		EndValue:   endSnapshot.TotalMarketValue,
	}

	// Calculate net cash flow and weighted cash flow during the period
	totalDays := endDate.Sub(startDate).Hours() / 24
	var weightedCashFlow float64

	for _, cf := range p.CashFlows {
		if cf.Date.After(startDate) && !cf.Date.After(endDate) {
			pr.NetCashFlow += cf.Amount

			// Weight by proportion of period remaining
			daysRemaining := endDate.Sub(cf.Date).Hours() / 24
			weight := daysRemaining / totalDays
			weightedCashFlow += cf.Amount * weight
		}
	}

	// Absolute return: gain/loss after accounting for cash flows
	pr.AbsoluteReturn = pr.EndValue - pr.StartValue - pr.NetCashFlow

	// Modified Dietz return
	// Return = (EndValue - StartValue - NetCashFlow) / (StartValue + WeightedCashFlow)
	denominator := pr.StartValue + weightedCashFlow
	if denominator > 0 {
		pr.Return = pr.AbsoluteReturn / denominator
	} else if pr.AbsoluteReturn != 0 {
		// Handle edge case: no starting value but there's a return
		// This can happen with initial deposits
		pr.Return = pr.AbsoluteReturn / pr.NetCashFlow
	}

	// Annualized return (CAGR)
	// CAGR = (1 + Return)^(365/days) - 1
	years := totalDays / 365.0
	if years > 0 && pr.Return > -1 {
		pr.AnnualizedReturn = math.Pow(1+pr.Return, 1/years) - 1
	}

	return pr, nil
}

// SimpleReturn calculates the simple return percentage between two values.
// Returns (endValue - startValue) / startValue.
func SimpleReturn(startValue, endValue float64) float64 {
	if startValue == 0 {
		return 0
	}
	return (endValue - startValue) / startValue
}

// CAGR calculates the Compound Annual Growth Rate.
// Returns the annualized return given start value, end value, and number of years.
func CAGR(startValue, endValue, years float64) float64 {
	if startValue <= 0 || years <= 0 {
		return 0
	}
	return math.Pow(endValue/startValue, 1/years) - 1
}
