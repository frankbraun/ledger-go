# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ledger-go is a Go tool that parses a subset of the Ledger CLI accounting file format. Its main purpose is enforcing that every expense/income entry is linked to an invoice file via metadata annotations, with SHA-256 hash verification for file integrity.

## Build Commands

```bash
# Build and install
make                    # runs: go install -v ./...
go install ./...        # alternative direct command

# Run the tool
ledger-go -file <ledger-file> [options]
```

## Key Command-Line Flags

- `-file` - Path to ledger journal file
- `-strict` - Enable strict validation (checks accounts/commodities are declared, verifies hashes)
- `-add-missing-hashes` - Automatically add SHA-256 hashes for invoice files
- `-no-metadata` - Config file listing accounts that don't require metadata (default: no-metadata.conf)
- `-price-db` - Read price DB from FILE
- `-no-pager` - Disables the pager on TTY output
- `-disable-metadata` - Skip all metadata validation
- `-portfolio` - Show portfolio snapshot instead of normal ledger output
- `-asset-accounts` - Comma-separated asset account prefixes for portfolio tracking (e.g., "Assets:Crypto:,Assets:Stocks:")

## Architecture

### Package Structure

- `cmd/ledger-go/` - CLI entry point; handles flags, reads ~/.ledgerrc config
- `ledger/` - Core parsing and validation logic
- `util/file/` - File utilities (existence check, SHA-256 hashing)

### Core Data Types (ledger/ledger.go)

- `Config` - Configuration for `New()`: Filename, Strict, AddMissingHashes, DisableMetadata, NoMetadataFilename, AssetAccounts
- `Ledger` - Full ledger: header comments, commodities, accounts, tags, prices, entries, lots
- `LedgerEntry` - Single transaction with date, name, accounts, metadata, effective date (supports `2006/01/02=2006/01/02` format)
- `LedgerAccount` - Account name, amount, commodity, price annotations (`@`/`@@`), elided flag (amount was omitted and inferred)

### Creating a Ledger

```go
l, err := ledger.New(ledger.Config{
    Filename:           "path/to/file.ledger",
    Strict:             true,  // enable strict validation
    AddMissingHashes:   false, // auto-add SHA256 hashes
    DisableMetadata:    false, // skip metadata validation
    NoMetadataFilename: "",    // accounts exempt from metadata
    AssetAccounts:      []string{"Assets:Crypto:", "Assets:Stocks:"}, // enable lot tracking
})
```

### Parsing State Machine

The parser processes ledger files in order: header comments → commodities → accounts → tags → prices → entries. Date format is `2006/01/02`. Effective dates use `2006/01/02=2006/01/02` format.

### Price Annotations

Amounts can include price annotations:
- `@ price` - Per-unit price (e.g., `10 AAPL @ $150.00`)
- `@@ price` - Total cost (e.g., `10 AAPL @@ $1500.00`)

### Price Directives

Historical prices can be declared with price directives:
```
P 2024/01/15 BTC 42000,00 USD
P 2024/02/01 BTC 45000,00 USD
```

Format: `P DATE COMMODITY PRICE PRICECOMMODITY`

Price directives are parsed and stored in `Ledger.Prices` (a `PriceHistory`). In strict mode, commodities must be declared.

### Metadata Validation

Entries can have metadata like:
```
    ; file: /path/to/invoice.pdf
    ; sha256: <hash>
    ; fileTwo: /path/to/second-invoice.pdf
    ; sha256Two: <hash>
    ; duplicate: true
```

In strict mode:
- Invoice files must exist and be PDFs
- SHA-256 hashes are verified
- No duplicate files (unless marked `duplicate: true`)
- All PDFs in `invoices/` directory must be referenced

## Portfolio Tracking

The ledger package includes data structures for tracking asset performance over time. All monetary values are in USD.

### Lot Tracking (ledger/lot.go)

When `AssetAccounts` is configured, the parser automatically creates lots from purchases and records disposals:

- `Lot` - Single purchase with cost basis (commodity, acquisition date, quantity, cost basis, cost per unit)
- `LotDisposal` - Sale record with realized gain calculation
- `LotRegistry` - Manages lots with FIFO ordering

```go
// Lots are created automatically from entries with price annotations
// on accounts matching AssetAccounts prefixes
// Example entry that creates a lot:
//   Assets:Crypto:BTC    1 BTC @ 40000,00 USD

// Disposals use FIFO (oldest lots sold first)
// Example entry that creates a disposal:
//   Assets:Crypto:BTC    -0.5 BTC @ 50000,00 USD
```

### Price History (ledger/price.go)

- `PricePoint` - Price at a specific date
- `PriceHistory` - Historical prices by commodity with lookup methods

```go
// Add prices
prices := ledger.NewPriceHistory()
prices.AddPrice("BTC", date, 42000)

// Lookup (returns most recent price on or before date)
price, err := prices.GetPrice("BTC", date)
```

### Portfolio State (ledger/portfolio.go)

- `CommodityHolding` - Aggregated position in a single commodity
- `PortfolioSnapshot` - Complete portfolio state at a point in time
- `CashFlow` - Cash movement record
- `Portfolio` - Full portfolio with lots, prices, snapshots

```go
// Create portfolio from a parsed ledger (recommended)
l, err := ledger.New(ledger.Config{
    Filename:      "file.ledger",
    AssetAccounts: []string{"Assets:Crypto:"},
})
p, err := l.Portfolio()

// Or create manually
p := ledger.NewPortfolio()
// ... add lots and prices ...

// Get snapshot at a date
snapshot, err := p.Snapshot(date)
// snapshot.TotalMarketValue, snapshot.Holdings, snapshot.Allocations, etc.
```

### Performance Metrics (ledger/performance.go)

- `PeriodReturn` - Performance over a time period (uses Modified Dietz method)
- `AssetPerformance` - Per-commodity performance
- `PerformanceReport` - Aggregated metrics

```go
returns, err := portfolio.PeriodReturns(startDate, endDate)
// returns.Return (MWR), returns.AnnualizedReturn (CAGR), returns.AbsoluteReturn
```

### Time Series (ledger/timeseries.go)

- `TimeSeries` - Sequence of dated values with Add/Get/Min/Max methods
- `PortfolioTimeSeries` - Multiple metrics over time (value, cost basis, gains)

```go
pts, err := portfolio.BuildTimeSeries(startDate, endDate, ledger.Monthly)
// pts.Value, pts.CostBasis, pts.UnrealizedGain, pts.Holdings["BTC"], etc.
```

Intervals: `Daily`, `Weekly`, `Monthly`

## Configuration

The tool reads `~/.ledgerrc` for default flags. Paths support `~` expansion.

## Maintaining This Document

When making changes to the codebase, update this file to reflect:
- New or modified command-line flags (check `cmd/ledger-go/main.go`)
- Changes to core data types in `ledger/ledger.go`
- New ledger syntax support (date formats, price annotations, price directives, etc.)
- New metadata fields or validation rules
- Changes to package structure
- Updates to portfolio tracking features (lot.go, price.go, portfolio.go, performance.go, timeseries.go)
