package ledger

import (
	"fmt"
	"sort"
	"time"
)

// Lot represents a single purchase of an asset with its cost basis in USD.
// Disposals use FIFO (First In, First Out) - oldest lots are sold first.
type Lot struct {
	Commodity         string    // Asset type (e.g., "BTC", "AAPL")
	AcquisitionDate   time.Time // When purchased
	OriginalQuantity  float64   // Amount originally acquired
	RemainingQuantity float64   // Amount still held (decreases on partial sales)
	CostBasis         float64   // Total cost in USD
	CostPerUnit       float64   // CostBasis / OriginalQuantity (USD per unit)
	Account           string    // Ledger account (e.g., "Assets:Bitcoin")
}

// LotDisposal records a sale/transfer from a specific lot.
type LotDisposal struct {
	Lot              *Lot      // Reference to the lot being disposed
	DisposalDate     time.Time // When disposed
	QuantityDisposed float64   // Amount disposed
	Proceeds         float64   // Sale proceeds in USD
	RealizedGain     float64   // Proceeds - (CostPerUnit * QuantityDisposed)
}

// LotRegistry manages all lots with FIFO ordering.
type LotRegistry struct {
	LotsByCommodity map[string][]*Lot // Commodity -> []*Lot (ordered by acquisition date)
	Disposals       []LotDisposal     // All disposals chronologically
}

// NewLotRegistry creates a new empty LotRegistry.
func NewLotRegistry() *LotRegistry {
	return &LotRegistry{
		LotsByCommodity: make(map[string][]*Lot),
	}
}

// AddLot adds a new lot to the registry, maintaining FIFO order by acquisition date.
func (r *LotRegistry) AddLot(lot *Lot) {
	lots := r.LotsByCommodity[lot.Commodity]
	lots = append(lots, lot)
	// Sort by acquisition date to maintain FIFO order
	sort.Slice(lots, func(i, j int) bool {
		return lots[i].AcquisitionDate.Before(lots[j].AcquisitionDate)
	})
	r.LotsByCommodity[lot.Commodity] = lots
}

// RemainingQuantity returns the total remaining quantity for a commodity.
func (r *LotRegistry) RemainingQuantity(commodity string) float64 {
	var total float64
	for _, lot := range r.LotsByCommodity[commodity] {
		total += lot.RemainingQuantity
	}
	return total
}

// DisposeFIFO disposes of the specified quantity using FIFO (oldest lots first).
// Returns the disposals created and any error.
// The proceeds are allocated proportionally across lots based on quantity disposed.
// The lineNumber parameter is used for error messages to indicate the source line in the ledger file.
func (r *LotRegistry) DisposeFIFO(commodity string, quantity float64, disposalDate time.Time, totalProceeds float64, lineNumber int) ([]LotDisposal, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("ledger: line %d: disposal quantity must be positive, got %f", lineNumber, quantity)
	}

	remaining := r.RemainingQuantity(commodity)
	if quantity > remaining {
		return nil, fmt.Errorf("ledger: line %d: insufficient quantity for %s: want %f, have %f", lineNumber, commodity, quantity, remaining)
	}

	var disposals []LotDisposal
	quantityToDispose := quantity
	proceedsPerUnit := totalProceeds / quantity

	for _, lot := range r.LotsByCommodity[commodity] {
		if quantityToDispose <= 0 {
			break
		}
		if lot.RemainingQuantity <= 0 {
			continue
		}

		// Determine how much to dispose from this lot
		disposeFromLot := lot.RemainingQuantity
		if disposeFromLot > quantityToDispose {
			disposeFromLot = quantityToDispose
		}

		// Calculate proceeds and gain for this disposal
		proceeds := disposeFromLot * proceedsPerUnit
		costOfDisposed := disposeFromLot * lot.CostPerUnit
		realizedGain := proceeds - costOfDisposed

		// Create disposal record
		disposal := LotDisposal{
			Lot:              lot,
			DisposalDate:     disposalDate,
			QuantityDisposed: disposeFromLot,
			Proceeds:         proceeds,
			RealizedGain:     realizedGain,
		}
		disposals = append(disposals, disposal)
		r.Disposals = append(r.Disposals, disposal)

		// Update lot
		lot.RemainingQuantity -= disposeFromLot
		quantityToDispose -= disposeFromLot
	}

	return disposals, nil
}
