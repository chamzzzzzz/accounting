package alipay

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/analyzer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
)

const (
	DriverName = "alipay"
	ParamName  = "v1"
)

type Analyzer struct {
}

func (analyzer *Analyzer) Analyze(sourcefrom any) (*types.SourceDocument, error) {
	source := sourcefrom.(*types.Source)
	for _, keyword := range []string{"账单详情", "付款方式", "支付宝积分", "商品说明", "创建时间"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &types.SourceDocument{}
	sourcedocument.Source = source
	sourcedocument.Name = "Alipay"
	sourcedocument.Class = "Expensive"
	sourcedocument.From = source.HorizontalKeyValueText("付款方式")
	sourcedocument.Description = source.HorizontalKeyValueText("商品说明")
	sourcedocument.Merchant = source.HorizontalKeyValueText("收款方全称")
	sourcedocument.OrderNumber = source.HorizontalKeyValueText("订单号")
	sourcedocument.Date = source.HorizontalKeyValueText("创建时间")
	if item := source.First(source.TextEquals("交易成功")); item != nil {
		index := item.Index
		if item := source.Item(index - 1); item != nil {
			sourcedocument.Amount = item.Text
		}
		if item := source.Item(index - 2); item != nil {
			if sourcedocument.Merchant == "" {
				sourcedocument.Merchant = item.Text
			}
		}
	}
	if item, _ := source.HorizontalKeyValue("支付时间"); item != nil {
		sourcedocument.Date = item.Text
	}
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
