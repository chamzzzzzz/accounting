package basic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/rule"
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/voucher"
)

type Strategy struct {
}

func (s *Strategy) PrepareVoucher(ctx context.Context, book *book.Book, sourcedocument *sourcedocument.SourceDocument) ([]*voucher.Voucher, error) {
	var vouchers []*voucher.Voucher
	for _, rule := range book.Rules {
		if match(book, rule, sourcedocument) {
			voucher, err := prepare(book, rule, sourcedocument)
			if err != nil {
				return nil, err
			}
			vouchers = append(vouchers, voucher)
		}
	}
	return vouchers, nil
}

func match(_ *book.Book, rule *rule.Rule, sourcedocument *sourcedocument.SourceDocument) bool {
	for _, a := range rule.Match.Annotations {
		if a.Label == "" && a.Text == "" {
			continue
		}
		found := false
		for _, b := range sourcedocument.Annotations {
			if a.Label != "" && !strings.Contains(b.Label, a.Label) {
				continue
			}
			if a.Text != "" && !strings.Contains(b.Text, a.Text) {
				continue
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

func prepare(book *book.Book, rule *rule.Rule, sourcedocument *sourcedocument.SourceDocument) (*voucher.Voucher, error) {
	data := struct {
		Id     string
		From   *account.Account
		To     *account.Account
		Date   string
		Amount *amount.Amount
	}{}

	from, err := findAccount(book, sourcedocument, rule.Prepare.From)
	if err != nil {
		return nil, err
	}
	data.From = from

	to, err := findAccount(book, sourcedocument, rule.Prepare.To)
	if err != nil {
		return nil, err
	}
	data.To = to

	date, err := findDate(sourcedocument, rule.Prepare.Date)
	if err != nil {
		return nil, err
	}
	data.Date = date

	amount, err := findAmount(sourcedocument, rule.Prepare.Amount)
	if err != nil {
		return nil, err
	}
	data.Amount = amount

	id, err := generateId(date)
	if err != nil {
		return nil, err
	}
	data.Id = id

	b, err := json.Marshal(rule.Voucher)
	if err != nil {
		return nil, err
	}
	tpl, err := template.New("voucher").Option("missingkey=zero").Parse(string(b))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	voucher := &voucher.Voucher{}
	if err := json.Unmarshal(buf.Bytes(), voucher); err != nil {
		return nil, err
	}
	return voucher, nil
}

func findAccount(book *book.Book, sourcedocument *sourcedocument.SourceDocument, label string) (*account.Account, error) {
	label = strings.TrimSpace(label)
	if label == "" || label == "-" {
		return nil, nil
	}
	annotation := findAnnotation(sourcedocument, label)
	if annotation == nil {
		return nil, fmt.Errorf("account not found in %q", label)
	}
	return formatAccount(book, annotation.Text), nil
}

func formatAccount(book *book.Book, title string) *account.Account {
	title = strings.TrimSpace(title)
	for _, acc := range book.Accounts {
		if acc == nil {
			continue
		}
		for _, label := range acc.Labels {
			if matchAccountLabel(title, label) {
				return acc
			}
		}
	}
	return nil
}

func matchAccountLabel(title string, label *account.Label) bool {
	if label == nil || len(label.Words) == 0 {
		return false
	}
	for _, word := range label.Words {
		if !strings.Contains(title, word) {
			return false
		}
	}
	return true
}

func findDate(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	label = strings.TrimSpace(label)
	if label != "" {
		annotation := findAnnotation(sourcedocument, label)
		if annotation == nil {
			return "", fmt.Errorf("date not found in %q", label)
		}
		return formatDate(annotation.Text)
	}
	for _, a := range sourcedocument.Annotations {
		if strings.TrimSpace(a.Label) != "" || strings.TrimSpace(a.Text) == "" {
			continue
		}
		if text, ok := matchDate(a.Text); ok {
			return formatDate(text)
		}
	}
	for _, a := range sourcedocument.Annotations {
		if strings.TrimSpace(a.Label) == "" || strings.TrimSpace(a.Text) == "" {
			continue
		}
		if text, ok := matchDate(a.Text); ok {
			return formatDate(text)
		}
	}
	return "", fmt.Errorf("date not found")
}

func matchDate(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	patterns := []string{
		`\d{4}年\d{1,2}月\d{1,2}日\s*\d{1,2}:\d{1,2}:\d{1,2}`,
		`\d{4}-\d{1,2}-\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if m := re.FindString(text); m != "" {
			return m, true
		}
	}
	return "", false
}

func formatDate(text string) (string, error) {
	text = strings.TrimSpace(text)
	layouts := []string{
		"2006年1月2日 15:04:05",
		"2006年1月2日15:04:05",
		"2006年01月02日 15:04:05",
		"2006年01月02日15:04:05",
		"2006-1-2 15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, text, time.Local)
		if err == nil {
			return t.Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("date layout not supported %q", text)
}

func findAmount(sourcedocument *sourcedocument.SourceDocument, label string) (*amount.Amount, error) {
	label = strings.TrimSpace(label)
	if label != "" {
		annotation := findAnnotation(sourcedocument, label)
		if annotation == nil {
			return nil, fmt.Errorf("amount not found in %q", label)
		}
		return formatAmount(annotation.Text)
	}
	for _, a := range sourcedocument.Annotations {
		if strings.TrimSpace(a.Label) != "" || strings.TrimSpace(a.Text) == "" {
			continue
		}
		if text, ok := matchAmount(a.Text); ok {
			return formatAmount(text)
		}
	}
	for _, a := range sourcedocument.Annotations {
		if strings.TrimSpace(a.Label) == "" || strings.TrimSpace(a.Text) == "" {
			continue
		}
		if text, ok := matchAmount(a.Text); ok {
			return formatAmount(text)
		}
	}
	return nil, fmt.Errorf("amount not found")
}

func matchAmount(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	re := regexp.MustCompile(`[¥￥]?\s*[-+]?(?:\d{1,3}(?:,\d{3})+|\d+)(?:\.\d+)?`)
	indexes := re.FindAllStringIndex(text, -1)
	for _, idx := range indexes {
		m := strings.TrimSpace(text[idx[0]:idx[1]])
		if m == "" || m == "¥" || m == "￥" {
			continue
		}
		if !strings.ContainsAny(m, "¥￥.,") {
			continue
		}
		left := text[:idx[0]]
		right := text[idx[1]:]
		if strings.HasSuffix(left, ":") || strings.HasSuffix(left, "：") || strings.HasPrefix(right, ":") || strings.HasPrefix(right, "：") {
			continue
		}
		if strings.HasSuffix(left, "-") || strings.HasPrefix(right, "-") || strings.HasSuffix(left, "/") || strings.HasPrefix(right, "/") {
			continue
		}
		return m, true
	}
	return "", false
}

func formatAmount(text string) (*amount.Amount, error) {
	text, ok := matchAmount(text)
	if !ok {
		return nil, fmt.Errorf("amount format not supported %q", text)
	}
	clean := strings.ReplaceAll(text, " ", "")
	clean = strings.TrimPrefix(clean, "¥")
	clean = strings.TrimPrefix(clean, "￥")
	clean = strings.ReplaceAll(clean, ",", "")
	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return nil, err
	}
	if value < 0 {
		value = -value
	}
	return &amount.Amount{Quantity: fmt.Sprintf("%.2f", value), Currency: "CNY"}, nil
}

func findAnnotation(sourcedocument *sourcedocument.SourceDocument, label string) *sourcedocument.Annotation {
	for _, a := range sourcedocument.Annotations {
		if strings.Contains(a.Label, label) {
			return a
		}
	}
	return nil
}

func generateId(date string) (string, error) {
	t, err := time.ParseInLocation(time.RFC3339, date, time.Local)
	if err != nil {
		return "", err
	}
	return t.Format("20060102150405"), nil
}
