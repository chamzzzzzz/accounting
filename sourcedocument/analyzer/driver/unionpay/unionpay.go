package unionpay

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
)

const (
	DriverName = "unionpay"
	ParamName  = "v1"
)

type Analyzer struct {
}

func (analyzer *Analyzer) Analyze(sourcefrom any) (*types.SourceDocument, error) {
	source := sourcefrom.(*types.Source)
	for _, keyword := range []string{"还款详情", "订单类型", "还款卡号", "付款卡号", "订单编号", "创建时间"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &types.SourceDocument{}
	sourcedocument.Source = source
	sourcedocument.Name = "UnionPay"
	sourcedocument.Class = "CreditCardRepayment"
	sourcedocument.From = source.HorizontalKeyValueText("付款卡号")
	sourcedocument.To = source.HorizontalKeyValueText("还款卡号")
	sourcedocument.Amount = source.ColonJoinedKeyValueText("还款金额")
	sourcedocument.OrderNumber = source.HorizontalKeyValueText("订单编号")
	sourcedocument.Description = source.HorizontalKeyValueText("订单类型")
	sourcedocument.Date = source.HorizontalKeyValueText("创建时间")
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
