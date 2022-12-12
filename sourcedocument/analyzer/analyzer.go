package analyzer

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"

	_ "github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver/alipay"
	_ "github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver/unionpay"
	_ "github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver/wechatpay"
)

type Analyzer struct {
	da driver.Analyzer
}

func (analyzer *Analyzer) Analyze(sourcefrom any) (*types.SourceDocument, error) {
	return analyzer.da.Analyze(sourcefrom)
}

func (analyzer *Analyzer) Driver() driver.Driver {
	return analyzer.da.Driver()
}

func Open(driverName, paramName string) (*Analyzer, error) {
	analyzer, err := driver.Open(driverName, paramName)
	if err != nil {
		return nil, err
	}
	return &Analyzer{analyzer}, nil
}

func Drivers() []string {
	return driver.Drivers()
}
