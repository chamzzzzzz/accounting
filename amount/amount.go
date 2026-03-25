package amount

type Amount struct {
	Quantity string `json:"quantity,omitempty"`
	Currency string `json:"currency,omitempty"`
}
