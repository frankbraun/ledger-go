package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frankbraun/ledger-go/ledger"
)

func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, err
}

type flags struct {
	file       string
	priceDB    string
	noMetadata string
	strict     bool
	noPager    bool

	// extensions
	addMissingHashes bool
}

func defineFlags() *flags {
	var f flags
	flag.StringVar(&f.file, "file", "", "Read journal data from FILE.")
	flag.StringVar(&f.priceDB, "price-db", "", "Read price DB from FILE.")
	flag.StringVar(&f.noMetadata, "no-metadata", "no-metadata.conf", "Read no metadata configuration from FILE.")
	flag.BoolVar(&f.strict, "strict", false,
		"Accounts or commodities not previously declared will cause warnings.")
	flag.BoolVar(&f.noPager, "no-pager", false,
		"Disables the pager on TTY output.")

	// extensions
	flag.BoolVar(&f.addMissingHashes, "add-missing-hashes", false,
		"Add missing SHA256 hashes for file metadata")
	return &f
}

func parseLedgerRC(f *flags) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fn := filepath.Join(homeDir, ".ledgerrc")
	exists, err := fileExists(fn)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	b, err := os.ReadFile(fn)
	if err != nil {
		return err
	}
	s := strings.TrimRight(string(b), "\n")
	s = strings.Replace(s, "\n", " ", -1)
	sb := strings.Split(s, " ")
	if err := flag.CommandLine.Parse(sb); err != nil {
		return err
	}
	f.file = strings.Replace(f.file, "~", homeDir, 1)
	f.priceDB = strings.Replace(f.priceDB, "~", homeDir, 1)
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s: error: %s\n", os.Args[0], err)
	os.Exit(1)
}

func main() {
	f := defineFlags()
	// parse flags from .ledgerrc
	if err := parseLedgerRC(f); err != nil {
		fatal(err)
	}
	// parse command line flags
	flag.Parse()
	l, err := ledger.New(f.file, f.strict, f.addMissingHashes, f.noMetadata)
	if err != nil {
		fatal(err)
	}
	l.Print()
}
