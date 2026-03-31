package accountant

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/report"
	"github.com/chamzzzzzz/accounting/sourcedocument/processor"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/voucher"
)

type (
	Account               = account.Account
	Voucher               = voucher.Voucher
	Journal               = journal.Journal
	ReportParameters      = report.ReportParameters
	AccountBalanceReport  = report.AccountBalanceReport
	AccountRegisterReport = report.AccountRegisterReport
)

var (
	ErrNoBook       = errors.New("no book")
	ErrTrialBalance = errors.New("trial balance error")
)

type Accountant struct {
	Scanners   []scanner.Scanner
	Processors []processor.Processor
	Book       *book.Book
}

func normalizeAccountTitle(title string) string {
	parts := strings.Split(title, ":")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, ":")
}

func roundAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

func formatAmount(v float64) string {
	return strconv.FormatFloat(roundAmount(v), 'f', 2, 64)
}

func (a *Accountant) AddAccount(account *Account) error {
	if a.Book == nil {
		return ErrNoBook
	}
	a.Book.Accounts = append(a.Book.Accounts, account)
	return nil
}

func (a *Accountant) DelAccount(title string) error {
	if a.Book == nil {
		return ErrNoBook
	}
	for i, acc := range a.Book.Accounts {
		if acc.Title == title {
			a.Book.Accounts = append(a.Book.Accounts[:i], a.Book.Accounts[i+1:]...)
			return nil
		}
	}
	return nil
}

func (a *Accountant) GetAccount(title string) (*Account, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	for _, acc := range a.Book.Accounts {
		if acc.Title == title {
			return acc, nil
		}
	}
	return nil, nil
}

func (a *Accountant) AddJournal(journal *Journal) error {
	if a.Book == nil {
		return ErrNoBook
	}
	a.Book.Journals = append(a.Book.Journals, journal)
	return nil
}

func (a *Accountant) DelJournal(date string, catalog string) error {
	if a.Book == nil {
		return ErrNoBook
	}
	for i, j := range a.Book.Journals {
		if j.Date == date && j.Catalog == catalog {
			a.Book.Journals = append(a.Book.Journals[:i], a.Book.Journals[i+1:]...)
			return nil
		}
	}
	return nil
}

func (a *Accountant) GetJournal(date string, catalog string) (*Journal, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	for _, j := range a.Book.Journals {
		if j.Date == date && j.Catalog == catalog {
			return j, nil
		}
	}
	return nil, nil
}

func (a *Accountant) CheckTrialBalance(voucher *Voucher) error {
	totals := make(map[string]float64)
	for _, entry := range voucher.Entries {
		if entry.Amount == nil {
			continue
		}
		q, err := strconv.ParseFloat(entry.Amount.Quantity, 64)
		if err != nil {
			return err
		}
		totals[entry.Amount.Currency] = roundAmount(totals[entry.Amount.Currency] + q)
	}

	for _, total := range totals {
		if math.Abs(total) > 0.001 {
			return ErrTrialBalance
		}
	}
	return nil
}

func (a *Accountant) AddVoucher(voucher *Voucher) error {
	if a.Book == nil {
		return ErrNoBook
	}

	if err := a.CheckTrialBalance(voucher); err != nil {
		return err
	}

	t, err := time.ParseInLocation(time.RFC3339, voucher.Date, time.Local)
	if err != nil {
		return err
	}
	date := t.Format("2006-01")
	j, err := a.GetJournal(date, voucher.Catalog)
	if err != nil {
		return err
	}
	if j == nil {
		j = &Journal{
			Date:    date,
			Catalog: voucher.Catalog,
		}
		if err := a.AddJournal(j); err != nil {
			return err
		}
	}
	j.Vouchers = append(j.Vouchers, voucher)
	return nil
}

func (a *Accountant) ReportAccountBalance(parameters *ReportParameters) (*AccountBalanceReport, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}

	nodes := map[string]*report.AccountBalance{}
	rootOrder := []string{}
	childAttached := map[string]map[string]bool{}

	ensureNode := func(path string) *report.AccountBalance {
		if node, ok := nodes[path]; ok {
			return node
		}
		node := &report.AccountBalance{Title: path}
		nodes[path] = node
		if !strings.Contains(path, ":") {
			rootOrder = append(rootOrder, path)
		}
		return node
	}

	ensurePath := func(title string) {
		normalized := normalizeAccountTitle(title)
		if normalized == "" {
			return
		}
		parts := strings.Split(normalized, ":")
		for i := 0; i < len(parts); i++ {
			current := strings.Join(parts[:i+1], ":")
			ensureNode(current)
			if i == 0 {
				continue
			}
			parent := strings.Join(parts[:i], ":")
			parentNode := ensureNode(parent)
			if childAttached[parent] == nil {
				childAttached[parent] = map[string]bool{}
			}
			if !childAttached[parent][current] {
				parentNode.Children = append(parentNode.Children, ensureNode(current))
				childAttached[parent][current] = true
			}
		}
	}

	directTotals := map[string]map[string]float64{}
	addAmount := func(title, currency string, value float64) {
		if directTotals[title] == nil {
			directTotals[title] = map[string]float64{}
		}
		directTotals[title][currency] = roundAmount(directTotals[title][currency] + value)
	}

	for _, j := range a.Book.Journals {
		if j == nil {
			continue
		}
		for _, v := range j.Vouchers {
			if v == nil {
				continue
			}
			for _, e := range v.Entries {
				if e == nil || e.Amount == nil {
					continue
				}
				title := normalizeAccountTitle(e.Account)
				if title == "" {
					continue
				}
				ensurePath(title)
				quantity, err := strconv.ParseFloat(e.Amount.Quantity, 64)
				if err != nil {
					continue
				}
				addAmount(title, e.Amount.Currency, quantity)
			}
		}
	}

	for _, node := range nodes {
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Title < node.Children[j].Title
		})
	}

	buildAmounts := func(totals map[string]float64) []*amount.Amount {
		if len(totals) == 0 {
			return nil
		}
		currencies := make([]string, 0, len(totals))
		for currency := range totals {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		amounts := make([]*amount.Amount, 0, len(currencies))
		for _, currency := range currencies {
			amounts = append(amounts, &amount.Amount{
				Quantity: formatAmount(totals[currency]),
				Currency: currency,
			})
		}
		return amounts
	}

	var aggregate func(path string) map[string]float64
	aggregate = func(path string) map[string]float64 {
		totals := map[string]float64{}
		for currency, value := range directTotals[path] {
			totals[currency] = roundAmount(totals[currency] + value)
		}
		node := nodes[path]
		if node != nil {
			for _, child := range node.Children {
				childTotals := aggregate(child.Title)
				for currency, value := range childTotals {
					totals[currency] = roundAmount(totals[currency] + value)
				}
			}
			node.Amounts = buildAmounts(totals)
		}
		return totals
	}

	for _, root := range rootOrder {
		aggregate(root)
	}

	selectAll := parameters == nil || len(parameters.Titles) == 0
	targetTitles := map[string]bool{}
	if !selectAll {
		for _, title := range parameters.Titles {
			normalized := normalizeAccountTitle(title)
			if normalized == "" {
				selectAll = true
				break
			}
			targetTitles[normalized] = true
		}
	}

	balance := []*report.AccountBalance{}
	for _, root := range rootOrder {
		node := nodes[root]
		if node == nil {
			continue
		}
		if selectAll {
			balance = append(balance, node)
		} else {
			var filterNode func(*report.AccountBalance) *report.AccountBalance
			filterNode = func(n *report.AccountBalance) *report.AccountBalance {
				if targetTitles[n.Title] {
					return &report.AccountBalance{
						Title:    n.Title,
						Amounts:  n.Amounts,
						Children: n.Children,
					}
				}

				isPrefixOfTarget := false
				for target := range targetTitles {
					if strings.HasPrefix(target, n.Title+":") {
						isPrefixOfTarget = true
						break
					}
				}

				if isPrefixOfTarget {
					var newChildren []*report.AccountBalance
					newTotals := make(map[string]float64)
					for _, child := range n.Children {
						if filteredChild := filterNode(child); filteredChild != nil {
							newChildren = append(newChildren, filteredChild)
							for _, a := range filteredChild.Amounts {
								v, _ := strconv.ParseFloat(a.Quantity, 64)
								newTotals[a.Currency] = roundAmount(newTotals[a.Currency] + v)
							}
						}
					}
					if len(newChildren) > 0 {
						return &report.AccountBalance{
							Title:    n.Title,
							Amounts:  buildAmounts(newTotals),
							Children: newChildren,
						}
					}
				}

				for target := range targetTitles {
					if strings.HasPrefix(n.Title, target+":") {
						return n
					}
				}

				return nil
			}
			if filtered := filterNode(node); filtered != nil {
				balance = append(balance, filtered)
			}
		}
	}

	return &report.AccountBalanceReport{Balance: balance}, nil
}

func (a *Accountant) ReportAccountRegister(parameters *ReportParameters) (*AccountRegisterReport, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}

	matchAll := false
	targetAccounts := make(map[string]bool)
	if parameters != nil {
		for _, title := range parameters.Titles {
			normalized := normalizeAccountTitle(title)
			if normalized == "" {
				matchAll = true
				break
			}
			targetAccounts[normalized] = true
		}
	} else {
		matchAll = true
	}

	isMatch := func(accountTitle string) bool {
		if matchAll {
			return true
		}
		normalized := normalizeAccountTitle(accountTitle)
		for target := range targetAccounts {
			if normalized == target || strings.HasPrefix(normalized, target+":") {
				return true
			}
		}
		return false
	}

	type voucherWithDate struct {
		v    *voucher.Voucher
		date time.Time
	}
	var allVouchers []voucherWithDate
	for _, j := range a.Book.Journals {
		for _, v := range j.Vouchers {
			t, err := time.ParseInLocation(time.RFC3339, v.Date, time.Local)
			if err != nil {
				t, _ = time.Parse("2006-01-02", v.Date)
			}
			allVouchers = append(allVouchers, voucherWithDate{v: v, date: t})
		}
	}

	sort.Slice(allVouchers, func(i, j int) bool {
		if !allVouchers[i].date.Equal(allVouchers[j].date) {
			return allVouchers[i].date.Before(allVouchers[j].date)
		}
		return allVouchers[i].v.Id < allVouchers[j].v.Id
	})

	var registers []*report.AccountRegister
	runningBalances := make(map[string]float64)

	for _, vd := range allVouchers {
		v := vd.v
		for _, e := range v.Entries {
			if isMatch(e.Account) {
				q, _ := strconv.ParseFloat(e.Amount.Quantity, 64)
				runningBalances[e.Amount.Currency] = roundAmount(runningBalances[e.Amount.Currency] + q)

				var balances []*amount.Amount
				currencies := make([]string, 0, len(runningBalances))
				for c := range runningBalances {
					currencies = append(currencies, c)
				}
				sort.Strings(currencies)
				for _, c := range currencies {
					balances = append(balances, &amount.Amount{
						Quantity: formatAmount(runningBalances[c]),
						Currency: c,
					})
				}

				registers = append(registers, &report.AccountRegister{
					Title:    e.Account,
					Amounts:  []*amount.Amount{e.Amount},
					Balances: balances,
					Voucher:  v,
				})
			}
		}
	}

	return &AccountRegisterReport{Register: registers}, nil
}
