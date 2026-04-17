package rule

import (
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/voucher"
)

type Rule struct {
	Name           string                         `json:"name,omitempty"`
	Description    string                         `json:"description,omitempty"`
	Catalog        string                         `json:"catalog,omitempty"`
	SourceDocument *sourcedocument.SourceDocument `json:"source_document,omitempty"`
	Voucher        *voucher.Voucher               `json:"voucher,omitempty"`
}
