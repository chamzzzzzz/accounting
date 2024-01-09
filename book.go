package accounting

type SourceDocument struct {
	Source      *Source `json:"source"`
	Name        string  `json:"name"`
	Class       string  `json:"class"`
	From        string  `json:"from"`
	To          string  `json:"to"`
	Amount      string  `json:"amount"`
	OrderNumber string  `json:"orderNumber"`
	Merchant    string  `json:"merchant"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

type Account struct {
	Name  string   `json:"name"`
	Alias []string `json:"alias"`
}

type VoucherEntry struct {
	Account string `json:"account"`
	Amount  string `json:"amount"`
}

type Voucher struct {
	SourceDocument *SourceDocument `json:"sourceDocument"`
	From           []*VoucherEntry `json:"from"`
	To             []*VoucherEntry `json:"to"`
}

type Journal struct {
	Vouchers []*Voucher `json:"vouchers"`
}

type Book struct {
	Accounts []*Account `json:"accounts"`
	Journals []*Journal `json:"journals"`
}
