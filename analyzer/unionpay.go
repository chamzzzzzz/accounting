package analyzer

type UnionPay struct {
}

func (u *UnionPay) Analyze(source *Source) (*SourceDocument, error) {
	for _, keyword := range []string{"还款详情", "订单类型", "还款卡号", "付款卡号", "订单编号", "创建时间"} {
		if len(source.TextEquals(keyword)) == 0 {
			return nil, nil
		}
	}
	sourcedocument := &SourceDocument{}
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
