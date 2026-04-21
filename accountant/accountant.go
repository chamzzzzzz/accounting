package accountant

import (
	"context"
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
	"github.com/chamzzzzzz/accounting/rule"
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/sourcedocument/processor"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/strategy"
	"github.com/chamzzzzzz/accounting/voucher"
)

type (
	Account               = account.Account
	Rule                  = rule.Rule
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
	Strategy   strategy.Strategy
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
	if account.Title == "" {
		return errors.New("no account title")
	}
	if account.Catalog == "" {
		return errors.New("no account catalog")
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

func (a *Accountant) AddRule(rule *Rule) error {
	if a.Book == nil {
		return ErrNoBook
	}
	if rule == nil {
		return errors.New("no rule")
	}
	if rule.Catalog == "" {
		return errors.New("no rule catalog")
	}
	if r, _ := a.GetRule(rule.Catalog); r != nil {
		return errors.New("rule exists: " + rule.Catalog)
	}
	a.Book.Rules = append(a.Book.Rules, rule)
	return nil
}

func (a *Accountant) DelRule(catalog string) error {
	if a.Book == nil {
		return ErrNoBook
	}
	if catalog == "" {
		return errors.New("no rule catalog")
	}
	for i, rule := range a.Book.Rules {
		if rule.Catalog == catalog {
			a.Book.Rules = append(a.Book.Rules[:i], a.Book.Rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func (a *Accountant) GetRule(catalog string) (*Rule, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	if catalog == "" {
		return nil, errors.New("no rule catalog")
	}
	for _, rule := range a.Book.Rules {
		if rule.Catalog == catalog {
			return rule, nil
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

func (a *Accountant) PrepareVoucher(ctx context.Context, source *sourcedocument.SourceDocument) ([]*Voucher, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	if source == nil {
		return nil, errors.New("no source document")
	}

	if a.Strategy == nil {
		return nil, errors.New("no strategy")
	}

	return a.Strategy.PrepareVoucher(ctx, a.Book, source)
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

	if parameters != nil && len(parameters.Titles) > 0 {
		titles := make([]string, 0, len(parameters.Titles))
		for _, title := range parameters.Titles {
			normalized := normalizeAccountTitle(title)
			if normalized != "" {
				titles = append(titles, normalized)
			}
		}
		sort.Strings(titles)

		unique := make([]string, 0, len(titles))
		for i := 0; i < len(titles); i++ {
			isSub := false
			for j := 0; j < len(titles); j++ {
				if i == j {
					continue
				}
				if titles[i] == titles[j] && i > j {
					isSub = true
					break
				}
				if strings.HasPrefix(titles[i], titles[j]+":") {
					isSub = true
					break
				}
			}
			if !isSub {
				unique = append(unique, titles[i])
			}
		}

		balance := []*report.AccountBalance{}
		for _, title := range unique {
			if node, ok := nodes[title]; ok {
				balance = append(balance, node)
			}
		}
		return &report.AccountBalanceReport{Balance: balance}, nil
	}

	balance := []*report.AccountBalance{}
	for _, root := range rootOrder {
		if node, ok := nodes[root]; ok {
			balance = append(balance, node)
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
				t, _ = time.ParseInLocation("2006-01-02", v.Date, time.Local)
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
