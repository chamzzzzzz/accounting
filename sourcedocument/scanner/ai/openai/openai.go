package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ai"
)

type Scanner struct {
	ai.Spec
}

func (s *Scanner) Scan(ctx context.Context, document *scanner.Document) (*sourcedocument.SourceDocument, error) {
	spec := s.Spec
	if spec.APIKey == "" || spec.BaseURL == "" {
		return nil, fmt.Errorf("openai: APIKey and BaseURL are required")
	}
	if spec.Model == "" {
		spec.Model = "gpt-4o"
	}
	if spec.Prompt == nil {
		spec.Prompt = &ai.Prompt{
			System: "You are an accountant assistant. Extract information from the source document and return it as JSON with the following schema: " +
				`{"annotations":[{"label":"","text":""},{"text":""}],"from":"","to":"","amount":"","order_number":"","merchant":"","description":"","date":""}`,
			User: "Extract information from this document.",
		}
	}

	b, err := os.ReadFile(document.Path)
	if err != nil {
		return nil, err
	}

	var mime string
	switch ext := strings.ToLower(filepath.Ext(document.Path)); ext {
	case ".png":
		mime = "image/png"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	default:
		mime = "image/jpeg"
	}

	url := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(b))

	temperature := 1.0
	if spec.Parameters != nil && spec.Parameters.Temperature != 0 {
		temperature = spec.Parameters.Temperature
	}
	topP := 1.0
	if spec.Parameters != nil && spec.Parameters.TopP != 0 {
		topP = spec.Parameters.TopP
	}

	body := map[string]any{
		"model": spec.Model,
		"messages": []any{
			map[string]any{"role": "system", "content": spec.Prompt.System},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": spec.Prompt.User},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}},
				},
			},
		},
		"temperature": temperature,
		"top_p":       topP,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", spec.BaseURL+"/chat/completions", buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+spec.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: %d %s", resp.StatusCode, b)
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	if len(res.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices")
	}

	var sd sourcedocument.SourceDocument
	if err := json.Unmarshal([]byte(res.Choices[0].Message.Content), &sd); err != nil {
		return nil, err
	}

	return &sd, nil
}
