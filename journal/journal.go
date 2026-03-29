package journal

import "github.com/chamzzzzzz/accounting/voucher"

type Journal struct {
	Date     string             `json:"date,omitempty"`
	Catalog  string             `json:"catalog,omitempty"`
	Vouchers []*voucher.Voucher `json:"vouchers,omitempty"`
}
