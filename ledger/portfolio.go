package ledger

import (
	"fmt"
	"io"
	"sort"
	"time"
)

// CommodityHolding aggregates position in a single commodity.
type CommodityHolding struct {
	Commodity             string    // Asset type (e.g., "BTC", "AAPL")
	TotalQuantity         float64   // Total units held
	TotalCostBasis        float64   // Total cost in USD
	WeightedAverageCost   float64   // TotalCostBasis / TotalQuantity (USD per unit)
	MarketValue           float64   // TotalQuantity * current price (USD)
	UnrealizedGain        float64   // MarketValue - TotalCostBasis
	UnrealizedGainPercent float64   // (UnrealizedGain / TotalCostBasis) * 100
	LotCount              int       // Number of lots making up this holding
	FirstAcquisition      time.Time // Date of earliest lot
	LastAcquisition       time.Time // Date of most recent lot
}

// PortfolioSnapshot represents complete portfolio state at a point in time.
type PortfolioSnapshot struct {
	Date                       time.Time                  // Snapshot date
	Holdings                   map[string]*CommodityHolding // Commodity -> holding
	TotalMarketValue           float64                    // Sum of all holdings market values (USD)
	TotalCostBasis             float64                    // Sum of all holdings cost basis (USD)
	TotalUnrealizedGain        float64                    // TotalMarketValue - TotalCostBasis
	TotalUnrealizedGainPercent float64                    // (TotalUnrealizedGain / TotalCostBasis) * 100
	CashFlowToDate             float64                    // Net cash invested (USD)
	Allocations                map[string]float64         // Commodity -> % of portfolio
}

// CashFlow records cash movement into/out of portfolio (in USD).
type CashFlow struct {
	Date     time.Time // Date of cash flow
	Amount   float64   // Positive = deposit, negative = withdrawal (USD)
	FlowType string    // "deposit", "withdrawal", "dividend", "interest"
}

// Portfolio manages complete portfolio state with historical tracking.
type Portfolio struct {
	Lots          *LotRegistry                  // All lots
	Prices        *PriceHistory                 // Historical prices
	Snapshots     map[string]*PortfolioSnapshot // Date string -> snapshot
	CashFlows     []CashFlow                    // All cash flows chronologically
	AssetAccounts []string                      // Account prefixes for investable assets
}

// NewPortfolio creates a new empty Portfolio.
func NewPortfolio() *Portfolio {
	return &Portfolio{
		Lots:      NewLotRegistry(),
		Prices:    NewPriceHistory(),
		Snapshots: make(map[string]*PortfolioSnapshot),
	}
}

// Snapshot calculates the portfolio state at the given date.
// It aggregates all lots acquired on or before the date, using current remaining quantities.
// Prices are looked up from the price history for market value calculation.
func (p *Portfolio) Snapshot(date time.Time) (*PortfolioSnapshot, error) {
	snapshot := &PortfolioSnapshot{
		Date:        date,
		Holdings:    make(map[string]*CommodityHolding),
		Allocations: make(map[string]float64),
	}

	// Aggregate lots by commodity
	for commodity, lots := range p.Lots.LotsByCommodity {
		var holding *CommodityHolding

		for _, lot := range lots {
			// Skip lots acquired after the snapshot date
			if lot.AcquisitionDate.After(date) {
				continue
			}
			// Skip fully disposed lots
			if lot.RemainingQuantity <= 0 {
				continue
			}

			// Initialize holding on first lot
			if holding == nil {
				holding = &CommodityHolding{
					Commodity:        commodity,
					FirstAcquisition: lot.AcquisitionDate,
				}
			}

			// Calculate cost basis for remaining quantity
			remainingCostBasis := lot.CostPerUnit * lot.RemainingQuantity

			holding.TotalQuantity += lot.RemainingQuantity
			holding.TotalCostBasis += remainingCostBasis
			holding.LotCount++
			holding.LastAcquisition = lot.AcquisitionDate
		}

		if holding != nil && holding.TotalQuantity > 0 {
			// Calculate weighted average cost
			holding.WeightedAverageCost = holding.TotalCostBasis / holding.TotalQuantity

			// Look up price for market value
			price, err := p.Prices.GetPrice(commodity, date)
			if err != nil {
				return nil, err
			}

			holding.MarketValue = holding.TotalQuantity * price
			holding.UnrealizedGain = holding.MarketValue - holding.TotalCostBasis
			if holding.TotalCostBasis > 0 {
				holding.UnrealizedGainPercent = (holding.UnrealizedGain / holding.TotalCostBasis) * 100
			}

			snapshot.Holdings[commodity] = holding
			snapshot.TotalMarketValue += holding.MarketValue
			snapshot.TotalCostBasis += holding.TotalCostBasis
		}
	}

	// Calculate portfolio-level metrics
	snapshot.TotalUnrealizedGain = snapshot.TotalMarketValue - snapshot.TotalCostBasis
	if snapshot.TotalCostBasis > 0 {
		snapshot.TotalUnrealizedGainPercent = (snapshot.TotalUnrealizedGain / snapshot.TotalCostBasis) * 100
	}

	// Calculate allocations
	if snapshot.TotalMarketValue > 0 {
		for commodity, holding := range snapshot.Holdings {
			snapshot.Allocations[commodity] = (holding.MarketValue / snapshot.TotalMarketValue) * 100
		}
	}

	// Calculate cash flow to date
	for _, cf := range p.CashFlows {
		if !cf.Date.After(date) {
			snapshot.CashFlowToDate += cf.Amount
		}
	}

	// Cache the snapshot
	p.Snapshots[date.Format(DateFormat)] = snapshot

	return snapshot, nil
}

// formatMoney formats a float64 as a dollar amount with commas.
func formatMoney(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	// Format with 2 decimal places
	s := fmt.Sprintf("%.2f", amount)

	// Split into integer and decimal parts
	parts := make([]byte, 0, len(s)+4)
	dotIndex := len(s) - 3 // position of the decimal point

	// Add commas to integer part
	intPart := s[:dotIndex]
	for i, c := range intPart {
		if i > 0 && (dotIndex-i)%3 == 0 {
			parts = append(parts, ',')
		}
		parts = append(parts, byte(c))
	}

	// Add decimal part
	parts = append(parts, s[dotIndex:]...)

	if negative {
		return "-$" + string(parts)
	}
	return "$" + string(parts)
}

// formatPercent formats a float64 as a percentage.
func formatPercent(pct float64) string {
	return fmt.Sprintf("%.2f%%", pct)
}

// Print writes a formatted representation of the holding to w.
func (h *CommodityHolding) Print(w io.Writer) {
	fmt.Fprintf(w, "  %s\n", h.Commodity)
	fmt.Fprintf(w, "    Quantity:           %s\n", formatQuantity(h.TotalQuantity))
	fmt.Fprintf(w, "    Cost Basis:         %s\n", formatMoney(h.TotalCostBasis))
	fmt.Fprintf(w, "    Market Value:       %s\n", formatMoney(h.MarketValue))
	fmt.Fprintf(w, "    Unrealized Gain:    %s (%s)\n",
		formatMoney(h.UnrealizedGain), formatPercent(h.UnrealizedGainPercent))
	fmt.Fprintf(w, "    Avg Cost/Unit:      %s\n", formatMoney(h.WeightedAverageCost))
	fmt.Fprintf(w, "    Lots:               %d\n", h.LotCount)
	fmt.Fprintf(w, "    First Acquired:     %s\n", h.FirstAcquisition.Format(DateFormat))
	fmt.Fprintf(w, "    Last Acquired:      %s\n", h.LastAcquisition.Format(DateFormat))
}

// formatQuantity formats a quantity with appropriate decimal places.
func formatQuantity(q float64) string {
	// Use up to 8 decimal places, but trim trailing zeros
	s := fmt.Sprintf("%.8f", q)
	// Trim trailing zeros after decimal point
	for len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] != '.' {
		s = s[:len(s)-1]
	}
	return s
}

// Print writes a formatted representation of the portfolio snapshot to w.
func (s *PortfolioSnapshot) Print(w io.Writer) {
	fmt.Fprintf(w, "Portfolio Snapshot as of %s\n", s.Date.Format(DateFormat))
	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintln(w)

	if len(s.Holdings) == 0 {
		fmt.Fprintln(w, "No holdings.")
		return
	}

	fmt.Fprintln(w, "Holdings:")

	// Sort commodities alphabetically
	commodities := make([]string, 0, len(s.Holdings))
	for c := range s.Holdings {
		commodities = append(commodities, c)
	}
	sort.Strings(commodities)

	for i, c := range commodities {
		s.Holdings[c].Print(w)
		if i < len(commodities)-1 {
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  Total Cost Basis:     %s\n", formatMoney(s.TotalCostBasis))
	fmt.Fprintf(w, "  Total Market Value:   %s\n", formatMoney(s.TotalMarketValue))
	fmt.Fprintf(w, "  Total Unrealized:     %s (%s)\n",
		formatMoney(s.TotalUnrealizedGain), formatPercent(s.TotalUnrealizedGainPercent))

	if len(s.Allocations) > 0 {
		fmt.Fprintln(w, "  Allocation:")
		for _, c := range commodities {
			fmt.Fprintf(w, "    %s:  %s\n", c, formatPercent(s.Allocations[c]))
		}
	}

	fmt.Fprintln(w, "================================================================================")
}
