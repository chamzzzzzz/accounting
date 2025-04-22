package accounting

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Account struct {
	ID          string `json:"id,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Amount struct {
	Quantity string `json:"quantity,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type Entry struct {
	Account string  `json:"account,omitempty"`
	Amount  *Amount `json:"amount,omitempty"`
	Note    string  `json:"note,omitempty"`
}

type Voucher struct {
	ID          string   `json:"id,omitempty"`
	Date        string   `json:"date,omitempty"`
	Entries     []*Entry `json:"entries,omitempty"`
	Description string   `json:"description,omitempty"`
	Note        string   `json:"note,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type Journal struct {
	Date     string     `json:"date,omitempty"`
	Vouchers []*Voucher `json:"vouchers,omitempty"`
}

type Book struct {
	Accounts []*Account `json:"accounts,omitempty"`
	Journals []*Journal `json:"journals,omitempty"`
}

type BookKeeper interface {
	SetAccounts(accounts []*Account) error
	GetAccounts(names []string) ([]*Account, error)
	SetJournal(journal *Journal) error
	GetJournal(date string) (*Journal, error)
	GetJournals(voucher bool) ([]*Journal, error)
}

type FileBookKeeper struct {
	File string
}

func (fbk *FileBookKeeper) SetAccounts(accounts []*Account) error {
	book, err := fbk.ReadBook()
	if err != nil {
		return err
	}
	if book == nil {
		book = &Book{}
	}
	for _, account := range accounts {
		new := true
		for _, a := range book.Accounts {
			if a.Name == account.Name {
				*a = *account
				new = false
			}
		}
		if new {
			book.Accounts = append(book.Accounts, account)
		}
	}
	return fbk.WriteBook(book)
}

func (fbk *FileBookKeeper) GetAccounts(names []string) ([]*Account, error) {
	book, err := fbk.ReadBook()
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, nil
	}
	var accounts []*Account
	if len(names) > 0 {
		for _, a := range book.Accounts {
			if slices.Contains(names, a.Name) {
				accounts = append(accounts, a)
			}
		}
	} else {
		accounts = book.Accounts
	}
	return accounts, nil
}

func (fbk *FileBookKeeper) SetJournal(journal *Journal) error {
	book, err := fbk.ReadBook()
	if err != nil {
		return err
	}
	if book == nil {
		book = &Book{}
	}
	new := true
	for _, j := range book.Journals {
		if j.Date == journal.Date {
			*j = *journal
			new = false
		}
	}
	if new {
		book.Journals = append(book.Journals, journal)
	}
	return fbk.WriteBook(book)
}

func (fbk *FileBookKeeper) GetJournal(date string) (*Journal, error) {
	book, err := fbk.ReadBook()
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, nil
	}
	for _, j := range book.Journals {
		if j.Date == date {
			return j, nil
		}
	}
	return nil, nil
}

func (fbk *FileBookKeeper) GetJournals(voucher bool) ([]*Journal, error) {
	book, err := fbk.ReadBook()
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, nil
	}
	var journals []*Journal
	if voucher {
		journals = book.Journals
	} else {
		for _, j := range book.Journals {
			journals = append(journals, &Journal{Date: j.Date})
		}
	}
	return journals, nil
}

func (fbk *FileBookKeeper) ReadBook() (*Book, error) {
	b, err := os.ReadFile(fbk.File)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	book := &Book{}
	err = json.Unmarshal(b, book)
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (fbk *FileBookKeeper) WriteBook(book *Book) error {
	b, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(fbk.File, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

type DirBookKeeper struct {
	Dir string
}

func (dbk *DirBookKeeper) SetAccounts(accounts []*Account) error {
	v, err := dbk.readAccounts()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		new := true
		for _, a := range v {
			if a.Name == account.Name {
				*a = *account
				new = false
			}
		}
		if new {
			v = append(v, account)
		}
	}
	return dbk.writeAccounts(v)
}

func (dbk *DirBookKeeper) GetAccounts(names []string) ([]*Account, error) {
	v, err := dbk.readAccounts()
	if err != nil {
		return nil, err
	}
	var accounts []*Account
	if len(names) > 0 {
		for _, a := range v {
			if slices.Contains(names, a.Name) {
				accounts = append(accounts, a)
			}
		}
	} else {
		accounts = v
	}
	return accounts, nil
}

func (dbk *DirBookKeeper) SetJournal(journal *Journal) error {
	return dbk.writeJournal(journal)
}

func (dbk *DirBookKeeper) GetJournal(date string) (*Journal, error) {
	return dbk.readJournal(date)
}

func (dbk *DirBookKeeper) GetJournals(voucher bool) ([]*Journal, error) {
	dir := filepath.Join(dbk.Dir, "journal")
	var journals []*Journal
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		date := strings.ReplaceAll(filepath.ToSlash(rel), "/", "-")
		date = strings.ReplaceAll(date, ".json", "")
		if voucher {
			j, err := dbk.readJournal(date)
			if err != nil {
				return err
			}
			journals = append(journals, j)
		} else {
			journals = append(journals, &Journal{Date: date})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return journals, nil
}

func (dbk *DirBookKeeper) readAccounts() ([]*Account, error) {
	file := filepath.Join(dbk.Dir, "account.json")
	b, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	accounts := []*Account{}
	err = json.Unmarshal(b, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (dbk *DirBookKeeper) writeAccounts(accounts []*Account) error {
	file := filepath.Join(dbk.Dir, "account.json")
	b, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	return os.WriteFile(file, b, 0644)
}

func (dbk *DirBookKeeper) readJournal(date string) (*Journal, error) {
	t, err := time.ParseInLocation("2006-01", date, time.Local)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(dbk.Dir, "journal", fmt.Sprintf("%d", t.Year()), fmt.Sprintf("%02d.json", t.Month()))
	b, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	journal := &Journal{}
	if err = json.Unmarshal(b, journal); err != nil {
		return nil, err
	}
	return journal, nil
}

func (dbk *DirBookKeeper) writeJournal(journal *Journal) error {
	t, err := time.ParseInLocation("2006-01", journal.Date, time.Local)
	if err != nil {
		return err
	}
	file := filepath.Join(dbk.Dir, "journal", fmt.Sprintf("%d", t.Year()), fmt.Sprintf("%02d.json", t.Month()))
	b, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	return os.WriteFile(file, b, 0644)
}
