package sourcedocument

type Location struct {
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
	W int `json:"w,omitempty"`
	H int `json:"h,omitempty"`
}

type Annotation struct {
	Text     string    `json:"text,omitempty"`
	Location *Location `json:"location,omitempty"`
}

type SourceDocument struct {
	Annotations []*Annotation `json:"annotations,omitempty"`
	Name        string        `json:"name,omitempty"`
	Class       string        `json:"class,omitempty"`
	From        string        `json:"from,omitempty"`
	To          string        `json:"to,omitempty"`
	Amount      string        `json:"amount,omitempty"`
	OrderNumber string        `json:"order_number,omitempty"`
	Merchant    string        `json:"merchant,omitempty"`
	Description string        `json:"description,omitempty"`
	Date        string        `json:"date,omitempty"`
}
