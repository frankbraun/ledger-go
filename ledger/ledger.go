package ledger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/frankbraun/ledger-go/util/file"
)

// DateFormat is the standard date format used in ledger entries.
const DateFormat = "2006/01/02"

// AccountWidth is the width of the account column in the ledger.
const AccountWidth = 46

// invoiceSubtree is the directory containing the invoice PDFs.
const invoiceSubtree = "invoices"

// LedgerAccount defines a single account in a ledger entry.
type LedgerAccount struct {
	Name           string
	Amount         float64
	Commodity      string
	PriceType      string  // "", "@" (per-unit), or "@@" (total cost)
	PriceAmount    float64
	PriceCommodity string
}

// Print prints the LedgerAccount to stdout.
func (a *LedgerAccount) Print() {
	if a.Commodity != "" {
		buf := strings.Repeat(" ", AccountWidth-len(a.Name))
		printSum := strings.ReplaceAll(fmt.Sprintf("%.2f", a.Amount), ".", ",")
		if a.PriceType != "" {
			printPrice := strings.ReplaceAll(fmt.Sprintf("%.2f", a.PriceAmount), ".", ",")
			fmt.Printf("  %s%s  %s %s %s %s %s\n",
				a.Name, buf, printSum, a.Commodity,
				a.PriceType, printPrice, a.PriceCommodity)
		} else {
			fmt.Printf("  %s%s  %s %s\n", a.Name, buf, printSum, a.Commodity)
		}
	} else {
		fmt.Printf("  %s\n", a.Name)
	}
}

// LedgerEntry represents a single entry in the ledger with one or more accounts.
type LedgerEntry struct {
	Date          time.Time
	EffectiveDate time.Time
	Name          string
	Accounts      []LedgerAccount
	Metadata      map[string]string // optional
}

// balanceEpsilon is the tolerance for floating-point balance comparisons.
const balanceEpsilon = 0.005

// balanceAmount returns the amount and commodity to use for balance calculation.
// If the account has a price annotation, the amount is converted to the price commodity:
//   - @ (per-unit): returns Amount * PriceAmount in PriceCommodity
//   - @@ (total cost): returns PriceAmount (with sign of Amount) in PriceCommodity
//
// Otherwise returns the original Amount and Commodity.
func (a *LedgerAccount) balanceAmount() (float64, string) {
	if a.PriceType == "" {
		return a.Amount, a.Commodity
	}
	if a.PriceType == "@" {
		// Per-unit price: total = amount * price
		return a.Amount * a.PriceAmount, a.PriceCommodity
	}
	// Total cost (@@): use price amount with the sign of the original amount
	if a.Amount < 0 {
		return -a.PriceAmount, a.PriceCommodity
	}
	return a.PriceAmount, a.PriceCommodity
}

// validateBalance checks that the entry is balanced (amounts sum to zero per commodity).
// If exactly one account has an elided amount (no commodity), it calculates and sets
// the missing amount. Returns an error if the entry is unbalanced or has multiple
// elided amounts.
//
// Price annotations affect balance calculation:
//   - @ (per-unit): 10 BTC @ 50000 EUR contributes 500000 EUR to balance
//   - @@ (total cost): 10 BTC @@ 500000 EUR contributes 500000 EUR to balance
func (e *LedgerEntry) validateBalance(startLine int) error {
	// Find accounts with elided amounts (no commodity set)
	var elidedIdx = -1
	for i, a := range e.Accounts {
		if a.Commodity == "" {
			if elidedIdx >= 0 {
				return fmt.Errorf("ledger: line %d: multiple accounts with elided amounts", startLine)
			}
			elidedIdx = i
		}
	}

	// Sum amounts by commodity (using balance amounts for price conversions)
	sums := make(map[string]float64)
	for i := range e.Accounts {
		if i == elidedIdx {
			continue // skip elided account for now
		}
		amount, commodity := e.Accounts[i].balanceAmount()
		sums[commodity] += amount
	}

	// If there's an elided amount, calculate it
	if elidedIdx >= 0 {
		if len(sums) == 0 {
			return fmt.Errorf("ledger: line %d: cannot infer elided amount without other amounts", startLine)
		}
		if len(sums) == 1 {
			// Single commodity: set the elided amount to balance the entry
			for commodity, sum := range sums {
				e.Accounts[elidedIdx].Amount = -sum
				e.Accounts[elidedIdx].Commodity = commodity
			}
		}
		// Multiple commodities: the elided account implicitly receives balancing
		// amounts for each commodity. We can't represent this in a single
		// LedgerAccount, so leave the elided account as-is (no amount/commodity).
		// The entry is considered balanced by construction.
		return nil
	}

	// No elided amount - verify balance
	// If multiple commodities are present (after price conversion), skip balance
	// validation - Ledger tracks each commodity independently (e.g., currency exchange)
	if len(sums) > 1 {
		return nil
	}

	// Single commodity: verify it sums to zero
	for commodity, sum := range sums {
		if sum < -balanceEpsilon || sum > balanceEpsilon {
			return fmt.Errorf("ledger: line %d: entry not balanced for %s (off by %.2f)",
				startLine, commodity, sum)
		}
	}

	return nil
}

// Print prints the LedgerEntry to stdout.
func (e *LedgerEntry) Print() {
	if e.EffectiveDate.IsZero() {
		fmt.Printf("%s %s\n", e.Date.Format(DateFormat), e.Name)
	} else {
		fmt.Printf("%s=%s %s\n", e.Date.Format(DateFormat),
			e.EffectiveDate.Format(DateFormat), e.Name)
	}
	for _, a := range e.Accounts {
		a.Print()
	}
	if e.Metadata != nil {
		var tags []string
		for tag := range e.Metadata {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		for _, tag := range tags {
			fmt.Printf("    ; %s: %s\n", tag, e.Metadata[tag])
		}
	}
}

// parseMetadata parses a single metadata line and adds it to the LedgerEntry's Metadata map.
func (e *LedgerEntry) parseMetadata(line string, ln int) error {
	elems := strings.Split(line, ":")
	if len(elems) != 2 {
		return fmt.Errorf("ledger: line %d: not metadata: %s", ln, line)
	}
	tag := strings.TrimSpace(strings.TrimPrefix(elems[0], ";"))
	value := strings.TrimSpace(elems[1])
	_, ok := e.Metadata[tag]
	if ok {
		return fmt.Errorf("ledger: line %d: metadata tag already exists: %s", ln, line)
	}
	e.Metadata[tag] = value
	return nil
}

func procFilename(filename string) error {
	exists, err := file.Exists(filename)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("ledger: file doesn't exist: %s", filename)
	}
	if !strings.HasSuffix(filename, ".pdf") {
		return fmt.Errorf("ledger: file is not a PDF: %s", filename)
	}
	return nil
}

func (e *LedgerEntry) procHash(
	metadataKey string,
	filename string,
	strict bool,
	addMissingHashes bool,
	ln int,
) error {
	hash, ok := e.Metadata[metadataKey]
	if ok {
		if strict {
			// check hash
			h, err := file.SHA256Sum(filename)
			if err != nil {
				return err
			}
			if h != hash {
				return fmt.Errorf("ledger: line %d: hash mismatch for file: %s",
					ln, filename)
			}
		}
	} else {
		if addMissingHashes {
			// add missing SHA256 hash
			h, err := file.SHA256Sum(filename)
			if err != nil {
				return err
			}
			e.Metadata[metadataKey] = h
		} else if strict {
			return fmt.Errorf("ledger: no hash for file (use -add-missing-hashes): %s", filename)
		}
	}
	return nil
}

// procMetadata checks if a single ledger entry has metadata and validates it.
func (e *LedgerEntry) procMetadata(
	strict, addMissingHashes bool,
	ln int,
	noMetadata map[string]bool,
) error {
	filenameDefined := false
	if e.Metadata != nil {
		filename, ok := e.Metadata["file"]
		if ok {
			if err := procFilename(filename); err != nil {
				return err
			}
			filenameDefined = true
		}
		filenameTwo, ok := e.Metadata["fileTwo"]
		if ok {
			if !filenameDefined {
				return fmt.Errorf("ledger: line %d: 'fileTwo' defined but not 'file'", ln)
			}
			if err := procFilename(filenameTwo); err != nil {
				return err
			}
		}
		err := e.procHash("sha256", filename, strict, addMissingHashes, ln)
		if err != nil {
			return err
		}
		if filenameTwo != "" {
			err = e.procHash("sha256Two", filenameTwo, strict, addMissingHashes, ln)
			if err != nil {
				return err
			}
		}
	}

	// make sure file metadata is defined where needed
	if !filenameDefined {
		skip := false
		for _, a := range e.Accounts {
			if noMetadata[a.Name] {
				skip = true
				break
			}
		}
		if !skip {
			// only enforce metadata lines for expenses or income
			hasExpenseOrIncome := false
			for _, a := range e.Accounts {
				if strings.HasPrefix(a.Name, "Expenses:") ||
					strings.HasPrefix(a.Name, "Income:") {
					hasExpenseOrIncome = true
					break
				}
			}
			if hasExpenseOrIncome {
				warning(fmt.Sprintf("file metadata missing for: %s %s",
					e.Date.Format(DateFormat), e.Name))
			}
		}
	}

	return nil
}

const (
	parseHeaderComments = iota
	parseCommodities
	parseAccounts
	parseTags
	parseEntries
)

// Ledger represents the entire ledger, including header comments, commodities,
// accounts, and entries.
type Ledger struct {
	HeaderComments []string
	Commodities    map[string]bool
	Accounts       map[string]bool
	Tags           map[string]bool
	Entries        []LedgerEntry

	// config
	NoMetadata map[string]bool
}

// parseAccount parses a single account line and returns a LedgerAccount.
// Supported formats:
//   - AccountName (elided amount)
//   - AccountName Amount Commodity
//   - AccountName Amount Commodity @ PriceAmount PriceCommodity (per-unit price)
//   - AccountName Amount Commodity @@ PriceAmount PriceCommodity (total cost)
func parseAccount(
	line string,
	ln int,
	strict bool,
	commodities map[string]bool,
	accounts map[string]bool,
) (LedgerAccount, error) {
	var a LedgerAccount

	elems := strings.Fields(line)
	if len(elems) != 1 && len(elems) != 3 && len(elems) != 6 {
		return a, fmt.Errorf("ledger: line %d: invalid account format (expected 1, 3, or 6 elements, got %d)", ln, len(elems))
	}
	account := elems[0]
	if strict && !accounts[account] {
		return a, fmt.Errorf("ledger: line %d: account unknown: %s", ln, account)
	}
	a.Name = account

	if len(elems) >= 3 {
		amount := strings.ReplaceAll(elems[1], ",", ".")
		var err error
		a.Amount, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			return a, fmt.Errorf("ledger: line %d: %s", ln, err)
		}
		commodity := elems[2]
		if strict && !commodities[commodity] {
			return a, fmt.Errorf("ledger: line %d: commodity unknown: %s", ln, commodity)
		}
		a.Commodity = commodity
	}

	if len(elems) == 6 {
		// Parse price annotation
		priceType := elems[3]
		if priceType != "@" && priceType != "@@" {
			return a, fmt.Errorf("ledger: line %d: invalid price annotation (expected @ or @@, got %s)", ln, priceType)
		}
		a.PriceType = priceType

		priceAmount := strings.ReplaceAll(elems[4], ",", ".")
		var err error
		a.PriceAmount, err = strconv.ParseFloat(priceAmount, 64)
		if err != nil {
			return a, fmt.Errorf("ledger: line %d: invalid price amount: %s", ln, err)
		}

		priceCommodity := elems[5]
		if strict && !commodities[priceCommodity] {
			return a, fmt.Errorf("ledger: line %d: price commodity unknown: %s", ln, priceCommodity)
		}
		a.PriceCommodity = priceCommodity
	}

	return a, nil
}

// parseEntry parses a single entry and returns the corresponding LedgerEntry.
func parseEntry(
	scanner *bufio.Scanner,
	line string,
	ln *int,
	previousDate *time.Time,
	strict bool,
	addMissingHashes bool,
	commodities map[string]bool,
	accounts map[string]bool,
	noMetadata map[string]bool,
) (*LedgerEntry, error) {
	var (
		e         LedgerEntry
		name      string
		err       error
		startLine = *ln // remember starting line for error messages
	)

	// parse date line
	parts := strings.SplitN(line, " ", 2)
	date := parts[0]
	if len(parts) > 1 {
		name = parts[1]
	}

	if strings.Contains(date, "=") {
		// parse with effective date
		parts := strings.SplitN(date, "=", 2)
		accountingDate := parts[0]
		effectiveDate := parts[1]
		e.Date, err = time.Parse(DateFormat, accountingDate)
		if err != nil {
			return nil, fmt.Errorf("ledger: line %d: %s", *ln, err)
		}
		e.EffectiveDate, err = time.Parse(DateFormat, effectiveDate)
		if err != nil {
			return nil, fmt.Errorf("ledger: line %d: %s", *ln, err)
		}
	} else {
		// parse without effective date
		e.Date, err = time.Parse(DateFormat, date)
		if err != nil {
			return nil, fmt.Errorf("ledger: line %d: %s", *ln, err)
		}
	}
	e.Name = name

	// make sure dates are in ascending order
	var currentDate time.Time
	if e.EffectiveDate.IsZero() {
		currentDate = e.Date
	} else {
		currentDate = e.EffectiveDate
	}
	if currentDate.Before(*previousDate) {
		return nil, fmt.Errorf("ledger: line %d: %s is before %s", *ln,
			e.Date.Format(DateFormat), previousDate.Format(DateFormat))
	}
	if e.EffectiveDate.IsZero() {
		*previousDate = e.Date
	} else {
		*previousDate = e.EffectiveDate
	}

	// parse accounts
	metadataMode := false
	for scanner.Scan() {
		line = scanner.Text()
		(*ln)++
		if line == "" {
			// entry finished - validate balance and metadata
			if err := e.validateBalance(startLine); err != nil {
				return nil, err
			}
			if err := e.procMetadata(strict, addMissingHashes, *ln-1, noMetadata); err != nil {
				return nil, err
			}
			return &e, nil
		}

		if !strings.HasPrefix(line, "  ") {
			return nil, fmt.Errorf("ledger: line %d: not an account line", *ln)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ";") {
			metadataMode = true
			if e.Metadata == nil {
				e.Metadata = make(map[string]string)
			}
			if err := e.parseMetadata(line, *ln); err != nil {
				return nil, err
			}
		} else {
			if metadataMode {
				return nil, fmt.Errorf("ledger: line %d: already parsing metadata", *ln)
			}
			a, err := parseAccount(line, *ln, strict, commodities, accounts)
			if err != nil {
				return nil, err
			}
			e.Accounts = append(e.Accounts, a)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// last entry in file (no trailing newline) - validate balance
	if err := e.validateBalance(startLine); err != nil {
		return nil, err
	}
	return &e, nil
}

// warning prints a warning to stderr.
func warning(warn string) {
	fmt.Fprintf(os.Stderr, "%s: warning: %s\n", os.Args[0], warn)
}

func (l *Ledger) parseNoMetadataFile(noMetadataFilename string) error {
	l.NoMetadata = make(map[string]bool)
	if noMetadataFilename == "" {
		return nil
	}
	fp, err := os.Open(noMetadataFilename)
	if err != nil {
		return err
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		l.NoMetadata[scanner.Text()] = true
	}
	return scanner.Err()
}

// New creates a new Ledger from a file, if strict is true, the ledger is
// validated more strictly. If addMissingHashes is true, missing SHA256
// hashes are added to the ledger. If noMetadataFilename is not empty, read
// accounts for which no metadata is required from file.
func New(
	filename string,
	strict, addMissingHashes bool,
	noMetadataFilename string,
) (*Ledger, error) {
	var l Ledger
	l.Commodities = make(map[string]bool)
	l.Accounts = make(map[string]bool)
	l.Tags = make(map[string]bool)
	if err := l.parseNoMetadataFile(noMetadataFilename); err != nil {
		return nil, err
	}
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	state := parseHeaderComments
	scanner := bufio.NewScanner(fp)
	ln := 0
	previousDate := time.Unix(0, 0)
	for scanner.Scan() {
		line := scanner.Text()
		ln++
		if len(line) == 0 {
			// skip empty lines
			continue
		}
		if state == parseHeaderComments {
			if strings.HasPrefix(line, ";") {
				l.HeaderComments = append(l.HeaderComments, line)
				continue
			} else {
				state = parseCommodities
			}
		}
		if state == parseCommodities {
			if strings.HasPrefix(line, "commodity ") {
				l.Commodities[strings.TrimPrefix(line, "commodity ")] = true
				continue
			} else {
				state = parseAccounts
			}
		}
		if state == parseAccounts {
			if strings.HasPrefix(line, "account ") {
				l.Accounts[strings.TrimPrefix(line, "account ")] = true
				continue
			} else {
				state = parseTags
			}
		}
		if state == parseTags {
			if strings.HasPrefix(line, "tag ") {
				l.Tags[strings.TrimPrefix(line, "tag ")] = true
				continue
			} else {
				state = parseEntries
			}
		}
		if state == parseTags || state == parseEntries {
			if strings.HasPrefix(line, ";") {
				// skip
				warning(fmt.Sprintf("line %d: skipping comment", ln))
				continue
			}
			e, err := parseEntry(scanner, line, &ln, &previousDate, strict,
				addMissingHashes, l.Commodities, l.Accounts, l.NoMetadata)
			if err != nil {
				return nil, err
			}
			l.Entries = append(l.Entries, *e)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if err := l.validateMetadata(strict); err != nil {
		return nil, err
	}

	return &l, nil
}

func validateSubtree(seenFiles map[string]bool) error {
	// Traverse the invoice subtree
	err := filepath.Walk(invoiceSubtree, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process PDF files
		if !strings.HasSuffix(info.Name(), ".pdf") {
			return nil
		}

		// Check if the file has been seen
		if !seenFiles[path] {
			//return fmt.Errorf("file not referenced in ledger: %s", path)
			warning(fmt.Sprintf("file not referenced in ledger: %s", path))
		} else {
			// Mark the file as processed
			delete(seenFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error traversing invoice subtree: %v", err)
	}

	// Check if there are any files in the ledger that don't exist in the filesystem
	for file := range seenFiles {
		return fmt.Errorf("file referenced in ledger but not found in filesystem: %s", file)
	}

	return nil
}

func (l *Ledger) validateMetadata(strict bool) error {
	// only validate metadata in strict mode
	if !strict {
		return nil
	}

	// make sure no two files have the same hash and files are not referenced twice
	seenHashes := make(map[string]string)
	seenFiles := make(map[string]bool)
	for _, entry := range l.Entries {
		// skip entries without file metadata
		if entry.Metadata["file"] == "" {
			continue
		}

		// skip entries which are marked as duplicates
		if entry.Metadata["duplicate"] == "true" {
			continue
		}

		// make sure no file is referenced twice
		if seenFiles[entry.Metadata["file"]] {
			return fmt.Errorf("ledger: duplicate file: %s", entry.Metadata["file"])
		}
		seenFiles[entry.Metadata["file"]] = true

		hash, ok := entry.Metadata["sha256"]
		if !ok {
			var err error
			hash, err = file.SHA256Sum(entry.Metadata["file"])
			if err != nil {
				return fmt.Errorf("ledger: failed to calculate SHA256 hash for file '%s': %v",
					entry.Metadata["file"], err)
			}
		}
		if _, ok := seenHashes[hash]; ok {
			return fmt.Errorf("ledger: duplicate hash for files '%s' and '%s'",
				seenHashes[hash], entry.Metadata["file"])
		}
		seenHashes[hash] = entry.Metadata["file"]

		// skip entries without fileTwo metadata
		if entry.Metadata["fileTwo"] == "" {
			continue
		}

		// make sure no file is referenced twice
		if seenFiles[entry.Metadata["fileTwo"]] {
			return fmt.Errorf("ledger: duplicate file: %s", entry.Metadata["fileTwo"])
		}
		seenFiles[entry.Metadata["fileTwo"]] = true

		hash, ok = entry.Metadata["sha256Two"]
		if !ok {
			var err error
			hash, err = file.SHA256Sum(entry.Metadata["fileTwo"])
			if err != nil {
				return fmt.Errorf("ledger: failed to calculate SHA256 hash for file '%s': %v",
					entry.Metadata["fileTwo"], err)
			}
		}
		if _, ok := seenHashes[hash]; ok {
			return fmt.Errorf("ledger: duplicate hash for files '%s' and '%s'",
				seenHashes[hash], entry.Metadata["fileTwo"])
		}
		seenHashes[hash] = entry.Metadata["fileTwo"]
	}

	// make sure every PDF file in the invoice subtree is referenced at least once
	if err := validateSubtree(seenFiles); err != nil {
		return err
	}

	return nil
}

// Print outputs the entire Ledger to stdout.
func (l *Ledger) Print() {
	if len(l.HeaderComments) > 0 {
		for _, line := range l.HeaderComments {
			fmt.Println(line)
		}
		fmt.Println()
	}
	if len(l.Commodities) > 0 {
		var commodities []string
		for c := range l.Commodities {
			commodities = append(commodities, c)
		}
		sort.Strings(commodities)
		for _, c := range commodities {
			fmt.Printf("commodity %s\n", c)
		}
		fmt.Println()
	}
	if len(l.Accounts) > 0 {
		var accounts []string
		for a := range l.Accounts {
			accounts = append(accounts, a)
		}
		sort.Strings(accounts)
		for _, a := range accounts {
			fmt.Printf("account %s\n", a)
		}
		fmt.Println()
	}
	if len(l.Tags) > 0 {
		var tags []string
		for t := range l.Tags {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		for _, t := range tags {
			fmt.Printf("tag %s\n", t)
		}
		fmt.Println()
	}
	for i, entry := range l.Entries {
		if i > 0 {
			fmt.Println()
		}
		entry.Print()
	}
}
