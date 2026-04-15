package ledger

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/voucher"
)

type (
	Account = account.Account
	Amount  = amount.Amount
	Book    = book.Book
	Entry   = voucher.Entry
	Info    = book.Info
	Journal = journal.Journal
	Voucher = voucher.Voucher
)

type Provider struct {
	Dir string
}

func (p *Provider) Load(ctx context.Context, id string) (*Book, error) {
	path := filepath.Join(p.Dir, id)

	info, err := readInfo(path)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	accounts, err := readAccounts(path)
	if err != nil {
		return nil, err
	}

	journals, err := readJournals(path)
	if err != nil {
		return nil, err
	}

	book := &Book{
		Id:       id,
		Info:     info,
		Accounts: accounts,
		Journals: journals,
	}
	return book, nil
}

func (p *Provider) Save(ctx context.Context, book *Book) error {
	path := filepath.Join(p.Dir, book.Id)

	info := book.Info
	if info == nil {
		info = &Info{}
	}
	accounts := book.Accounts
	if accounts == nil {
		accounts = []*Account{}
	}

	if err := checkAccounts(accounts); err != nil {
		return err
	}

	if err := writeInfo(path, info); err != nil {
		return err
	}
	if err := writeAccounts(path, accounts); err != nil {
		return err
	}
	if err := writeJournals(path, book.Journals); err != nil {
		return err
	}
	return nil
}

func readInfo(path string) (*Info, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("book path is not a directory")
	}
	return &Info{}, nil
}

func writeInfo(path string, info *Info) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	_ = info
	return nil
}

func readAccounts(path string) ([]*Account, error) {
	root := filepath.Join(path, "ACCOUNT")
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("ACCOUNT is not a directory")
	}

	var accounts []*Account
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".ledger" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		catalog := strings.TrimSuffix(filepath.ToSlash(rel), ".ledger")
		for _, line := range strings.Split(string(b), "\n") {
			title, ok := strings.CutPrefix(strings.TrimSpace(line), "account ")
			if !ok {
				continue
			}

			title = strings.TrimSpace(title)
			if title == "" {
				continue
			}

			accounts = append(accounts, &Account{Catalog: catalog, Title: title})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return accounts, nil
}

func writeAccounts(path string, accounts []*Account) error {
	root := filepath.Join(path, "ACCOUNT")
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}

	catalogs, err := catalogAccounts(accounts)
	if err != nil {
		return err
	}
	for catalog, accounts := range catalogs {
		name := filepath.Join(root, catalog+".ledger")
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return err
		}

		var sb strings.Builder
		for _, account := range accounts {
			fmt.Fprintf(&sb, "account %s\n", account.Title)
		}
		if err := os.WriteFile(name, []byte(sb.String()), 0644); err != nil {
			return err
		}
	}
	return nil
}

func catalogAccounts(accounts []*Account) (map[string][]*Account, error) {
	catalogs := make(map[string][]*account.Account)
	for _, account := range accounts {
		if account.Catalog == "" {
			return nil, errors.New("no account catalog")
		}
		catalogs[account.Catalog] = append(catalogs[account.Catalog], account)
	}
	for k1 := range catalogs {
		for k2 := range catalogs {
			if k1 != k2 && (strings.HasPrefix(k1, k2) || strings.HasPrefix(k2, k1)) {
				return nil, errors.New("account catalog conflict: " + k1 + " and " + k2)
			}
		}
	}
	return catalogs, nil
}

func checkAccounts(accounts []*Account) error {
	if _, err := catalogAccounts(accounts); err != nil {
		return err
	}
	return nil
}

func readJournals(path string) ([]*Journal, error) {
	root := filepath.Join(path, "TRANSACTION")
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("TRANSACTION is not a directory")
	}

	var journals []*Journal
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".ledger" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 3 {
			return nil
		}

		date := parts[0] + "-" + parts[1]
		catalog := strings.TrimSuffix(parts[2], ".ledger")

		journal, err := readJournalFile(path)
		if err != nil {
			return err
		}
		journal.Date = date
		journal.Catalog = catalog

		journals = append(journals, journal)
		return nil
	}); err != nil {
		return nil, err
	}
	return journals, nil
}

func writeJournals(path string, journals []*Journal) error {
	root := filepath.Join(path, "TRANSACTION")
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	for _, journal := range journals {
		journal, err := formatJournal(journal)
		if err != nil {
			return err
		}
		if journal.Catalog == "" {
			return errors.New("no journal catalog")
		}
		if journal.Date == "" {
			return errors.New("no journal date")
		}
		t, err := time.ParseInLocation("2006-01", journal.Date, time.Local)
		if err != nil {
			return errors.New("invalid journal date: " + journal.Date)
		}
		year := t.Format("2006")
		month := t.Format("01")
		name := filepath.Join(root, year, month, journal.Catalog+".ledger")
		if err := writeJournalFile(name, journal); err != nil {
			return err
		}
	}
	return nil
}

func readJournalFile(path string) (*Journal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var vouchers []*Voucher
	var current *Voucher
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			if current != nil {
				vouchers = append(vouchers, current)
				current = nil
			}
			continue

		case strings.HasPrefix(trimmed, ";"):
			if current == nil {
				continue
			}

			comment := strings.TrimSpace(trimmed[1:])
			if after, ok := strings.CutPrefix(comment, "date:"); ok {
				if t, err := time.Parse(time.RFC3339, strings.TrimSpace(after)); err == nil {
					current.Date = t.Format(time.RFC3339)
					continue
				}
			}
			if current.Comment != "" {
				current.Comment += "\n"
			}
			current.Comment += comment
			continue

		case len(line) > 0 && (line[0] == ' ' || line[0] == '\t'):
			if current == nil {
				continue
			}

			entryLine := trimmed
			var comment string
			if idx := strings.Index(entryLine, ";"); idx >= 0 {
				comment = strings.TrimSpace(entryLine[idx+1:])
				entryLine = entryLine[:idx]
			}

			fields := strings.Fields(entryLine)
			if len(fields) < 3 {
				return nil, fmt.Errorf("line %d: invalid posting: '%s'. Expected 'Account Amount Currency'", i+1, trimmed)
			}

			quantity := fields[len(fields)-2]
			if quantity == "" {
				return nil, fmt.Errorf("line %d: invalid amount '%s'", i+1, quantity)
			}
			first := quantity[0]
			if (first != '-' && first != '+' && first != '.' && (first < '0' || first > '9')) || !strings.ContainsAny(quantity, "0123456789") {
				return nil, fmt.Errorf("line %d: invalid amount '%s'", i+1, quantity)
			}

			current.Entries = append(current.Entries, &Entry{
				Account: strings.Join(fields[:len(fields)-2], " "),
				Amount: &Amount{
					Quantity: quantity,
					Currency: fields[len(fields)-1],
				},
				Comment: strings.ReplaceAll(comment, "\\n", "\n"),
			})
			continue

		default:
			if current != nil {
				vouchers = append(vouchers, current)
			}

			header := trimmed
			var comment string
			if idx := strings.Index(header, ";"); idx >= 0 {
				comment = strings.TrimSpace(header[idx+1:])
				header = strings.TrimSpace(header[:idx])
			}

			parts := strings.SplitN(header, " ", 2)
			t, err := time.Parse("2006/01/02", parts[0])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid date: %s", i+1, parts[0])
			}

			current = &Voucher{Date: t.Format(time.RFC3339)}
			if len(parts) > 1 {
				current.Description = strings.TrimSpace(parts[1])
			}
			if comment == "" {
				continue
			}
			if after, ok := strings.CutPrefix(comment, "date:"); ok {
				if t, err := time.Parse(time.RFC3339, strings.TrimSpace(after)); err == nil {
					current.Date = t.Format(time.RFC3339)
					continue
				}
			}
			current.Comment = comment
		}
	}
	if current != nil {
		vouchers = append(vouchers, current)
	}

	return &Journal{Vouchers: vouchers}, nil
}

func writeJournalFile(path string, journal *Journal) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	var sb strings.Builder
	for i, voucher := range journal.Vouchers {
		if i > 0 {
			sb.WriteString("\n")
		}

		t, _ := time.Parse(time.RFC3339, voucher.Date)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02", voucher.Date)
		}
		if voucher.Description != "" {
			fmt.Fprintf(&sb, "%s %s\n", t.Format("2006/01/02"), voucher.Description)
		} else {
			fmt.Fprintf(&sb, "%s\n", t.Format("2006/01/02"))
		}
		if !t.IsZero() && (t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0) {
			fmt.Fprintf(&sb, "  ;date:%s\n", voucher.Date)
		}
		for _, c := range strings.Split(voucher.Comment, "\n") {
			if c != "" {
				fmt.Fprintf(&sb, "  ;%s\n", c)
			}
		}
		for _, entry := range voucher.Entries {
			s := "  " + entry.Account
			if entry.Amount != nil {
				s += "  " + entry.Amount.Quantity
				if entry.Amount.Currency != "" {
					s += " " + entry.Amount.Currency
				}
			}
			if entry.Comment != "" {
				s += " ;" + strings.ReplaceAll(entry.Comment, "\n", "\\n")
			}
			sb.WriteString(s + "\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func formatJournal(journal *Journal) (*Journal, error) {
	if len(journal.Vouchers) <= 1 {
		return journal, nil
	}

	type e struct {
		voucher *voucher.Voucher
		date    time.Time
	}

	x := make([]e, 0, len(journal.Vouchers))
	for _, v := range journal.Vouchers {
		date, err := time.ParseInLocation(time.RFC3339, v.Date, time.Local)
		if err != nil {
			return nil, errors.New("invalid voucher date: " + v.Date)
		}
		x = append(x, e{voucher: v, date: date})
	}

	sort.SliceStable(x, func(i, j int) bool {
		return x[i].date.Before(x[j].date)
	})

	j := &Journal{
		Date:    journal.Date,
		Catalog: journal.Catalog,
	}
	for _, e := range x {
		j.Vouchers = append(j.Vouchers, e.voucher)
	}
	return j, nil
}
