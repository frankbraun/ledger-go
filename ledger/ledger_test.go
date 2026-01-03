package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frankbraun/ledger-go/util/file"
)

func TestParseAccount(t *testing.T) {
	commodities := map[string]bool{"EUR": true, "USD": true}
	accounts := map[string]bool{"Assets:Bank": true, "Expenses:Food": true}

	tests := []struct {
		name        string
		line        string
		ln          int
		strict      bool
		wantName    string
		wantAmount  float64
		wantComm    string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid account with amount",
			line:       "Assets:Bank  100,00 EUR",
			ln:         1,
			strict:     false,
			wantName:   "Assets:Bank",
			wantAmount: 100.0,
			wantComm:   "EUR",
			wantErr:    false,
		},
		{
			name:       "valid account with decimal point",
			line:       "Expenses:Food  25.50 USD",
			ln:         1,
			strict:     false,
			wantName:   "Expenses:Food",
			wantAmount: 25.50,
			wantComm:   "USD",
			wantErr:    false,
		},
		{
			name:       "valid account without amount",
			line:       "Assets:Bank",
			ln:         1,
			strict:     false,
			wantName:   "Assets:Bank",
			wantAmount: 0,
			wantComm:   "",
			wantErr:    false,
		},
		{
			name:       "negative amount",
			line:       "Expenses:Food  -50,00 EUR",
			ln:         1,
			strict:     false,
			wantName:   "Expenses:Food",
			wantAmount: -50.0,
			wantComm:   "EUR",
			wantErr:    false,
		},
		{
			name:        "strict mode unknown account",
			line:        "Unknown:Account  10,00 EUR",
			ln:          5,
			strict:      true,
			wantErr:     true,
			errContains: "account unknown",
		},
		{
			name:        "strict mode unknown commodity",
			line:        "Assets:Bank  10,00 GBP",
			ln:          5,
			strict:      true,
			wantErr:     true,
			errContains: "commodity unknown",
		},
		{
			name:        "wrong number of elements",
			line:        "Assets:Bank 100,00",
			ln:          3,
			strict:      false,
			wantErr:     true,
			errContains: "doesn't have 3 or 1 element",
		},
		{
			name:        "invalid amount format",
			line:        "Assets:Bank  notanumber EUR",
			ln:          2,
			strict:      false,
			wantErr:     true,
			errContains: "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAccount(tt.line, tt.ln, tt.strict, commodities, accounts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAccount() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("parseAccount() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseAccount() unexpected error: %v", err)
				return
			}

			if got.Name != tt.wantName {
				t.Errorf("parseAccount() Name = %v, want %v", got.Name, tt.wantName)
			}
			if got.Amount != tt.wantAmount {
				t.Errorf("parseAccount() Amount = %v, want %v", got.Amount, tt.wantAmount)
			}
			if got.Commodity != tt.wantComm {
				t.Errorf("parseAccount() Commodity = %v, want %v", got.Commodity, tt.wantComm)
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		ln          int
		existing    map[string]string
		wantTag     string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid file metadata",
			line:      "; file: /path/to/invoice.pdf",
			ln:        1,
			existing:  nil,
			wantTag:   "file",
			wantValue: "/path/to/invoice.pdf",
			wantErr:   false,
		},
		{
			name:      "valid sha256 metadata",
			line:      "; sha256: abc123def456",
			ln:        1,
			existing:  nil,
			wantTag:   "sha256",
			wantValue: "abc123def456",
			wantErr:   false,
		},
		{
			name:      "valid duplicate flag",
			line:      "; duplicate: true",
			ln:        1,
			existing:  nil,
			wantTag:   "duplicate",
			wantValue: "true",
			wantErr:   false,
		},
		{
			name:        "duplicate tag error",
			line:        "; file: /another/path.pdf",
			ln:          5,
			existing:    map[string]string{"file": "/first/path.pdf"},
			wantErr:     true,
			errContains: "metadata tag already exists",
		},
		{
			name:        "malformed no colon",
			line:        "; this has no colon",
			ln:          3,
			wantErr:     true,
			errContains: "not metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &LedgerEntry{}
			if tt.existing != nil {
				e.Metadata = tt.existing
			} else {
				e.Metadata = make(map[string]string)
			}

			err := e.parseMetadata(tt.line, tt.ln)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseMetadata() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("parseMetadata() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseMetadata() unexpected error: %v", err)
				return
			}

			if got := e.Metadata[tt.wantTag]; got != tt.wantValue {
				t.Errorf("parseMetadata() Metadata[%q] = %v, want %v", tt.wantTag, got, tt.wantValue)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("minimal valid ledger", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `; Header comment

commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Grocery store
  Expenses:Food  50,00 EUR
  Assets:Bank

2024/01/15 Restaurant
  Expenses:Food  25,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.HeaderComments) != 1 {
			t.Errorf("HeaderComments len = %d, want 1", len(l.HeaderComments))
		}
		if !l.Commodities["EUR"] {
			t.Error("Commodities should contain EUR")
		}
		if !l.Accounts["Assets:Bank"] || !l.Accounts["Expenses:Food"] {
			t.Error("Accounts should contain Assets:Bank and Expenses:Food")
		}
		if len(l.Entries) != 2 {
			t.Errorf("Entries len = %d, want 2", len(l.Entries))
		}
	})

	t.Run("ledger with effective dates", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01=2024/01/15 Delayed payment
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if entry.Date.Format(DateFormat) != "2024/01/01" {
			t.Errorf("Date = %s, want 2024/01/01", entry.Date.Format(DateFormat))
		}
		if entry.EffectiveDate.Format(DateFormat) != "2024/01/15" {
			t.Errorf("EffectiveDate = %s, want 2024/01/15", entry.EffectiveDate.Format(DateFormat))
		}
	})

	t.Run("ledger with tags", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

tag invoice
tag receipt

2024/01/01 Grocery store
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if !l.Tags["invoice"] || !l.Tags["receipt"] {
			t.Error("Tags should contain invoice and receipt")
		}
	})

	t.Run("date order validation", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/15 Second entry
  Expenses:Food  25,00 EUR
  Assets:Bank

2024/01/01 First entry (out of order)
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for out-of-order dates, got nil")
		}
		if !contains(err.Error(), "before") {
			t.Errorf("error should mention date ordering, got: %v", err)
		}
	})

	t.Run("strict mode unknown account", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, true, false, "")
		if err == nil {
			t.Fatal("New() expected error for unknown account in strict mode, got nil")
		}
		if !contains(err.Error(), "account unknown") {
			t.Errorf("error should mention unknown account, got: %v", err)
		}
	})

	t.Run("strict mode unknown commodity", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, true, false, "")
		if err == nil {
			t.Fatal("New() expected error for unknown commodity in strict mode, got nil")
		}
		if !contains(err.Error(), "commodity unknown") {
			t.Errorf("error should mention unknown commodity, got: %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := New("/nonexistent/path/file.ledger", false, false, "")
		if err == nil {
			t.Fatal("New() expected error for nonexistent file, got nil")
		}
	})

	t.Run("noMetadata config file", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")
		noMetadataFile := filepath.Join(dir, "no-metadata.conf")

		ledgerContent := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		noMetadataContent := "Expenses:Food\n"

		if err := os.WriteFile(ledgerFile, []byte(ledgerContent), 0644); err != nil {
			t.Fatalf("failed to write ledger file: %v", err)
		}
		if err := os.WriteFile(noMetadataFile, []byte(noMetadataContent), 0644); err != nil {
			t.Fatalf("failed to write no-metadata file: %v", err)
		}

		l, err := New(ledgerFile, false, false, noMetadataFile)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if !l.NoMetadata["Expenses:Food"] {
			t.Error("NoMetadata should contain Expenses:Food")
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

01-01-2024 Invalid date format
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for invalid date format, got nil")
		}
	})

	t.Run("invalid effective date format", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01=invalid Delayed payment
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for invalid effective date format, got nil")
		}
	})

	t.Run("invalid accounting date with effective date", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

invalid=2024/01/15 Delayed payment
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for invalid accounting date, got nil")
		}
	})

	t.Run("account line not starting with spaces", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for account line not starting with spaces, got nil")
		}
		if !contains(err.Error(), "not an account line") {
			t.Errorf("error should mention not an account line, got: %v", err)
		}
	})

	t.Run("account line after metadata", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  ; note: this is metadata
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for account after metadata, got nil")
		}
		if !contains(err.Error(), "already parsing metadata") {
			t.Errorf("error should mention already parsing metadata, got: %v", err)
		}
	})

	t.Run("effective date ordering", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01=2024/01/20 First entry
  Expenses:Food  50,00 EUR
  Assets:Bank

2024/01/05=2024/01/10 Second entry (effective date before first)
  Expenses:Food  25,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := New(ledgerFile, false, false, "")
		if err == nil {
			t.Fatal("New() expected error for effective date ordering, got nil")
		}
		if !contains(err.Error(), "before") {
			t.Errorf("error should mention date ordering, got: %v", err)
		}
	})

	t.Run("entry without name", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if l.Entries[0].Name != "" {
			t.Errorf("Entry name should be empty, got: %s", l.Entries[0].Name)
		}
	})

	t.Run("entry with metadata", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")
		invoiceFile := filepath.Join(dir, "invoice.pdf")

		if err := os.WriteFile(invoiceFile, []byte("pdf content"), 0644); err != nil {
			t.Fatalf("failed to write invoice file: %v", err)
		}

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
  ; file: ` + invoiceFile + `
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if l.Entries[0].Metadata["file"] != invoiceFile {
			t.Errorf("Metadata file should be %s, got: %s", invoiceFile, l.Entries[0].Metadata["file"])
		}
	})

	t.Run("noMetadata file not found errors", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Pass a non-existent noMetadata file - should error
		_, err := New(ledgerFile, false, false, "/nonexistent/no-metadata.conf")
		if err == nil {
			t.Fatal("New() expected error for nonexistent noMetadata file, got nil")
		}
	})

	t.Run("empty noMetadata filename is ok", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food

2024/01/01 Test entry
  Expenses:Food  50,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Pass empty noMetadata filename - should be ok
		_, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
	})

	t.Run("entry with three accounts", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Assets:Cash
account Expenses:Food

2024/01/01 Split payment at grocery store
  Expenses:Food  100,00 EUR
  Assets:Bank  -70,00 EUR
  Assets:Cash  -30,00 EUR
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if len(entry.Accounts) != 3 {
			t.Errorf("Accounts len = %d, want 3", len(entry.Accounts))
		}
		if entry.Accounts[0].Name != "Expenses:Food" || entry.Accounts[0].Amount != 100.0 {
			t.Errorf("First account = %+v, want Expenses:Food 100.0", entry.Accounts[0])
		}
		if entry.Accounts[1].Name != "Assets:Bank" || entry.Accounts[1].Amount != -70.0 {
			t.Errorf("Second account = %+v, want Assets:Bank -70.0", entry.Accounts[1])
		}
		if entry.Accounts[2].Name != "Assets:Cash" || entry.Accounts[2].Amount != -30.0 {
			t.Errorf("Third account = %+v, want Assets:Cash -30.0", entry.Accounts[2])
		}
	})

	t.Run("entry with four accounts", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Assets:Cash
account Expenses:Food
account Expenses:Tips

2024/01/01 Restaurant with tip split
  Expenses:Food  80,00 EUR
  Expenses:Tips  15,00 EUR
  Assets:Bank  -50,00 EUR
  Assets:Cash  -45,00 EUR
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if len(entry.Accounts) != 4 {
			t.Errorf("Accounts len = %d, want 4", len(entry.Accounts))
		}
	})

	t.Run("entry with one account and elided amount", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank
account Expenses:Food
account Expenses:Tips

2024/01/01 Restaurant with tip
  Expenses:Food  80,00 EUR
  Expenses:Tips  15,00 EUR
  Assets:Bank
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if len(entry.Accounts) != 3 {
			t.Errorf("Accounts len = %d, want 3", len(entry.Accounts))
		}
		// Last account should have no amount (elided)
		if entry.Accounts[2].Amount != 0 || entry.Accounts[2].Commodity != "" {
			t.Errorf("Third account should have elided amount, got %+v", entry.Accounts[2])
		}
	})

	t.Run("entry with single account", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")

		content := `commodity EUR

account Assets:Bank

2024/01/01 Opening balance
  Assets:Bank  1000,00 EUR
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if len(entry.Accounts) != 1 {
			t.Errorf("Accounts len = %d, want 1", len(entry.Accounts))
		}
	})

	t.Run("multi-account entry with metadata", func(t *testing.T) {
		dir := t.TempDir()
		ledgerFile := filepath.Join(dir, "test.ledger")
		invoiceFile := filepath.Join(dir, "invoice.pdf")

		if err := os.WriteFile(invoiceFile, []byte("pdf content"), 0644); err != nil {
			t.Fatalf("failed to write invoice file: %v", err)
		}

		content := `commodity EUR

account Assets:Bank
account Assets:Cash
account Expenses:Food

2024/01/01 Split payment at grocery store
  Expenses:Food  100,00 EUR
  Assets:Bank  -70,00 EUR
  Assets:Cash  -30,00 EUR
  ; file: ` + invoiceFile + `
`
		if err := os.WriteFile(ledgerFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l, err := New(ledgerFile, false, false, "")
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if len(l.Entries) != 1 {
			t.Fatalf("Entries len = %d, want 1", len(l.Entries))
		}

		entry := l.Entries[0]
		if len(entry.Accounts) != 3 {
			t.Errorf("Accounts len = %d, want 3", len(entry.Accounts))
		}
		if entry.Metadata["file"] != invoiceFile {
			t.Errorf("Metadata file = %s, want %s", entry.Metadata["file"], invoiceFile)
		}
	})
}

func TestProcFilename(t *testing.T) {
	t.Run("file exists and is PDF", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if err := procFilename(file1); err != nil {
			t.Errorf("procFilename() error = %v, want nil", err)
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		err := procFilename("/nonexistent/path/invoice.pdf")
		if err == nil {
			t.Fatal("procFilename() expected error for nonexistent file, got nil")
		}
		if !contains(err.Error(), "doesn't exist") {
			t.Errorf("error should mention file doesn't exist, got: %v", err)
		}
	})

	t.Run("file exists but not PDF", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "document.txt")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		err := procFilename(file1)
		if err == nil {
			t.Fatal("procFilename() expected error for non-PDF file, got nil")
		}
		if !contains(err.Error(), "not a PDF") {
			t.Errorf("error should mention not a PDF, got: %v", err)
		}
	})
}

func TestProcHash(t *testing.T) {
	t.Run("hash exists strict mode matches", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		content := []byte("test content for hashing")
		if err := os.WriteFile(file1, content, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Calculate the actual hash using the same function
		actualHash, err := file.SHA256Sum(file1)
		if err != nil {
			t.Fatalf("failed to calculate hash: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"sha256": actualHash,
			},
		}

		err = e.procHash("sha256", file1, true, false, 1)
		if err != nil {
			t.Errorf("procHash() error = %v, want nil", err)
		}
	})

	t.Run("hash exists strict mode mismatch", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"sha256": "wronghash",
			},
		}

		err := e.procHash("sha256", file1, true, false, 5)
		if err == nil {
			t.Fatal("procHash() expected error for hash mismatch, got nil")
		}
		if !contains(err.Error(), "hash mismatch") {
			t.Errorf("error should mention hash mismatch, got: %v", err)
		}
		if !contains(err.Error(), "line 5") {
			t.Errorf("error should mention line number, got: %v", err)
		}
	})

	t.Run("hash exists non-strict mode skips verification", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"sha256": "wronghash", // wrong hash, but non-strict so should pass
			},
		}

		err := e.procHash("sha256", file1, false, false, 1)
		if err != nil {
			t.Errorf("procHash() error = %v, want nil", err)
		}
	})

	t.Run("hash missing with addMissingHashes", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{},
		}

		err := e.procHash("sha256", file1, false, true, 1)
		if err != nil {
			t.Errorf("procHash() error = %v, want nil", err)
		}
		if e.Metadata["sha256"] == "" {
			t.Error("hash should have been added to metadata")
		}
	})

	t.Run("hash missing strict mode no addMissingHashes", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{},
		}

		err := e.procHash("sha256", file1, true, false, 1)
		if err == nil {
			t.Fatal("procHash() expected error for missing hash in strict mode, got nil")
		}
		if !contains(err.Error(), "no hash for file") {
			t.Errorf("error should mention no hash, got: %v", err)
		}
	})

	t.Run("hash missing non-strict no addMissingHashes passes", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{},
		}

		err := e.procHash("sha256", file1, false, false, 1)
		if err != nil {
			t.Errorf("procHash() error = %v, want nil", err)
		}
	})

	t.Run("error calculating hash for missing file", func(t *testing.T) {
		e := &LedgerEntry{
			Metadata: map[string]string{
				"sha256": "somehash",
			},
		}

		err := e.procHash("sha256", "/nonexistent/file.pdf", true, false, 1)
		if err == nil {
			t.Fatal("procHash() expected error for missing file, got nil")
		}
	})

	t.Run("error calculating hash when adding missing hash", func(t *testing.T) {
		e := &LedgerEntry{
			Metadata: map[string]string{},
		}

		err := e.procHash("sha256", "/nonexistent/file.pdf", false, true, 1)
		if err == nil {
			t.Fatal("procHash() expected error for missing file, got nil")
		}
	})

	t.Run("sha256Two metadata key", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{},
		}

		err := e.procHash("sha256Two", file1, false, true, 1)
		if err != nil {
			t.Errorf("procHash() error = %v, want nil", err)
		}
		if e.Metadata["sha256Two"] == "" {
			t.Error("sha256Two should have been added to metadata")
		}
	})
}

func TestProcMetadata(t *testing.T) {
	t.Run("nil metadata passes", func(t *testing.T) {
		e := &LedgerEntry{
			Metadata: nil,
			Accounts: []LedgerAccount{
				{Name: "Assets:Bank"},
				{Name: "Assets:Cash"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("valid file metadata", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file": file1,
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("invalid file metadata - file does not exist", func(t *testing.T) {
		e := &LedgerEntry{
			Metadata: map[string]string{
				"file": "/nonexistent/invoice.pdf",
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for nonexistent file, got nil")
		}
		if !contains(err.Error(), "doesn't exist") {
			t.Errorf("error should mention file doesn't exist, got: %v", err)
		}
	})

	t.Run("invalid file metadata - not a PDF", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "document.txt")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file": file1,
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for non-PDF file, got nil")
		}
		if !contains(err.Error(), "not a PDF") {
			t.Errorf("error should mention not a PDF, got: %v", err)
		}
	})

	t.Run("fileTwo without file errors", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"fileTwo": file1,
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 5, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for fileTwo without file, got nil")
		}
		if !contains(err.Error(), "'fileTwo' defined but not 'file'") {
			t.Errorf("error should mention fileTwo without file, got: %v", err)
		}
		if !contains(err.Error(), "line 5") {
			t.Errorf("error should mention line number, got: %v", err)
		}
	})

	t.Run("valid file and fileTwo metadata", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file":    file1,
				"fileTwo": file2,
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("invalid fileTwo - file does not exist", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file":    file1,
				"fileTwo": "/nonexistent/invoice.pdf",
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for nonexistent fileTwo, got nil")
		}
		if !contains(err.Error(), "doesn't exist") {
			t.Errorf("error should mention file doesn't exist, got: %v", err)
		}
	})

	t.Run("strict mode hash error propagates", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file": file1,
				// no sha256 - strict mode should error
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(true, false, 1, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for missing hash in strict mode, got nil")
		}
		if !contains(err.Error(), "no hash for file") {
			t.Errorf("error should mention no hash, got: %v", err)
		}
	})

	t.Run("addMissingHashes adds hash", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file": file1,
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(false, true, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
		if e.Metadata["sha256"] == "" {
			t.Error("sha256 should have been added to metadata")
		}
	})

	t.Run("fileTwo hash error propagates", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Get actual hash for file1 so it passes
		hash1, _ := file.SHA256Sum(file1)

		e := &LedgerEntry{
			Metadata: map[string]string{
				"file":    file1,
				"sha256":  hash1,
				"fileTwo": file2,
				// no sha256Two - strict mode should error
			},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		err := e.procMetadata(true, false, 1, nil)
		if err == nil {
			t.Fatal("procMetadata() expected error for missing sha256Two in strict mode, got nil")
		}
		if !contains(err.Error(), "no hash for file") {
			t.Errorf("error should mention no hash, got: %v", err)
		}
	})

	t.Run("missing file metadata for Expenses warns but passes", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Test Entry",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		// Should pass (just logs warning)
		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("missing file metadata for Income warns but passes", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Salary",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Assets:Bank"},
				{Name: "Income:Salary"},
			},
		}

		// Should pass (just logs warning)
		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("missing file metadata but account in noMetadata skips warning", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Test Entry",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
				{Name: "Assets:Bank"},
			},
		}

		noMetadata := map[string]bool{"Expenses:Food": true}
		err := e.procMetadata(false, false, 1, noMetadata)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("missing file metadata for non-Expenses/Income no warning", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Transfer",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Assets:Bank"},
				{Name: "Assets:Cash"},
			},
		}

		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("entry with single Expenses account", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Single expense",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Expenses:Food"},
			},
		}

		// Should not panic with only one account
		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("entry with three accounts including Expenses", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Multi-account expense",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Assets:Bank"},
				{Name: "Assets:Cash"},
				{Name: "Expenses:Food"},
			},
		}

		// Should check all accounts for Expenses/Income, not just first two
		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})

	t.Run("entry with four accounts Income in third position", func(t *testing.T) {
		e := &LedgerEntry{
			Date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Name:     "Complex income entry",
			Metadata: map[string]string{},
			Accounts: []LedgerAccount{
				{Name: "Assets:Bank"},
				{Name: "Assets:Cash"},
				{Name: "Income:Salary"},
				{Name: "Liabilities:Tax"},
			},
		}

		// Should check all accounts for Expenses/Income
		err := e.procMetadata(false, false, 1, nil)
		if err != nil {
			t.Errorf("procMetadata() error = %v, want nil", err)
		}
	})
}

func TestValidateSubtree(t *testing.T) {
	t.Run("empty invoices directory with no files referenced", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		seenFiles := make(map[string]bool)
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
	})

	t.Run("PDF in invoices directory that is referenced", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		file1 := filepath.Join("invoices", "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		seenFiles := map[string]bool{file1: true}
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
		// File should be removed from seenFiles after processing
		if seenFiles[file1] {
			t.Error("file should have been removed from seenFiles")
		}
	})

	t.Run("PDF in invoices directory not referenced warns but passes", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		file1 := filepath.Join("invoices", "unreferenced.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		seenFiles := make(map[string]bool)
		// Should pass (just logs warning, doesn't error)
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
	})

	t.Run("non-PDF file in invoices directory is ignored", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		// Create a non-PDF file
		file1 := filepath.Join("invoices", "readme.txt")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		seenFiles := make(map[string]bool)
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
	})

	t.Run("file referenced but not in invoices directory", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		seenFiles := map[string]bool{"/some/other/path/invoice.pdf": true}
		err := validateSubtree(seenFiles)
		if err == nil {
			t.Fatal("validateSubtree() expected error for file not in filesystem, got nil")
		}
		if !contains(err.Error(), "file referenced in ledger but not found in filesystem") {
			t.Errorf("error should mention file not found, got: %v", err)
		}
	})

	t.Run("invoices directory does not exist", func(t *testing.T) {
		// Make sure invoices directory doesn't exist
		os.RemoveAll("invoices")

		seenFiles := make(map[string]bool)
		err := validateSubtree(seenFiles)
		if err == nil {
			t.Fatal("validateSubtree() expected error for missing invoices dir, got nil")
		}
		if !contains(err.Error(), "error traversing invoice subtree") {
			t.Errorf("error should mention traversing error, got: %v", err)
		}
	})

	t.Run("PDF in subdirectory of invoices", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join("invoices", "2024"), 0755); err != nil {
			t.Fatalf("failed to create invoices subdir: %v", err)
		}
		defer os.RemoveAll("invoices")

		file1 := filepath.Join("invoices", "2024", "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		seenFiles := map[string]bool{file1: true}
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
		if seenFiles[file1] {
			t.Error("file should have been removed from seenFiles")
		}
	})

	t.Run("multiple PDFs some referenced some not", func(t *testing.T) {
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		file1 := filepath.Join("invoices", "referenced.pdf")
		file2 := filepath.Join("invoices", "unreferenced.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		seenFiles := map[string]bool{file1: true}
		// Should pass - unreferenced file just logs warning
		if err := validateSubtree(seenFiles); err != nil {
			t.Errorf("validateSubtree() error = %v, want nil", err)
		}
		if seenFiles[file1] {
			t.Error("referenced file should have been removed from seenFiles")
		}
	})
}

func TestValidateMetadata(t *testing.T) {
	t.Run("non-strict mode returns nil", func(t *testing.T) {
		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file": "/nonexistent/file.pdf",
					},
				},
			},
		}
		// Non-strict mode should return nil without checking anything
		if err := l.validateMetadata(false); err != nil {
			t.Errorf("validateMetadata(false) error = %v, want nil", err)
		}
	})

	t.Run("entries without file metadata are skipped", func(t *testing.T) {
		// Create invoices directory to satisfy validateSubtree
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		l := &Ledger{
			Entries: []LedgerEntry{
				{Metadata: map[string]string{}},
				{Metadata: map[string]string{"note": "just a note"}},
			},
		}
		if err := l.validateMetadata(true); err != nil {
			t.Errorf("validateMetadata() error = %v, want nil", err)
		}
	})

	t.Run("duplicate flag skips validation", func(t *testing.T) {
		// Create invoices directory and put file there to satisfy validateSubtree
		if err := os.MkdirAll("invoices", 0755); err != nil {
			t.Fatalf("failed to create invoices dir: %v", err)
		}
		defer os.RemoveAll("invoices")

		file1 := filepath.Join("invoices", "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":   file1,
						"sha256": "abc123",
					},
				},
				{
					Metadata: map[string]string{
						"file":      file1,
						"sha256":    "abc123",
						"duplicate": "true",
					},
				},
			},
		}
		// Second entry is marked as duplicate, so should not error
		if err := l.validateMetadata(true); err != nil {
			t.Errorf("validateMetadata() error = %v, want nil", err)
		}
	})

	t.Run("duplicate file detection", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":   file1,
						"sha256": "hash1",
					},
				},
				{
					Metadata: map[string]string{
						"file":   file1,
						"sha256": "hash2",
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for duplicate file, got nil")
		}
		if !contains(err.Error(), "duplicate file") {
			t.Errorf("error should mention duplicate file, got: %v", err)
		}
	})

	t.Run("duplicate hash detection", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":   file1,
						"sha256": "samehash",
					},
				},
				{
					Metadata: map[string]string{
						"file":   file2,
						"sha256": "samehash",
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for duplicate hash, got nil")
		}
		if !contains(err.Error(), "duplicate hash") {
			t.Errorf("error should mention duplicate hash, got: %v", err)
		}
	})

	t.Run("fileTwo duplicate file detection", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":      file1,
						"sha256":    "hash1",
						"fileTwo":   file2,
						"sha256Two": "hash2",
					},
				},
				{
					Metadata: map[string]string{
						"file":   file2,
						"sha256": "hash3",
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for duplicate fileTwo, got nil")
		}
		if !contains(err.Error(), "duplicate file") {
			t.Errorf("error should mention duplicate file, got: %v", err)
		}
	})

	t.Run("fileTwo duplicate hash detection", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		file3 := filepath.Join(dir, "invoice3.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file3, []byte("content3"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":      file1,
						"sha256":    "hash1",
						"fileTwo":   file2,
						"sha256Two": "samehash",
					},
				},
				{
					Metadata: map[string]string{
						"file":   file3,
						"sha256": "samehash",
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for duplicate hash in fileTwo, got nil")
		}
		if !contains(err.Error(), "duplicate hash") {
			t.Errorf("error should mention duplicate hash, got: %v", err)
		}
	})

	t.Run("hash calculated when not provided", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		file2 := filepath.Join(dir, "invoice2.pdf")
		// Same content = same hash
		if err := os.WriteFile(file1, []byte("identical content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("identical content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file": file1,
						// no sha256 provided - will be calculated
					},
				},
				{
					Metadata: map[string]string{
						"file": file2,
						// no sha256 provided - will be calculated
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for duplicate calculated hash, got nil")
		}
		if !contains(err.Error(), "duplicate hash") {
			t.Errorf("error should mention duplicate hash, got: %v", err)
		}
	})

	t.Run("hash calculation error for missing file", func(t *testing.T) {
		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file": "/nonexistent/file.pdf",
						// no sha256 provided - will try to calculate
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for missing file, got nil")
		}
		if !contains(err.Error(), "failed to calculate SHA256") {
			t.Errorf("error should mention SHA256 calculation failure, got: %v", err)
		}
	})

	t.Run("fileTwo hash calculation error", func(t *testing.T) {
		dir := t.TempDir()
		file1 := filepath.Join(dir, "invoice1.pdf")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		l := &Ledger{
			Entries: []LedgerEntry{
				{
					Metadata: map[string]string{
						"file":    file1,
						"sha256":  "hash1",
						"fileTwo": "/nonexistent/file.pdf",
						// no sha256Two provided - will try to calculate
					},
				},
			},
		}
		err := l.validateMetadata(true)
		if err == nil {
			t.Fatal("validateMetadata() expected error for missing fileTwo, got nil")
		}
		if !contains(err.Error(), "failed to calculate SHA256") {
			t.Errorf("error should mention SHA256 calculation failure, got: %v", err)
		}
	})
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
