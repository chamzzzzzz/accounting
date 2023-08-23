package analyzer

type WechatPay struct {
}

func (w *WechatPay) Analyze(source *Source) (*SourceDocument, error) {
	for _, keyword := range []string{"当前状态", "商户全称", "支付时间", "支付方式", "交易单号", "商户单号", "商品"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &SourceDocument{}
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
