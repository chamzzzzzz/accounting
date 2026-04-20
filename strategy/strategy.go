package strategy

import (
	"context"

	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/voucher"
)

type Strategy interface {
	PrepareVoucher(ctx context.Context, book *book.Book, sourcedocument *sourcedocument.SourceDocument) ([]*voucher.Voucher, error)
}
