package account

type Label struct {
	Words []string `json:"words,omitempty"`
}

type Account struct {
	Catalog    string            `json:"catalog,omitempty"`
	Title      string            `json:"title,omitempty"`
	Labels     []*Label          `json:"labels,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}
