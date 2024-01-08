package accounting

import "strings"

type SourceItem struct {
	Index  int
	Name   string
	Text   string
	X      int
	Y      int
	W      int
	H      int
	Weight int
}

type Source struct {
	From  any
	Items []*SourceItem
}

func (source *Source) Item(i int) *SourceItem {
	if i >= 0 && i < len(source.Items) {
		return source.Items[i]
	}
	return nil
}

func (source *Source) TextContains(text string) (items []*SourceItem) {
	for _, item := range source.Items {
		if strings.Contains(item.Text, text) {
			items = append(items, item)
		}
	}
	return
}

func (source *Source) TextEquals(text string) (items []*SourceItem) {
	for _, item := range source.Items {
		if item.Text == text {
			items = append(items, item)
		}
	}
	return
}

func (source *Source) TextPrefix(text string) (items []*SourceItem) {
	for _, item := range source.Items {
		if strings.HasPrefix(item.Text, text) {
			items = append(items, item)
		}
	}
	return
}

func (source *Source) First(items []*SourceItem) *SourceItem {
	if len(items) > 0 {
		return items[0]
	}
	return nil
}

func (source *Source) Last(items []*SourceItem) *SourceItem {
	if len(items) > 0 {
		return items[len(items)-1]
	}
	return nil
}

func (source *Source) HorizontalKeyValue(key string) (item *SourceItem, text string) {
	itemkey := source.First(source.TextEquals(key))
	if itemkey == nil {
		return
	}
	itemval := source.Item(itemkey.Index + 1)
	if itemval == nil {
		return
	}
	item = itemval
	text = itemval.Text
	return
}

func (source *Source) HorizontalKeyValueText(key string) string {
	_, text := source.HorizontalKeyValue(key)
	return text
}

func (source *Source) SeparatorJoinedKeyValue(key string, separator string) (item *SourceItem, text string) {
	for _, itemkv := range source.TextPrefix(key) {
		if kv := strings.SplitN(itemkv.Text, separator, 2); len(kv) == 2 && kv[0] == key {
			item = itemkv
			text = kv[1]
			return
		}
	}
	return
}

func (source *Source) SeparatorJoinedKeyValueText(key string, separator string) string {
	_, text := source.SeparatorJoinedKeyValue(key, separator)
	return text
}

func (source *Source) ColonJoinedKeyValue(key string) (*SourceItem, string) {
	return source.SeparatorJoinedKeyValue(key, "：")
}

func (source *Source) ColonJoinedKeyValueText(key string) string {
	return source.SeparatorJoinedKeyValueText(key, "：")
}

type SourceDocument struct {
	Source      *Source `json:"source"`
	Name        string  `json:"name"`
	Class       string  `json:"class"`
	From        string  `json:"from"`
	To          string  `json:"to"`
	Amount      string  `json:"amount"`
	OrderNumber string  `json:"orderNumber"`
	Merchant    string  `json:"merchant"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}
