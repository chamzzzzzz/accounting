package wechatpay

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
)

const (
	DriverName = "wechatpay"
	ParamName  = "v1"
)

type Analyzer struct {
}

func (analyzer *Analyzer) Analyze(sourcefrom any) (*types.SourceDocument, error) {
	sourcedocument := &types.SourceDocument{}
	return sourcedocument, nil
}

func (analyzer *Analyzer) Driver() driver.Driver {
	return &Driver{}
}

type Driver struct {
}

func (driver *Driver) Open(paramName string) (driver.Analyzer, error) {
	return &Analyzer{}, nil
}

func init() {
	driver.Register(DriverName, &Driver{})
}
