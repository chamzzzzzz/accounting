package voucher

import (
	"github.com/chamzzzzzz/accounting/amount"
)

type Entry struct {
	Account string         `json:"account,omitempty"`
	Amount  *amount.Amount `json:"amount,omitempty"`
}

type Voucher struct {
	Id          string   `json:"id,omitempty"`
	Date        string   `json:"date,omitempty"`
	Entries     []*Entry `json:"entries,omitempty"`
	Description string   `json:"description,omitempty"`
}
