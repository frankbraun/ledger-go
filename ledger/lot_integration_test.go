package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractLots_Purchase(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1,5 BTC @ 40000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have one lot
	lots := l.Lots.LotsByCommodity["BTC"]
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}

	lot := lots[0]
	if lot.Commodity != "BTC" {
		t.Errorf("expected commodity BTC, got %s", lot.Commodity)
	}
	if lot.OriginalQuantity != 1.5 {
		t.Errorf("expected quantity 1.5, got %f", lot.OriginalQuantity)
	}
	if lot.RemainingQuantity != 1.5 {
		t.Errorf("expected remaining 1.5, got %f", lot.RemainingQuantity)
	}
	if lot.CostBasis != 60000 { // 1.5 * 40000
		t.Errorf("expected cost basis 60000, got %f", lot.CostBasis)
	}
	if lot.CostPerUnit != 40000 {
		t.Errorf("expected cost per unit 40000, got %f", lot.CostPerUnit)
	}
	if lot.Account != "Assets:Crypto:BTC" {
		t.Errorf("expected account Assets:Crypto:BTC, got %s", lot.Account)
	}
}

func TestExtractLots_PurchaseTotalCost(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              2 BTC @@ 100000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lots := l.Lots.LotsByCommodity["BTC"]
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}

	lot := lots[0]
	if lot.CostBasis != 100000 {
		t.Errorf("expected cost basis 100000, got %f", lot.CostBasis)
	}
	if lot.CostPerUnit != 50000 { // 100000 / 2
		t.Errorf("expected cost per unit 50000, got %f", lot.CostPerUnit)
	}
}

func TestExtractLots_MultiplePurchases(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank

2024/02/15 Buy More Bitcoin
  Assets:Crypto:BTC                              0,5 BTC @ 50000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lots := l.Lots.LotsByCommodity["BTC"]
	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}

	// First lot (FIFO order)
	if lots[0].OriginalQuantity != 1.0 {
		t.Errorf("first lot: expected quantity 1.0, got %f", lots[0].OriginalQuantity)
	}
	if lots[0].CostPerUnit != 40000 {
		t.Errorf("first lot: expected cost per unit 40000, got %f", lots[0].CostPerUnit)
	}

	// Second lot
	if lots[1].OriginalQuantity != 0.5 {
		t.Errorf("second lot: expected quantity 0.5, got %f", lots[1].OriginalQuantity)
	}
	if lots[1].CostPerUnit != 50000 {
		t.Errorf("second lot: expected cost per unit 50000, got %f", lots[1].CostPerUnit)
	}

	// Total remaining
	total := l.Lots.RemainingQuantity("BTC")
	if total != 1.5 {
		t.Errorf("expected total remaining 1.5, got %f", total)
	}
}

func TestExtractLots_PurchaseAndSale(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              2 BTC @ 40000,00 USD
  Assets:Bank

2024/03/15 Sell Bitcoin
  Assets:Crypto:BTC                              -1 BTC @ 60000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check remaining quantity
	remaining := l.Lots.RemainingQuantity("BTC")
	if remaining != 1.0 {
		t.Errorf("expected remaining 1.0, got %f", remaining)
	}

	// Check disposal was recorded
	if len(l.Lots.Disposals) != 1 {
		t.Fatalf("expected 1 disposal, got %d", len(l.Lots.Disposals))
	}

	disposal := l.Lots.Disposals[0]
	if disposal.QuantityDisposed != 1.0 {
		t.Errorf("expected disposed quantity 1.0, got %f", disposal.QuantityDisposed)
	}
	if disposal.Proceeds != 60000 {
		t.Errorf("expected proceeds 60000, got %f", disposal.Proceeds)
	}
	// Gain: 60000 - 40000 = 20000
	if disposal.RealizedGain != 20000 {
		t.Errorf("expected realized gain 20000, got %f", disposal.RealizedGain)
	}
}

func TestExtractLots_FIFO(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin at 40k
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank

2024/02/15 Buy Bitcoin at 50k
  Assets:Crypto:BTC                              1 BTC @ 50000,00 USD
  Assets:Bank

2024/03/15 Sell Bitcoin
  Assets:Crypto:BTC                              -1 BTC @ 60000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// FIFO: first lot (40k) should be sold
	lots := l.Lots.LotsByCommodity["BTC"]

	// First lot should have 0 remaining (sold)
	if lots[0].RemainingQuantity != 0 {
		t.Errorf("first lot: expected remaining 0, got %f", lots[0].RemainingQuantity)
	}

	// Second lot should still have full quantity
	if lots[1].RemainingQuantity != 1.0 {
		t.Errorf("second lot: expected remaining 1.0, got %f", lots[1].RemainingQuantity)
	}

	// Realized gain should be based on first lot cost (40k)
	// Gain: 60000 - 40000 = 20000
	if l.Lots.Disposals[0].RealizedGain != 20000 {
		t.Errorf("expected realized gain 20000, got %f", l.Lots.Disposals[0].RealizedGain)
	}
}

func TestExtractLots_NoAssetAccounts(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// No asset accounts configured
	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no lots
	if len(l.Lots.LotsByCommodity) != 0 {
		t.Errorf("expected no lots, got %d commodities", len(l.Lots.LotsByCommodity))
	}
}

func TestExtractLots_NonMatchingAccount(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Asset accounts configured but doesn't match
	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Stocks:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no lots (Crypto doesn't match Stocks prefix)
	if len(l.Lots.LotsByCommodity) != 0 {
		t.Errorf("expected no lots, got %d commodities", len(l.Lots.LotsByCommodity))
	}
}

func TestExtractLots_InsufficientQuantityError(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank

2024/03/15 Sell Too Much Bitcoin
  Assets:Crypto:BTC                              -2 BTC @ 60000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err == nil {
		t.Error("expected error for selling more than owned")
	}
}

func TestExtractLots_EffectiveDate(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

2024/01/15=2024/01/20 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lots := l.Lots.LotsByCommodity["BTC"]
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}

	// Should use effective date (Jan 20) not accounting date (Jan 15)
	expected := "2024/01/20"
	actual := lots[0].AcquisitionDate.Format(DateFormat)
	if actual != expected {
		t.Errorf("expected acquisition date %s, got %s", expected, actual)
	}
}

func TestExtractLots_MultipleCommodities(t *testing.T) {
	content := `commodity BTC
commodity ETH
commodity USD
account Assets:Crypto:BTC
account Assets:Crypto:ETH
account Assets:Bank

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 40000,00 USD
  Assets:Bank

2024/01/16 Buy Ethereum
  Assets:Crypto:ETH                              10 ETH @ 2000,00 USD
  Assets:Bank
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Crypto:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have lots for both commodities
	if len(l.Lots.LotsByCommodity["BTC"]) != 1 {
		t.Errorf("expected 1 BTC lot, got %d", len(l.Lots.LotsByCommodity["BTC"]))
	}
	if len(l.Lots.LotsByCommodity["ETH"]) != 1 {
		t.Errorf("expected 1 ETH lot, got %d", len(l.Lots.LotsByCommodity["ETH"]))
	}

	// Verify ETH lot
	ethLot := l.Lots.LotsByCommodity["ETH"][0]
	if ethLot.OriginalQuantity != 10 {
		t.Errorf("expected ETH quantity 10, got %f", ethLot.OriginalQuantity)
	}
	if ethLot.CostBasis != 20000 { // 10 * 2000
		t.Errorf("expected ETH cost basis 20000, got %f", ethLot.CostBasis)
	}
}

func TestExtractLots_DisposalWithoutPriorLots(t *testing.T) {
	// Regression test: disposing a commodity with no prior lots should not error.
	// This handles historical entries from before lot tracking was enabled.
	// Example: selling ZEC that was acquired before the ledger started tracking lots.
	content := `commodity EUR
commodity ZEC
account Assets:Cash
account Assets:Savings:Cryptocurrencies:Zcash

2017/05/04 Exchange
  Assets:Cash
  Assets:Savings:Cryptocurrencies:Zcash  -1,21932430 ZEC @ 93,00 EUR
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		AssetAccounts:   []string{"Assets:Savings:"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no lots (disposal was skipped since no prior lots existed)
	if len(l.Lots.LotsByCommodity["ZEC"]) != 0 {
		t.Errorf("expected 0 ZEC lots, got %d", len(l.Lots.LotsByCommodity["ZEC"]))
	}

	// Should have no disposals recorded
	if len(l.Lots.Disposals) != 0 {
		t.Errorf("expected 0 disposals, got %d", len(l.Lots.Disposals))
	}
}
