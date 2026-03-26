package scanner

import (
	"context"

	"github.com/chamzzzzzz/accounting/sourcedocument"
)

type Document struct {
	Path string
}

type Scanner interface {
	Scan(ctx context.Context, document *Document) (*sourcedocument.SourceDocument, error)
}
