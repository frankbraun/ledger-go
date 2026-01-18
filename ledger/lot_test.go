package ledger

import (
	"testing"
	"time"
)

func TestAddLot(t *testing.T) {
	r := NewLotRegistry()

	// Add lots out of order
	lot2 := &Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	}
	lot1 := &Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 2.0,
		CostBasis:         80000,
		CostPerUnit:       40000,
	}
	lot3 := &Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  0.5,
		RemainingQuantity: 0.5,
		CostBasis:         30000,
		CostPerUnit:       60000,
	}

	r.AddLot(lot2)
	r.AddLot(lot1)
	r.AddLot(lot3)

	// Verify FIFO ordering (sorted by acquisition date)
	lots := r.LotsByCommodity["BTC"]
	if len(lots) != 3 {
		t.Fatalf("expected 3 lots, got %d", len(lots))
	}
	if lots[0] != lot1 {
		t.Error("first lot should be lot1 (earliest acquisition)")
	}
	if lots[1] != lot2 {
		t.Error("second lot should be lot2")
	}
	if lots[2] != lot3 {
		t.Error("third lot should be lot3 (latest acquisition)")
	}
}

func TestRemainingQuantity(t *testing.T) {
	r := NewLotRegistry()

	// No lots
	if q := r.RemainingQuantity("BTC"); q != 0 {
		t.Errorf("expected 0 for unknown commodity, got %f", q)
	}

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 2.0,
		CostBasis:         80000,
		CostPerUnit:       40000,
	})
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.5,
		RemainingQuantity: 1.5,
		CostBasis:         75000,
		CostPerUnit:       50000,
	})

	if q := r.RemainingQuantity("BTC"); q != 3.5 {
		t.Errorf("expected 3.5, got %f", q)
	}
}

func TestDisposeFIFO_FullLotDisposal(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	disposals, err := r.DisposeFIFO("BTC", 1.0, disposalDate, 50000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disposals) != 1 {
		t.Fatalf("expected 1 disposal, got %d", len(disposals))
	}

	d := disposals[0]
	if d.QuantityDisposed != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", d.QuantityDisposed)
	}
	if d.Proceeds != 50000 {
		t.Errorf("expected proceeds 50000, got %f", d.Proceeds)
	}
	if d.RealizedGain != 10000 { // 50000 - 40000
		t.Errorf("expected realized gain 10000, got %f", d.RealizedGain)
	}
	if d.Lot.RemainingQuantity != 0 {
		t.Errorf("expected remaining quantity 0, got %f", d.Lot.RemainingQuantity)
	}
}

func TestDisposeFIFO_PartialLotDisposal(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 2.0,
		CostBasis:         80000,
		CostPerUnit:       40000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	disposals, err := r.DisposeFIFO("BTC", 0.5, disposalDate, 25000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disposals) != 1 {
		t.Fatalf("expected 1 disposal, got %d", len(disposals))
	}

	d := disposals[0]
	if d.QuantityDisposed != 0.5 {
		t.Errorf("expected quantity 0.5, got %f", d.QuantityDisposed)
	}
	if d.Proceeds != 25000 {
		t.Errorf("expected proceeds 25000, got %f", d.Proceeds)
	}
	// Cost: 0.5 * 40000 = 20000, Gain: 25000 - 20000 = 5000
	if d.RealizedGain != 5000 {
		t.Errorf("expected realized gain 5000, got %f", d.RealizedGain)
	}
	if d.Lot.RemainingQuantity != 1.5 {
		t.Errorf("expected remaining quantity 1.5, got %f", d.Lot.RemainingQuantity)
	}
}

func TestDisposeFIFO_MultipleLots(t *testing.T) {
	r := NewLotRegistry()

	// Lot 1: 1 BTC at $40,000
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	// Lot 2: 1 BTC at $50,000
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})
	// Lot 3: 1 BTC at $60,000
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         60000,
		CostPerUnit:       60000,
	})

	// Dispose 1.5 BTC at $55,000 per BTC ($82,500 total)
	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	disposals, err := r.DisposeFIFO("BTC", 1.5, disposalDate, 82500, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disposals) != 2 {
		t.Fatalf("expected 2 disposals, got %d", len(disposals))
	}

	// First disposal: full lot 1 (1.0 BTC)
	d1 := disposals[0]
	if d1.QuantityDisposed != 1.0 {
		t.Errorf("disposal 1: expected quantity 1.0, got %f", d1.QuantityDisposed)
	}
	// Proceeds per unit: 82500 / 1.5 = 55000
	if d1.Proceeds != 55000 {
		t.Errorf("disposal 1: expected proceeds 55000, got %f", d1.Proceeds)
	}
	// Gain: 55000 - 40000 = 15000
	if d1.RealizedGain != 15000 {
		t.Errorf("disposal 1: expected realized gain 15000, got %f", d1.RealizedGain)
	}

	// Second disposal: partial lot 2 (0.5 BTC)
	d2 := disposals[1]
	if d2.QuantityDisposed != 0.5 {
		t.Errorf("disposal 2: expected quantity 0.5, got %f", d2.QuantityDisposed)
	}
	// Proceeds: 0.5 * 55000 = 27500
	if d2.Proceeds != 27500 {
		t.Errorf("disposal 2: expected proceeds 27500, got %f", d2.Proceeds)
	}
	// Cost: 0.5 * 50000 = 25000, Gain: 27500 - 25000 = 2500
	if d2.RealizedGain != 2500 {
		t.Errorf("disposal 2: expected realized gain 2500, got %f", d2.RealizedGain)
	}

	// Verify remaining quantities
	lots := r.LotsByCommodity["BTC"]
	if lots[0].RemainingQuantity != 0 {
		t.Errorf("lot 1: expected remaining 0, got %f", lots[0].RemainingQuantity)
	}
	if lots[1].RemainingQuantity != 0.5 {
		t.Errorf("lot 2: expected remaining 0.5, got %f", lots[1].RemainingQuantity)
	}
	if lots[2].RemainingQuantity != 1.0 {
		t.Errorf("lot 3: expected remaining 1.0, got %f", lots[2].RemainingQuantity)
	}

	// Verify total remaining
	if q := r.RemainingQuantity("BTC"); q != 1.5 {
		t.Errorf("expected remaining 1.5, got %f", q)
	}
}

func TestDisposeFIFO_InsufficientQuantity(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := r.DisposeFIFO("BTC", 2.0, disposalDate, 100000, 0)
	if err == nil {
		t.Error("expected error for insufficient quantity")
	}
}

func TestDisposeFIFO_ZeroQuantity(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := r.DisposeFIFO("BTC", 0, disposalDate, 0, 0)
	if err == nil {
		t.Error("expected error for zero quantity")
	}
}

func TestDisposeFIFO_NegativeQuantity(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         40000,
		CostPerUnit:       40000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := r.DisposeFIFO("BTC", -1.0, disposalDate, 50000, 0)
	if err == nil {
		t.Error("expected error for negative quantity")
	}
}

func TestDisposeFIFO_RealizedLoss(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	// Sell at a loss
	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	disposals, err := r.DisposeFIFO("BTC", 1.0, disposalDate, 30000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	d := disposals[0]
	// Loss: 30000 - 50000 = -20000
	if d.RealizedGain != -20000 {
		t.Errorf("expected realized loss -20000, got %f", d.RealizedGain)
	}
}

func TestDisposeFIFO_DisposalsAppendedToRegistry(t *testing.T) {
	r := NewLotRegistry()

	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  2.0,
		RemainingQuantity: 2.0,
		CostBasis:         80000,
		CostPerUnit:       40000,
	})

	// First disposal
	date1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := r.DisposeFIFO("BTC", 0.5, date1, 25000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second disposal
	date2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err = r.DisposeFIFO("BTC", 0.5, date2, 30000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Disposals) != 2 {
		t.Errorf("expected 2 disposals in registry, got %d", len(r.Disposals))
	}

	if !r.Disposals[0].DisposalDate.Equal(date1) {
		t.Error("first disposal should have date1")
	}
	if !r.Disposals[1].DisposalDate.Equal(date2) {
		t.Error("second disposal should have date2")
	}
}

func TestDisposeFIFO_UnknownCommodity(t *testing.T) {
	r := NewLotRegistry()

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := r.DisposeFIFO("UNKNOWN", 1.0, disposalDate, 1000, 0)
	if err == nil {
		t.Error("expected error for unknown commodity")
	}
}

func TestDisposeFIFO_SkipsExhaustedLots(t *testing.T) {
	r := NewLotRegistry()

	// Lot 1: already exhausted
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 0, // Already fully disposed
		CostBasis:         40000,
		CostPerUnit:       40000,
	})
	// Lot 2: has remaining quantity
	r.AddLot(&Lot{
		Commodity:         "BTC",
		AcquisitionDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		OriginalQuantity:  1.0,
		RemainingQuantity: 1.0,
		CostBasis:         50000,
		CostPerUnit:       50000,
	})

	disposalDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	disposals, err := r.DisposeFIFO("BTC", 1.0, disposalDate, 60000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disposals) != 1 {
		t.Fatalf("expected 1 disposal, got %d", len(disposals))
	}

	// Should have disposed from lot 2, not lot 1
	if disposals[0].Lot.CostPerUnit != 50000 {
		t.Error("disposal should be from lot 2 (cost per unit 50000)")
	}
}
