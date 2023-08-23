package accounting

import (
	"github.com/chamzzzzzz/gocr"
)

type Recognizer struct {
	ocr *gocr.OCR
}

func NewRecognizer(driverName string) (*Recognizer, error) {
	ocr, err := gocr.Open(driverName, "")
	if err != nil {
		return nil, err
	}
	return &Recognizer{ocr: ocr}, nil
}

func (r *Recognizer) Recognize(file string) (*Source, error) {
	result, err := r.ocr.Recognize(file)
	if err != nil {
		return nil, err
	}
	source := &Source{
		From: result,
	}
	for i, o := range result.Observations {
		item := &SourceItem{
			Index:  i,
			Text:   o.Text,
			X:      o.BoudingBox.Origin.X,
			Y:      o.BoudingBox.Origin.Y,
			W:      o.BoudingBox.Size.Width,
			H:      o.BoudingBox.Size.Height,
			Weight: o.Confidence,
		}
		source.Items = append(source.Items, item)
	}
	return source, nil
}
