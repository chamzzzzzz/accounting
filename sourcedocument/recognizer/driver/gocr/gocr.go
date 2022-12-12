package gocr

import (
	"github.com/chamzzzzzz/accounting/sourcedocument/recognizer/driver"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
	"github.com/chamzzzzzz/gocr"
)

const (
	DriverName = "gocr"
	ParamName  = "macOCR"
)

type Recognizer struct {
	ocr *gocr.OCR
}

func (recognizer *Recognizer) Recognize(sourcefrom any) (*types.Source, error) {
	result, err := recognizer.ocr.Recognize(sourcefrom.(string))
	if err != nil {
		return nil, err
	}
	source := &types.Source{
		From: result,
	}
	for i, observation := range result.Observations {
		item := &types.SourceItem{
			Index:  i,
			Text:   observation.Text,
			X:      observation.BoudingBox.Origin.X,
			Y:      observation.BoudingBox.Origin.Y,
			Width:  observation.BoudingBox.Size.Width,
			Height: observation.BoudingBox.Size.Height,
			Weight: observation.Confidence,
		}
		source.Items = append(source.Items, item)
	}
	return source, nil
}

func (recognizer *Recognizer) Driver() driver.Driver {
	return &Driver{}
}

type Driver struct {
}

func (driver *Driver) Open(paramName string) (driver.Recognizer, error) {
	if paramName == "" {
		paramName = ParamName
	}
	ocr, err := gocr.Open(paramName, "")
	if err != nil {
		return nil, err
	}
	return &Recognizer{ocr}, nil
}

func init() {
	driver.Register(DriverName, &Driver{})
}
