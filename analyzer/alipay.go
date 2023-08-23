package analyzer

type Alipay struct {
}

func (a *Alipay) Analyze(source *Source) (*SourceDocument, error) {
	for _, keyword := range []string{"账单详情", "付款方式", "商品说明", "支付时间"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &SourceDocument{}
	sourcedocument.Source = source
	sourcedocument.Name = "Alipay"
	sourcedocument.Class = "Expensive"
	sourcedocument.From = source.HorizontalKeyValueText("付款方式")
	sourcedocument.Description = source.HorizontalKeyValueText("商品说明")
	sourcedocument.Merchant = source.HorizontalKeyValueText("收款方全称")
	sourcedocument.OrderNumber = source.HorizontalKeyValueText("订单号")
	sourcedocument.Date = source.HorizontalKeyValueText("支付时间")
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
