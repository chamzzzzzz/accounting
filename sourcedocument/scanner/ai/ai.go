package ai

type Prompt struct {
	System string `json:"system,omitempty"`
	User   string `json:"user,omitempty"`
}

type Parameters struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

type Spec struct {
	BaseURL    string      `json:"base_url,omitempty"`
	APIKey     string      `json:"api_key,omitempty"`
	Model      string      `json:"model,omitempty"`
	Prompt     *Prompt     `json:"prompt,omitempty"`
	Parameters *Parameters `json:"parameters,omitempty"`
}
