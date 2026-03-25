package processor

import (
	"context"

	"github.com/chamzzzzzz/accounting/sourcedocument"
)

type Processor interface {
	Process(ctx context.Context, source *sourcedocument.SourceDocument) (*sourcedocument.SourceDocument, error)
}
