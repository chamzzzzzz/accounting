package accounting

type Account struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Entry struct {
	Account  string `json:"account"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type Voucher struct {
	Date string   `json:"date"`
	From []*Entry `json:"from"`
	To   []*Entry `json:"to"`
}

type Journal struct {
	Date     string     `json:"date"`
	Vouchers []*Voucher `json:"vouchers"`
}

type Book struct {
	Accounts []*Account `json:"accounts"`
	Journals []*Journal `json:"journals"`
}
