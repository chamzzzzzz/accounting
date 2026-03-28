package journal

import "github.com/chamzzzzzz/accounting/voucher"

type Journal struct {
	Name     string             `json:"name,omitempty"`
	Date     string             `json:"date,omitempty"`
	Vouchers []*voucher.Voucher `json:"vouchers,omitempty"`
}
