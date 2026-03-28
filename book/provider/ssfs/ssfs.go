package ssfs

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/journal"
)

type (
	Account = account.Account
	Journal = journal.Journal
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
	name := filepath.Join(path, "ACCOUNTS")
	var accounts []*Account
	if err := read(name, &accounts); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return accounts, nil
}

func writeAccounts(path string, accounts []*Account) error {
	name := filepath.Join(path, "ACCOUNTS")
	return write(name, accounts)
}

func readJournals(path string) ([]*Journal, error) {
	root := filepath.Join(path, "JOURNALS")
	fi, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("JOURNALS is not a directory")
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
	root := filepath.Join(path, "JOURNALS")
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	for _, journal := range journals {
		if journal.Name == "" {
			return errors.New("no journal name")
		}
		if journal.Date == "" {
			return errors.New("no journal date")
		}
		t, err := time.ParseInLocation(time.RFC3339, journal.Date, time.Local)
		if err != nil {
			return errors.New("invalid journal date: " + journal.Date)
		}
		year := t.Format("2006")
		month := t.Format("01")
		name := filepath.Join(root, year, month, journal.Name)
		if err := write(name, journal); err != nil {
			return err
		}
	}
	return nil
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
