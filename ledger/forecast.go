package ledger

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

/*
type LedgerAccount struct {
	Name      string
	Amount    float64
	Commodity string
}

type LedgerEntry struct {
	Date          time.Time
	EffectiveDate time.Time
	Name          string
	Accounts      []LedgerAccount
	Metadata      map[string]string // optional
}
*/

func (l *Ledger) Forecast(includeCurrent bool) error {
	now := time.Now().UTC()
	y := now.Year()
	m := now.Month()
	if !includeCurrent {
		if m == 1 {
			m = 12
			y--
		} else {
			m--
		}
	}
	start := time.Date(y-2, m, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(y-1, m, 1, 0, 0, 0, 0, time.UTC)

	//fmt.Println(cutoff.Format(time.DateOnly))

	var (
		account string
		amount  float64
	)
	accMap := make(map[string]float64)
	//var firstDate time.Time
	for _, entry := range l.Entries {
		if entry.Date.Before(start) || entry.Date.After(end) {
			continue
		}
		if strings.HasPrefix(entry.Accounts[0].Name, "Expenses:") {
			if entry.Accounts[0].Amount == 0.0 {
				entry.Print()
				return errors.New("ledger: zero amount")
			}

			account = entry.Accounts[0].Name
			// TODO
			if entry.Accounts[0].Commodity != "USD" &&
				entry.Accounts[0].Commodity != "AED" {
				return errors.New("ledger: only USD and AED supported so far")
			}
			// TODO
			if entry.Accounts[0].Commodity == "USD" {
				amount = entry.Accounts[0].Amount
			} else {
				amount = entry.Accounts[0].Amount / 3.6725
			}
		} else if strings.HasPrefix(entry.Accounts[len(entry.Accounts)-1].Name, "Income:") {
			if len(entry.Accounts) != 2 {
				return errors.New("ledger: income entries can only have 2 accounts so far")
			}
			if entry.Accounts[0].Amount == 0.0 {
				entry.Print()
				return errors.New("ledger: zero amount")
			}

			account = entry.Accounts[1].Name
			// TODO
			if entry.Accounts[0].Commodity != "USD" &&
				entry.Accounts[0].Commodity != "AED" {
				return errors.New("ledger: only USD and AED supported so far")
			}
			// TODO
			if entry.Accounts[0].Commodity == "USD" {
				amount = entry.Accounts[0].Amount
			} else {
				amount = entry.Accounts[0].Amount / 3.6725
			}

		} else {
			continue
		}
		accMap[account] += amount
	}

	// take average
	var divisor int
	//y1 := firstDate.Year()
	//m1 := int(firstDate.Month())
	//y2 := cutoff.Year()
	//m2 := int(cutoff.Month())
	/*
		if y1 == y2 {
			divisor = m2 - m1
		}
	*/
	fmt.Printf("divisor = %d\n", divisor)
	for acc, amount := range accMap {
		accMap[acc] = amount / float64(divisor)
	}

	// sort account names
	var accounts []string
	var maxLen int
	for acc := range accMap {
		accounts = append(accounts, acc)
		if len(acc) > maxLen {
			maxLen = len(acc)
		}
	}
	sort.Strings(accounts)

	for _, acc := range accounts {
		fmt.Printf("%s%s USD %.2f\n",
			acc,
			strings.Repeat(" ", maxLen-len(acc)),
			accMap[acc])
	}

	return nil
}
