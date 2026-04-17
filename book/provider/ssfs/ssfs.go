package ssfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/rule"
	"github.com/chamzzzzzz/accounting/voucher"
)

type (
	Account = account.Account
	Journal = journal.Journal
	Rule    = rule.Rule
	Info    = book.Info
	Book    = book.Book
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
	rules, err := readRules(path)
	if err != nil {
		return nil, err
	}

	book := &Book{
		Id:       id,
		Info:     info,
		Accounts: accounts,
		Journals: journals,
		Rules:    rules,
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
	if err := checkRules(book.Rules); err != nil {
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
	if err := writeRules(path, book.Rules); err != nil {
		return err
	}
	return nil
}

func readInfo(path string) (*Info, error) {
	name := filepath.Join(path, "INFO")
	info := &Info{}
	if err := read(name, info); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return info, nil
}

func writeInfo(path string, info *Info) error {
	name := filepath.Join(path, "INFO")
	return write(name, info)
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
		var list []*Account
		if err := read(path, &list); err != nil {
			return err
		}
		accounts = append(accounts, list...)
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
		name := filepath.Join(root, catalog)
		if err := write(name, accounts); err != nil {
			return err
		}
	}
	return nil
}

func catalogAccounts(accounts []*Account) (map[string][]*Account, error) {
	catalogs := make(map[string][]*Account)
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
	root := filepath.Join(path, "JOURNAL")
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("JOURNAL is not a directory")
	}

	var journals []*Journal
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		journal := &Journal{}
		if err := read(path, journal); err != nil {
			return err
		}
		journals = append(journals, journal)
		return nil
	}); err != nil {
		return nil, err
	}
	return journals, nil
}

func writeJournals(path string, journals []*Journal) error {
	root := filepath.Join(path, "JOURNAL")
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
		name := filepath.Join(root, year, month, journal.Catalog)
		if err := write(name, journal); err != nil {
			return err
		}
	}
	return nil
}

func readRules(path string) ([]*Rule, error) {
	root := filepath.Join(path, "RULE")
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("RULE is not a directory")
	}

	var rules []*Rule
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		r := &Rule{}
		if err := read(path, r); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		r.Catalog = filepath.ToSlash(rel)
		rules = append(rules, r)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := checkRules(rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func writeRules(path string, rules []*Rule) error {
	root := filepath.Join(path, "RULE")
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	if err := checkRules(rules); err != nil {
		return err
	}
	for _, rule := range rules {
		name := filepath.Join(root, filepath.FromSlash(rule.Catalog))
		if err := write(name, rule); err != nil {
			return err
		}
	}
	return nil
}

func checkRules(rules []*Rule) error {
	seen := make(map[string]bool)
	for _, rule := range rules {
		if rule.Catalog == "" {
			return errors.New("no rule catalog")
		}
		if seen[rule.Catalog] {
			return fmt.Errorf("rule catalog conflict: %s", rule.Catalog)
		}
		seen[rule.Catalog] = true
	}
	for c1 := range seen {
		for c2 := range seen {
			if c1 == c2 {
				continue
			}
			if strings.HasPrefix(c1, c2+"/") || strings.HasPrefix(c2, c1+"/") {
				return fmt.Errorf("rule catalog conflict: %s and %s", c1, c2)
			}
		}
	}
	return nil
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

func read(name string, v any) error {
	b, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func write(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	return os.WriteFile(name, b, 0644)
}
