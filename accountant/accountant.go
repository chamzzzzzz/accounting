package accountant

import (
	"errors"
	"time"

	"github.com/chamzzzzzz/accounting/account"
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
	Parameters            = report.ReportParameters
	AccountBalanceReport  = report.AccountBalanceReport
	AccountRegisterReport = report.AccountRegisterReport
)

var (
	ErrNoBook = errors.New("no book")
)

type Accountant struct {
	Scanners   []scanner.Scanner
	Processors []processor.Processor
	Book       *book.Book
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

func (a *Accountant) AddVoucher(voucher *Voucher) error {
	if a.Book == nil {
		return ErrNoBook
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

func (a *Accountant) ReportAccountBalance(parameters *Parameters) (*AccountBalanceReport, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	return nil, nil
}

func (a *Accountant) ReportAccountRegister(parameters *Parameters) (*AccountRegisterReport, error) {
	if a.Book == nil {
		return nil, ErrNoBook
	}
	return nil, nil
}
