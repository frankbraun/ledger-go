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

## Architecture

### Package Structure

- `cmd/ledger-go/` - CLI entry point; handles flags, reads ~/.ledgerrc config
- `ledger/` - Core parsing and validation logic
- `util/file/` - File utilities (existence check, SHA-256 hashing)

### Core Data Types (ledger/ledger.go)

- `Ledger` - Full ledger: header comments, commodities, accounts, tags, entries
- `LedgerEntry` - Single transaction with date, name, accounts, metadata
- `LedgerAccount` - Account name, amount, commodity

### Parsing State Machine

The parser processes ledger files in order: header comments → commodities → accounts → tags → entries. Date format is `2006/01/02`.

### Metadata Validation

Entries can have metadata like:
```
    ; file: /path/to/invoice.pdf
    ; sha256: <hash>
    ; duplicate: true
```

In strict mode:
- Invoice files must exist and be PDFs
- SHA-256 hashes are verified
- No duplicate files (unless marked `duplicate: true`)
- All PDFs in `invoices/` directory must be referenced

## Configuration

The tool reads `~/.ledgerrc` for default flags. Paths support `~` expansion.
