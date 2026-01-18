package ledger

import (
	"fmt"
	"sort"
	"time"
)

// TimeSeriesPoint represents a single dated value.
type TimeSeriesPoint struct {
	Date  time.Time // Date of the data point
	Value float64   // Value at this date
}

// TimeSeries stores a sequence of dated values.
type TimeSeries struct {
	Name   string            // e.g., "Portfolio Value", "BTC Holdings"
	Points []TimeSeriesPoint // Sorted by date
}

// NewTimeSeries creates a new empty TimeSeries with the given name.
func NewTimeSeries(name string) *TimeSeries {
	return &TimeSeries{
		Name: name,
	}
}

// PortfolioTimeSeries tracks multiple metrics over time.
type PortfolioTimeSeries struct {
	Value          *TimeSeries            // Total portfolio value (USD)
	CostBasis      *TimeSeries            // Total cost basis (USD)
	UnrealizedGain *TimeSeries            // Unrealized gains (USD)
	RealizedGain   *TimeSeries            // Cumulative realized gains (USD)
	Holdings       map[string]*TimeSeries // Quantity per commodity
	HoldingValues  map[string]*TimeSeries // Value per commodity (USD)
}

// NewPortfolioTimeSeries creates a new PortfolioTimeSeries with initialized fields.
func NewPortfolioTimeSeries() *PortfolioTimeSeries {
	return &PortfolioTimeSeries{
		Value:          NewTimeSeries("Portfolio Value"),
		CostBasis:      NewTimeSeries("Cost Basis"),
		UnrealizedGain: NewTimeSeries("Unrealized Gain"),
		RealizedGain:   NewTimeSeries("Realized Gain"),
		Holdings:       make(map[string]*TimeSeries),
		HoldingValues:  make(map[string]*TimeSeries),
	}
}

// Add adds a point to the time series, maintaining chronological order.
func (ts *TimeSeries) Add(date time.Time, value float64) {
	point := TimeSeriesPoint{Date: date, Value: value}
	ts.Points = append(ts.Points, point)
	sort.Slice(ts.Points, func(i, j int) bool {
		return ts.Points[i].Date.Before(ts.Points[j].Date)
	})
}

// Get returns the value at the given date.
// If no exact match exists, returns the most recent value before that date.
// Returns an error if no value is available on or before the given date.
func (ts *TimeSeries) Get(date time.Time) (float64, error) {
	if len(ts.Points) == 0 {
		return 0, fmt.Errorf("time series %q is empty", ts.Name)
	}

	idx := sort.Search(len(ts.Points), func(i int) bool {
		return ts.Points[i].Date.After(date)
	})

	if idx == 0 {
		return 0, fmt.Errorf("no value in %q on or before %s", ts.Name, date.Format(DateFormat))
	}

	return ts.Points[idx-1].Value, nil
}

// First returns the first (earliest) point in the series.
// Returns an error if the series is empty.
func (ts *TimeSeries) First() (TimeSeriesPoint, error) {
	if len(ts.Points) == 0 {
		return TimeSeriesPoint{}, fmt.Errorf("time series %q is empty", ts.Name)
	}
	return ts.Points[0], nil
}

// Last returns the last (most recent) point in the series.
// Returns an error if the series is empty.
func (ts *TimeSeries) Last() (TimeSeriesPoint, error) {
	if len(ts.Points) == 0 {
		return TimeSeriesPoint{}, fmt.Errorf("time series %q is empty", ts.Name)
	}
	return ts.Points[len(ts.Points)-1], nil
}

// Len returns the number of points in the series.
func (ts *TimeSeries) Len() int {
	return len(ts.Points)
}

// Interval represents the frequency for time series generation.
type Interval int

const (
	Daily Interval = iota
	Weekly
	Monthly
)

// nextDate returns the next date based on the interval.
func nextDate(date time.Time, interval Interval) time.Time {
	switch interval {
	case Daily:
		return date.AddDate(0, 0, 1)
	case Weekly:
		return date.AddDate(0, 0, 7)
	case Monthly:
		return date.AddDate(0, 1, 0)
	default:
		return date.AddDate(0, 0, 1)
	}
}

// BuildTimeSeries generates a PortfolioTimeSeries from startDate to endDate
// at the specified interval. It captures portfolio snapshots at each date.
func (p *Portfolio) BuildTimeSeries(startDate, endDate time.Time, interval Interval) (*PortfolioTimeSeries, error) {
	if !startDate.Before(endDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	pts := NewPortfolioTimeSeries()

	// Track cumulative realized gains
	var cumulativeRealizedGain float64

	// Iterate through dates
	for date := startDate; !date.After(endDate); date = nextDate(date, interval) {
		snapshot, err := p.Snapshot(date)
		if err != nil {
			// Skip dates where we can't get a snapshot (e.g., no prices)
			continue
		}

		// Add portfolio-level metrics
		pts.Value.Add(date, snapshot.TotalMarketValue)
		pts.CostBasis.Add(date, snapshot.TotalCostBasis)
		pts.UnrealizedGain.Add(date, snapshot.TotalUnrealizedGain)

		// Calculate realized gains up to this date
		cumulativeRealizedGain = 0
		for _, disposal := range p.Lots.Disposals {
			if !disposal.DisposalDate.After(date) {
				cumulativeRealizedGain += disposal.RealizedGain
			}
		}
		pts.RealizedGain.Add(date, cumulativeRealizedGain)

		// Add per-commodity metrics
		for commodity, holding := range snapshot.Holdings {
			// Holdings quantity
			if pts.Holdings[commodity] == nil {
				pts.Holdings[commodity] = NewTimeSeries(commodity + " Holdings")
			}
			pts.Holdings[commodity].Add(date, holding.TotalQuantity)

			// Holdings value
			if pts.HoldingValues[commodity] == nil {
				pts.HoldingValues[commodity] = NewTimeSeries(commodity + " Value")
			}
			pts.HoldingValues[commodity].Add(date, holding.MarketValue)
		}
	}

	return pts, nil
}

// ValueChange calculates the change in value between two dates in the series.
func (ts *TimeSeries) ValueChange(startDate, endDate time.Time) (float64, error) {
	startVal, err := ts.Get(startDate)
	if err != nil {
		return 0, err
	}
	endVal, err := ts.Get(endDate)
	if err != nil {
		return 0, err
	}
	return endVal - startVal, nil
}

// PercentChange calculates the percentage change between two dates.
func (ts *TimeSeries) PercentChange(startDate, endDate time.Time) (float64, error) {
	startVal, err := ts.Get(startDate)
	if err != nil {
		return 0, err
	}
	if startVal == 0 {
		return 0, fmt.Errorf("cannot calculate percent change from zero")
	}
	endVal, err := ts.Get(endDate)
	if err != nil {
		return 0, err
	}
	return (endVal - startVal) / startVal * 100, nil
}

// Max returns the maximum value and its date in the series.
func (ts *TimeSeries) Max() (TimeSeriesPoint, error) {
	if len(ts.Points) == 0 {
		return TimeSeriesPoint{}, fmt.Errorf("time series %q is empty", ts.Name)
	}
	max := ts.Points[0]
	for _, p := range ts.Points[1:] {
		if p.Value > max.Value {
			max = p
		}
	}
	return max, nil
}

// Min returns the minimum value and its date in the series.
func (ts *TimeSeries) Min() (TimeSeriesPoint, error) {
	if len(ts.Points) == 0 {
		return TimeSeriesPoint{}, fmt.Errorf("time series %q is empty", ts.Name)
	}
	min := ts.Points[0]
	for _, p := range ts.Points[1:] {
		if p.Value < min.Value {
			min = p
		}
	}
	return min, nil
}
