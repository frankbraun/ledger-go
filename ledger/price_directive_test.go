package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePriceDirective_Basic(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/01/15 BTC 42000,00 USD

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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have one price
	prices := l.Prices.Prices["BTC"]
	if len(prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(prices))
	}

	price := prices[0]
	if price.Commodity != "BTC" {
		t.Errorf("expected commodity BTC, got %s", price.Commodity)
	}
	if price.Price != 42000 {
		t.Errorf("expected price 42000, got %f", price.Price)
	}
	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !price.Date.Equal(expectedDate) {
		t.Errorf("expected date %s, got %s", expectedDate.Format(DateFormat), price.Date.Format(DateFormat))
	}
}

func TestParsePriceDirective_MultiplePrices(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/01/01 BTC 40000,00 USD
P 2024/02/01 BTC 45000,00 USD
P 2024/03/01 BTC 50000,00 USD

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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices := l.Prices.Prices["BTC"]
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	// Verify chronological order
	if prices[0].Price != 40000 {
		t.Errorf("first price should be 40000, got %f", prices[0].Price)
	}
	if prices[1].Price != 45000 {
		t.Errorf("second price should be 45000, got %f", prices[1].Price)
	}
	if prices[2].Price != 50000 {
		t.Errorf("third price should be 50000, got %f", prices[2].Price)
	}
}

func TestParsePriceDirective_MultipleCommodities(t *testing.T) {
	content := `commodity BTC
commodity ETH
commodity USD

P 2024/01/15 BTC 42000,00 USD
P 2024/01/15 ETH 2500,00 USD

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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(l.Prices.Prices["BTC"]) != 1 {
		t.Errorf("expected 1 BTC price, got %d", len(l.Prices.Prices["BTC"]))
	}
	if len(l.Prices.Prices["ETH"]) != 1 {
		t.Errorf("expected 1 ETH price, got %d", len(l.Prices.Prices["ETH"]))
	}

	btcPrice, _ := l.Prices.GetPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if btcPrice != 42000 {
		t.Errorf("expected BTC price 42000, got %f", btcPrice)
	}

	ethPrice, _ := l.Prices.GetPrice("ETH", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if ethPrice != 2500 {
		t.Errorf("expected ETH price 2500, got %f", ethPrice)
	}
}

func TestParsePriceDirective_DecimalPoint(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/01/15 BTC 42000.50 USD

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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	price, _ := l.Prices.GetPrice("BTC", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if price != 42000.50 {
		t.Errorf("expected price 42000.50, got %f", price)
	}
}

func TestParsePriceDirective_StrictModeUnknownCommodity(t *testing.T) {
	content := `commodity USD

P 2024/01/15 BTC 42000,00 USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		Strict:          true,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for unknown commodity in strict mode")
	}
}

func TestParsePriceDirective_StrictModeUnknownPriceCommodity(t *testing.T) {
	content := `commodity BTC

P 2024/01/15 BTC 42000,00 USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 BTC
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		Strict:          true,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for unknown price commodity in strict mode")
	}
}

func TestParsePriceDirective_InvalidFormat(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/01/15 BTC

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for invalid price directive format")
	}
}

func TestParsePriceDirective_InvalidDate(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024-01-15 BTC 42000,00 USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for invalid date format")
	}
}

func TestParsePriceDirective_InvalidPrice(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/01/15 BTC abc USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for invalid price")
	}
}

func TestParsePriceDirective_IntegrationWithPortfolio(t *testing.T) {
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

	// Verify lot was created
	if l.Lots.RemainingQuantity("BTC") != 1.0 {
		t.Errorf("expected 1 BTC remaining, got %f", l.Lots.RemainingQuantity("BTC"))
	}

	// Verify price lookup works
	price, err := l.Prices.GetPrice("BTC", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 60000 {
		t.Errorf("expected price 60000, got %f", price)
	}

	// Can use prices to calculate portfolio value
	price, _ = l.Prices.GetPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if price != 40000 { // Uses Jan 1 price (most recent before Mar 1)
		t.Errorf("expected interpolated price 40000, got %f", price)
	}
}

func TestParsePriceDirective_OutOfOrder(t *testing.T) {
	content := `commodity BTC
commodity USD

P 2024/03/01 BTC 50000,00 USD
P 2024/01/01 BTC 40000,00 USD
P 2024/02/01 BTC 45000,00 USD

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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Prices should be sorted chronologically regardless of input order
	prices := l.Prices.Prices["BTC"]
	if prices[0].Price != 40000 {
		t.Errorf("first price should be 40000, got %f", prices[0].Price)
	}
	if prices[1].Price != 45000 {
		t.Errorf("second price should be 45000, got %f", prices[1].Price)
	}
	if prices[2].Price != 50000 {
		t.Errorf("third price should be 50000, got %f", prices[2].Price)
	}
}

func TestParsePriceDirective_NoPrices(t *testing.T) {
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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(l.Prices.Prices) != 0 {
		t.Errorf("expected no prices, got %d commodities", len(l.Prices.Prices))
	}
}

func TestPriceDB_Basic(t *testing.T) {
	ledgerContent := `commodity BTC
commodity USD
account Assets:Bank

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	priceDBContent := `P 2024/01/15 BTC 42000,00 USD
P 2024/02/01 BTC 45000,00 USD
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	priceDBFile := filepath.Join(tmpDir, "prices.db")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priceDBFile, []byte(priceDBContent), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: priceDBFile,
		DisableMetadata: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices := l.Prices.Prices["BTC"]
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}

	price, _ := l.Prices.GetPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if price != 45000 {
		t.Errorf("expected price 45000, got %f", price)
	}
}

func TestPriceDB_WithComments(t *testing.T) {
	ledgerContent := `commodity BTC
commodity USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	priceDBContent := `; This is a price database file
P 2024/01/15 BTC 42000,00 USD

; Another comment
P 2024/02/01 BTC 45000,00 USD
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	priceDBFile := filepath.Join(tmpDir, "prices.db")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priceDBFile, []byte(priceDBContent), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: priceDBFile,
		DisableMetadata: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices := l.Prices.Prices["BTC"]
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
}

func TestPriceDB_MergesWithLedgerPrices(t *testing.T) {
	ledgerContent := `commodity BTC
commodity USD

P 2024/01/01 BTC 40000,00 USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	priceDBContent := `P 2024/02/01 BTC 45000,00 USD
P 2024/03/01 BTC 50000,00 USD
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	priceDBFile := filepath.Join(tmpDir, "prices.db")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priceDBFile, []byte(priceDBContent), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: priceDBFile,
		DisableMetadata: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 prices total (1 from ledger + 2 from price-db)
	prices := l.Prices.Prices["BTC"]
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	// Verify all prices are accessible
	price, _ := l.Prices.GetPrice("BTC", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if price != 40000 {
		t.Errorf("expected price 40000, got %f", price)
	}
	price, _ = l.Prices.GetPrice("BTC", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if price != 45000 {
		t.Errorf("expected price 45000, got %f", price)
	}
	price, _ = l.Prices.GetPrice("BTC", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if price != 50000 {
		t.Errorf("expected price 50000, got %f", price)
	}
}

func TestPriceDB_StrictModeValidation(t *testing.T) {
	ledgerContent := `commodity USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	priceDBContent := `P 2024/01/15 BTC 42000,00 USD
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	priceDBFile := filepath.Join(tmpDir, "prices.db")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priceDBFile, []byte(priceDBContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: priceDBFile,
		Strict:          true,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for unknown commodity in strict mode")
	}
}

func TestPriceDB_InvalidLine(t *testing.T) {
	ledgerContent := `commodity BTC
commodity USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	priceDBContent := `P 2024/01/15 BTC 42000,00 USD
commodity ETH
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	priceDBFile := filepath.Join(tmpDir, "prices.db")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priceDBFile, []byte(priceDBContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: priceDBFile,
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for invalid line in price-db")
	}
}

func TestPriceDB_FileNotFound(t *testing.T) {
	ledgerContent := `commodity USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: filepath.Join(tmpDir, "nonexistent.db"),
		DisableMetadata: true,
	})
	if err == nil {
		t.Error("expected error for nonexistent price-db file")
	}
}

func TestPriceDB_EmptyFilename(t *testing.T) {
	ledgerContent := `commodity USD

2024/01/01 Opening
  Assets:Bank                                    1000,00 USD
  Equity:Opening
`
	tmpDir := t.TempDir()
	ledgerFile := filepath.Join(tmpDir, "test.ledger")
	if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Empty PriceDBFilename should be ignored (no error)
	l, err := New(Config{
		Filename:        ledgerFile,
		PriceDBFilename: "",
		DisableMetadata: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(l.Prices.Prices) != 0 {
		t.Errorf("expected no prices, got %d commodities", len(l.Prices.Prices))
	}
}
