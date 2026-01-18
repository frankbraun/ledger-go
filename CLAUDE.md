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

## Architecture

### Package Structure

- `cmd/ledger-go/` - CLI entry point; handles flags, reads ~/.ledgerrc config
- `ledger/` - Core parsing and validation logic
- `util/file/` - File utilities (existence check, SHA-256 hashing)

### Core Data Types (ledger/ledger.go)

- `Config` - Configuration for `New()`: Filename, Strict, AddMissingHashes, DisableMetadata, NoMetadataFilename
- `Ledger` - Full ledger: header comments, commodities, accounts, tags, entries
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
})
```

### Parsing State Machine

The parser processes ledger files in order: header comments → commodities → accounts → tags → entries. Date format is `2006/01/02`. Effective dates use `2006/01/02=2006/01/02` format.

### Price Annotations

Amounts can include price annotations:
- `@ price` - Per-unit price (e.g., `10 AAPL @ $150.00`)
- `@@ price` - Total cost (e.g., `10 AAPL @@ $1500.00`)

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

## Configuration

The tool reads `~/.ledgerrc` for default flags. Paths support `~` expansion.

## Maintaining This Document

When making changes to the codebase, update this file to reflect:
- New or modified command-line flags (check `cmd/ledger-go/main.go`)
- Changes to core data types in `ledger/ledger.go`
- New ledger syntax support (date formats, price annotations, etc.)
- New metadata fields or validation rules
- Changes to package structure
