package book

import (
	"context"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/rule"
)

type Info struct {
	Title string `json:"title,omitempty"`
}

type Book struct {
	Id       string             `json:"-"`
	Info     *Info              `json:"info,omitempty"`
	Accounts []*account.Account `json:"accounts,omitempty"`
	Journals []*journal.Journal `json:"journals,omitempty"`
	Rules    []*rule.Rule       `json:"rules,omitempty"`
}

type Provider interface {
	Load(ctx context.Context, id string) (*Book, error)
	Save(ctx context.Context, book *Book) error
}
