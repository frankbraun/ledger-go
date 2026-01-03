package ledger

import (
	"os"
	"path/filepath"
	"testing"
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
