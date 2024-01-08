package accounting

type VoucherEntry struct {
	Account string `json:"account"`
	Amount  string `json:"amount"`
}

type Voucher struct {
	SourceDocument *SourceDocument `json:"sourceDocument"`
	From           []*VoucherEntry `json:"from"`
	To             []*VoucherEntry `json:"to"`
}
