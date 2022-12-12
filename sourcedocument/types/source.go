package types

import (
	"strings"
)

type SourceItem struct {
	Index  int
	Text   string
	X      int
	Y      int
	Width  int
	Height int
	Weight int
}

func (item *SourceItem) Point() (int, int) {
	return item.X, item.Y
}

func (item *SourceItem) Size() (int, int) {
	return item.Width, item.Height
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

func (source *Source) Search(text string) (items []*SourceItem) {
	for _, item := range source.Items {
		if strings.Contains(item.Text, text) {
			items = append(items, item)
		}
	}
	return
}
