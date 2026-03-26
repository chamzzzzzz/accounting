package book

import (
	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/journal"
)

type Book struct {
	Coa      account.ChartOfAccounts `json:"coa,omitempty"`
	Journals []*journal.Journal      `json:"journals,omitempty"`
}
