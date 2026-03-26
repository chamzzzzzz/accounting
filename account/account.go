package account

type Account struct {
	Code        string `json:"code,omitempty"`
	Title       string `json:"title,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type ChartOfAccounts struct {
	Accounts []*Account `json:"accounts,omitempty"`
}
