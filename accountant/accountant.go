package accountant

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/processor"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
)

type Accountant struct {
	Scanners   []scanner.Scanner
	Processors []processor.Processor
}
