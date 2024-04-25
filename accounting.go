package accounting

type Account struct {
	ID          string `json:"id,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Entry struct {
	Account  string `json:"account,omitempty"`
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
	Note     string `json:"note,omitempty"`
}

type Voucher struct {
	ID          string   `json:"id,omitempty"`
	Date        string   `json:"date,omitempty"`
	Entries     []*Entry `json:"entries,omitempty"`
	Description string   `json:"description,omitempty"`
	Note        string   `json:"note,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type Journal struct {
	Date     string     `json:"date,omitempty"`
	Vouchers []*Voucher `json:"vouchers,omitempty"`
}

type Book struct {
	Accounts []*Account `json:"accounts,omitempty"`
	Journals []*Journal `json:"journals,omitempty"`
}
