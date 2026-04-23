package basic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
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

var refer = &voucher.Voucher{
	Id:      "{{.Id}}",
	Date:    "{{.Date}}",
	Catalog: "{{.Catalog}}",
	Entries: []*voucher.Entry{
		{Account: "{{.From.Title}}", Amount: &amount.Amount{Quantity: "-{{.Amount.Quantity}}", Currency: "{{.Amount.Currency}}"}},
		{Account: "{{.To.Title}}", Amount: &amount.Amount{Quantity: "{{.Amount.Quantity}}", Currency: "{{.Amount.Currency}}"}},
	},
	OrderNumber: "{{.OrderNumber}}",
	Merchant:    "{{.Merchant}}",
	Description: "{{.Description}}",
}

type Strategy struct {
}

func (s *Strategy) PrepareVoucher(ctx context.Context, book *book.Book, sourcedocument *sourcedocument.SourceDocument) ([]*voucher.Voucher, error) {
	var prepared []*voucher.Voucher
	for _, rule := range sort(book.Rules) {
		var data data
		if match(book, rule, sourcedocument, data) {
			vouchers, err := prepare(book, rule, sourcedocument, refer, data, 0)
			if err != nil {
				return nil, err
			}
			if len(vouchers) > 0 {
				prepared = append(prepared, vouchers...)
				if !rule.Continue {
					break
				}
			}
		}
	}
	return prepared, nil
}

func match(_ *book.Book, rule *rule.Rule, sourcedocument *sourcedocument.SourceDocument, _ data) bool {
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

func prepare(book *book.Book, rule *rule.Rule, sourcedocument *sourcedocument.SourceDocument, refer *voucher.Voucher, data data, deep int) ([]*voucher.Voucher, error) {
	if rule.Prepare != nil {
		from, err := findAccount(book, sourcedocument, rule.Prepare.From)
		if err != nil {
			return nil, err
		}
		if from != nil {
			data.From = from
		}

		to, err := findAccount(book, sourcedocument, rule.Prepare.To)
		if err != nil {
			return nil, err
		}
		if to != nil {
			data.To = to
		}

		date, err := findDate(sourcedocument, rule.Prepare.Date)
		if err != nil {
			return nil, err
		}
		if date != "" {
			data.Date = date
		}

		amount, err := findAmount(sourcedocument, rule.Prepare.Amount)
		if err != nil {
			return nil, err
		}
		if amount != nil {
			data.Amount = amount
		}

		orderNumber, err := findOrderNumber(sourcedocument, rule.Prepare.OrderNumber)
		if err != nil {
			return nil, err
		}
		if orderNumber != "" {
			data.OrderNumber = orderNumber
		}

		merchant, err := findMerchant(sourcedocument, rule.Prepare.Merchant)
		if err != nil {
			return nil, err
		}
		if merchant != "" {
			data.Merchant = merchant
		}

		description, err := findDescription(sourcedocument, rule.Prepare.Description)
		if err != nil {
			return nil, err
		}
		if description != "" {
			data.Description = description
		}

		catalog, err := findCatalog(sourcedocument, rule.Prepare.Catalog)
		if err != nil {
			return nil, err
		}
		if catalog != "" {
			data.Catalog = catalog
		}
	}

	if data.Date != "" {
		id, err := generateId(data.Date)
		if err != nil {
			return nil, err
		}
		data.Id = id
	}

	refer = compose(refer, rule.Voucher)
	var prepared []*voucher.Voucher
	if len(rule.Specs) > 0 && deep < 10 {
		for _, spec := range sort(rule.Specs) {
			if match(book, spec, sourcedocument, data) {
				vouchers, err := prepare(book, spec, sourcedocument, refer, data, deep+1)
				if err != nil {
					return nil, err
				}
				if len(vouchers) > 0 {
					prepared = append(prepared, vouchers...)
					if !spec.Continue {
						break
					}
				}
			}
		}
	}
	if len(prepared) > 0 {
		return prepared, nil
	}

	if data.Date == "" {
		return nil, fmt.Errorf("no date")
	}
	if data.Amount == nil {
		return nil, fmt.Errorf("no amount")
	}
	if data.From == nil {
		data.From = &account.Account{}
	}
	if data.To == nil {
		data.To = &account.Account{}
	}

	b, err := json.Marshal(refer)
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
	prepared = append(prepared, voucher)
	return prepared, nil
}

func findAccount(book *book.Book, sourcedocument *sourcedocument.SourceDocument, label string) (*account.Account, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, nil
	}
	if label, ok := strings.CutPrefix(label, "#"); ok {
		label = strings.TrimSpace(label)
		if label == "" {
			return nil, nil
		}
		annotation := findAnnotation(sourcedocument, label)
		if annotation == nil {
			return nil, fmt.Errorf("account not found in %q", label)
		}
		return formatAccount(book, annotation.Text), nil
	} else {
		return getAccount(book, label), nil
	}
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

func getAccount(book *book.Book, title string) *account.Account {
	for _, acc := range book.Accounts {
		if acc == nil {
			continue
		}
		if acc.Title == title {
			return acc
		}
	}
	return nil
}

func findDate(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", nil
	}

	if label, ok := strings.CutPrefix(label, "#"); ok {
		label = strings.TrimSpace(label)
		if label != "" {
			annotation := findAnnotation(sourcedocument, label)
			if annotation == nil {
				return "", fmt.Errorf("date not found in %q", label)
			}
			if text, ok := matchDate(annotation.Text); ok {
				return formatDate(text)
			}
			return "", fmt.Errorf("date %q not match found in %q ", annotation.Text, label)
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
	} else {
		return formatDate(label)
	}
}

func matchDate(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	patterns := []string{
		`(\d{4}年\d{1,2}月\d{1,2}日).*?(\d{1,2}:\d{1,2}:\d{1,2})`,
		`(\d{4}-\d{1,2}-\d{1,2}).*?(\d{1,2}:\d{1,2}:\d{1,2})`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if m := re.FindStringSubmatch(text); len(m) == 3 {
			return m[1] + " " + m[2], true
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
	if label == "" {
		return nil, nil
	}

	if label, ok := strings.CutPrefix(label, "#"); ok {
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
	} else {
		return formatAmount(label)
	}
}

func matchAmount(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	re := regexp.MustCompile(`[-+]?\s*[¥￥]?\s*[-+]?(?:\d{1,3}(?:,\d{3})+|\d+)(?:\.\d+)?`)
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
	clean = strings.ReplaceAll(clean, "¥", "")
	clean = strings.ReplaceAll(clean, "￥", "")
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

func findOrderNumber(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	return findText(sourcedocument, label, "order number")
}

func findMerchant(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	return findText(sourcedocument, label, "merchant")
}

func findDescription(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	return findText(sourcedocument, label, "description")
}

func findCatalog(sourcedocument *sourcedocument.SourceDocument, label string) (string, error) {
	return findText(sourcedocument, label, "catalog")
}

func findText(sourcedocument *sourcedocument.SourceDocument, label string, name string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", nil
	}
	if label, ok := strings.CutPrefix(label, "#"); ok {
		label = strings.TrimSpace(label)
		if label == "" {
			return "", nil
		}
		annotation := findAnnotation(sourcedocument, label)
		if annotation == nil {
			return "", fmt.Errorf("%s not found in %q", name, label)
		}
		return strings.TrimSpace(annotation.Text), nil
	} else {
		return label, nil
	}
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

type data struct {
	Id          string
	Catalog     string
	From        *account.Account
	To          *account.Account
	Date        string
	Amount      *amount.Amount
	OrderNumber string
	Merchant    string
	Description string
}

func compose(refer *voucher.Voucher, v *voucher.Voucher) *voucher.Voucher {
	if refer == nil {
		return v
	}
	if v == nil {
		return refer
	}
	cv := &voucher.Voucher{}
	*cv = *refer
	if v.Id != "" {
		cv.Id = v.Id
	}
	if v.Date != "" {
		cv.Date = v.Date
	}
	if v.Catalog != "" {
		cv.Catalog = v.Catalog
	}
	if len(v.Entries) > 0 {
		cv.Entries = v.Entries
	}
	if v.Description != "" {
		cv.Description = v.Description
	}
	return cv
}

func sort(rules []*rule.Rule) []*rule.Rule {
	rs := make([]*rule.Rule, len(rules))
	copy(rs, rules)
	slices.SortStableFunc(rs, func(a, b *rule.Rule) int {
		if a.Priority > b.Priority {
			return 1
		} else if a.Priority < b.Priority {
			return -1
		} else {
			return 0
		}
	})
	return rs
}
