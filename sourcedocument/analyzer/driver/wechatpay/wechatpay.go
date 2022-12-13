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
	source := sourcefrom.(*types.Source)
	for _, keyword := range []string{"当前状态", "商户全称", "支付时间", "支付方式", "交易单号", "商户单号", "商品"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &types.SourceDocument{}
	sourcedocument.Source = source
	sourcedocument.Name = "WechatPay"
	sourcedocument.Class = "Expensive"
	sourcedocument.From = source.HorizontalKeyValueText("支付方式")
	sourcedocument.Description = source.HorizontalKeyValueText("商品")
	sourcedocument.Merchant = source.HorizontalKeyValueText("商户全称")
	sourcedocument.OrderNumber = source.HorizontalKeyValueText("交易单号")
	sourcedocument.Date = source.HorizontalKeyValueText("支付时间")
	if item := source.First(source.TextEquals("原价")); item != nil {
		index := item.Index
		if item := source.Item(index - 1); item != nil {
			sourcedocument.Amount = item.Text
		}
	} else if item := source.First(source.TextEquals("当前状态")); item != nil {
		index := item.Index
		if item := source.Item(index - 1); item != nil {
			sourcedocument.Amount = item.Text
		}
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
