package journal

import "github.com/chamzzzzzz/accounting/voucher"

type Journal struct {
	Date     string             `json:"date,omitempty"`
	Vouchers []*voucher.Voucher `json:"vouchers,omitempty"`
}
