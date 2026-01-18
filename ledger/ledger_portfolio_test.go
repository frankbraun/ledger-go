package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLedger_Portfolio_Basic(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/01/01 BTC 40000,00 USD
P 2024/06/01 BTC 60000,00 USD

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 42000,00 USD
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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify lots are shared
	if p.Lots != l.Lots {
		t.Error("portfolio should share lots with ledger")
	}

	// Verify prices are shared
	if p.Prices != l.Prices {
		t.Error("portfolio should share prices with ledger")
	}

	// Verify asset accounts
	if len(p.AssetAccounts) != 1 || p.AssetAccounts[0] != "Assets:Crypto:" {
		t.Errorf("expected AssetAccounts [Assets:Crypto:], got %v", p.AssetAccounts)
	}

	// Verify cash flow was extracted
	if len(p.CashFlows) != 1 {
		t.Fatalf("expected 1 cash flow, got %d", len(p.CashFlows))
	}

	cf := p.CashFlows[0]
	if cf.Amount != 42000 {
		t.Errorf("expected cash flow amount 42000, got %f", cf.Amount)
	}
	if cf.FlowType != "deposit" {
		t.Errorf("expected flow type deposit, got %s", cf.FlowType)
	}
}

func TestLedger_Portfolio_NoAssetAccounts(t *testing.T) {
	content := `commodity USD
account Assets:Bank

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
		// No AssetAccounts configured
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = l.Portfolio()
	if err == nil {
		t.Error("expected error when AssetAccounts not configured")
	}
}

func TestLedger_Portfolio_MultiplePurchases(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/06/01 BTC 60000,00 USD

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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cash flows
	if len(p.CashFlows) != 2 {
		t.Fatalf("expected 2 cash flows, got %d", len(p.CashFlows))
	}

	// First purchase: 40000
	if p.CashFlows[0].Amount != 40000 {
		t.Errorf("expected first cash flow 40000, got %f", p.CashFlows[0].Amount)
	}

	// Second purchase: 0.5 * 50000 = 25000
	if p.CashFlows[1].Amount != 25000 {
		t.Errorf("expected second cash flow 25000, got %f", p.CashFlows[1].Amount)
	}
}

func TestLedger_Portfolio_PurchaseAndSale(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/06/01 BTC 60000,00 USD

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              2 BTC @ 40000,00 USD
  Assets:Bank

2024/03/15 Sell Bitcoin
  Assets:Crypto:BTC                              -1 BTC @ 55000,00 USD
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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.CashFlows) != 2 {
		t.Fatalf("expected 2 cash flows, got %d", len(p.CashFlows))
	}

	// Purchase: deposit of 80000 (2 * 40000)
	if p.CashFlows[0].Amount != 80000 {
		t.Errorf("expected deposit 80000, got %f", p.CashFlows[0].Amount)
	}
	if p.CashFlows[0].FlowType != "deposit" {
		t.Errorf("expected deposit, got %s", p.CashFlows[0].FlowType)
	}

	// Sale: withdrawal of -55000 (proceeds)
	if p.CashFlows[1].Amount != -55000 {
		t.Errorf("expected withdrawal -55000, got %f", p.CashFlows[1].Amount)
	}
	if p.CashFlows[1].FlowType != "withdrawal" {
		t.Errorf("expected withdrawal, got %s", p.CashFlows[1].FlowType)
	}
}

func TestLedger_Portfolio_TotalCostAnnotation(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/06/01 BTC 60000,00 USD

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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.CashFlows) != 1 {
		t.Fatalf("expected 1 cash flow, got %d", len(p.CashFlows))
	}

	// Total cost annotation: 100000
	if p.CashFlows[0].Amount != 100000 {
		t.Errorf("expected cash flow 100000, got %f", p.CashFlows[0].Amount)
	}
}

func TestLedger_Portfolio_Snapshot(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/01/01 BTC 40000,00 USD
P 2024/06/01 BTC 60000,00 USD

2024/01/15 Buy Bitcoin
  Assets:Crypto:BTC                              1 BTC @ 42000,00 USD
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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get snapshot at June 1
	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Market value: 1 BTC * 60000 = 60000
	if snapshot.TotalMarketValue != 60000 {
		t.Errorf("expected market value 60000, got %f", snapshot.TotalMarketValue)
	}

	// Cost basis: 42000
	if snapshot.TotalCostBasis != 42000 {
		t.Errorf("expected cost basis 42000, got %f", snapshot.TotalCostBasis)
	}

	// Unrealized gain: 60000 - 42000 = 18000
	if snapshot.TotalUnrealizedGain != 18000 {
		t.Errorf("expected unrealized gain 18000, got %f", snapshot.TotalUnrealizedGain)
	}

	// Cash flow to date: 42000
	if snapshot.CashFlowToDate != 42000 {
		t.Errorf("expected cash flow to date 42000, got %f", snapshot.CashFlowToDate)
	}
}

func TestLedger_Portfolio_PeriodReturns(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/01/01 BTC 40000,00 USD
P 2024/12/31 BTC 50000,00 USD

2024/01/01 Buy Bitcoin
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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	returns, err := p.PeriodReturns(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start: 40000, End: 50000, No mid-period cash flows
	if returns.StartValue != 40000 {
		t.Errorf("expected start value 40000, got %f", returns.StartValue)
	}
	if returns.EndValue != 50000 {
		t.Errorf("expected end value 50000, got %f", returns.EndValue)
	}

	// Absolute return: 10000
	if returns.AbsoluteReturn != 10000 {
		t.Errorf("expected absolute return 10000, got %f", returns.AbsoluteReturn)
	}

	// Return: 25%
	if returns.Return < 0.24 || returns.Return > 0.26 {
		t.Errorf("expected return ~0.25, got %f", returns.Return)
	}
}

func TestLedger_Portfolio_MultipleCommodities(t *testing.T) {
	content := `commodity BTC
commodity ETH
commodity USD
account Assets:Crypto:BTC
account Assets:Crypto:ETH
account Assets:Bank

P 2024/06/01 BTC 60000,00 USD
P 2024/06/01 ETH 3000,00 USD

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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cash flows: 40000 + 20000 = 60000 total
	if len(p.CashFlows) != 2 {
		t.Fatalf("expected 2 cash flows, got %d", len(p.CashFlows))
	}

	totalCashFlow := p.CashFlows[0].Amount + p.CashFlows[1].Amount
	if totalCashFlow != 60000 {
		t.Errorf("expected total cash flow 60000, got %f", totalCashFlow)
	}

	// Snapshot should show both commodities
	snapshot, err := p.Snapshot(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(snapshot.Holdings) != 2 {
		t.Errorf("expected 2 holdings, got %d", len(snapshot.Holdings))
	}

	// BTC: 1 * 60000 = 60000
	// ETH: 10 * 3000 = 30000
	// Total: 90000
	if snapshot.TotalMarketValue != 90000 {
		t.Errorf("expected market value 90000, got %f", snapshot.TotalMarketValue)
	}
}

func TestLedger_Portfolio_EffectiveDate(t *testing.T) {
	content := `commodity BTC
commodity USD
account Assets:Crypto:BTC
account Assets:Bank

P 2024/06/01 BTC 60000,00 USD

2024/01/01=2024/01/15 Buy Bitcoin
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

	p, err := l.Portfolio()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cash flow should use effective date (Jan 15)
	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !p.CashFlows[0].Date.Equal(expectedDate) {
		t.Errorf("expected date %s, got %s", expectedDate.Format(DateFormat), p.CashFlows[0].Date.Format(DateFormat))
	}
}
