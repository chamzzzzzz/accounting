package recognizer

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/recognizer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"

	_ "github.com/chamzzzzzz/accounting/sourcedocument/recognizer/driver/gocr"
)

type Recognizer struct {
	dr driver.Recognizer
}

func (recognizer *Recognizer) Recognize(sourcefrom any) (*types.Source, error) {
	return recognizer.dr.Recognize(sourcefrom)
}

func (recognizer *Recognizer) Driver() driver.Driver {
	return recognizer.dr.Driver()
}

func Open(driverName, paramName string) (*Recognizer, error) {
	recognizer, err := driver.Open(driverName, paramName)
	if err != nil {
		return nil, err
	}
	return &Recognizer{recognizer}, nil
}

func Drivers() []string {
	return driver.Drivers()
}
