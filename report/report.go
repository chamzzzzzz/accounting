package report

import (
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/voucher"
)

type ReportParameters struct {
	Titles []string `json:"titles,omitempty"`
}

type AccountBalance struct {
	Title    string            `json:"title,omitempty"`
	Amounts  []*amount.Amount  `json:"amounts,omitempty"`
	Children []*AccountBalance `json:"children,omitempty"`
}

type AccountBalanceReport struct {
	Balance []*AccountBalance `json:"balance,omitempty"`
}

type AccountRegister struct {
	Title    string           `json:"title,omitempty"`
	Amounts  []*amount.Amount `json:"amounts,omitempty"`
	Balances []*amount.Amount `json:"balances,omitempty"`
	Voucher  *voucher.Voucher `json:"voucher,omitempty"`
}

type AccountRegisterReport struct {
	Register []*AccountRegister `json:"register,omitempty"`
}
