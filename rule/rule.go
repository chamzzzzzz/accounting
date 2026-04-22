package rule

import (
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/voucher"
)

type Rule struct {
	Name        string                         `json:"name,omitempty"`
	Description string                         `json:"description,omitempty"`
	Catalog     string                         `json:"catalog,omitempty"`
	Priority    int                            `json:"priority,omitempty"`
	Continue    bool                           `json:"continue,omitempty"`
	Match       *sourcedocument.SourceDocument `json:"match,omitempty"`
	Prepare     *sourcedocument.SourceDocument `json:"prepare,omitempty"`
	Specs       []*Rule                        `json:"specs,omitempty"`
	Voucher     *voucher.Voucher               `json:"voucher,omitempty"`
}
