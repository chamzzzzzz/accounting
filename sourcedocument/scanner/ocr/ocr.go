package ocr

import (
	"context"
	"fmt"

	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ocr/normalizer"
	"github.com/chamzzzzzz/gocr"
)

type Scanner struct {
	Recognizer gocr.Recognizer
	Normalizer *normalizer.Normalizer
}

func (s *Scanner) Scan(ctx context.Context, document *scanner.Document) (*sourcedocument.SourceDocument, error) {
	result, err := s.Recognizer.Recognize(ctx, &gocr.Document{Path: document.Path})
	if err != nil {
		return nil, err
	}
	if result.Code != "0" {
		return nil, fmt.Errorf("ocr error: %s, %s", result.Code, result.Message)
	}
	sd := &sourcedocument.SourceDocument{}
	for _, observation := range result.Observations {
		annotation := &sourcedocument.Annotation{
			Text: observation.Text,
		}
		if observation.BoundingBox != nil {
			annotation.Location = &sourcedocument.Location{
				X: observation.BoundingBox.Origin.X,
				Y: observation.BoundingBox.Origin.Y,
				W: observation.BoundingBox.Size.Width,
				H: observation.BoundingBox.Size.Height,
			}
		}
		sd.Annotations = append(sd.Annotations, annotation)
	}

	if s.Normalizer != nil {
		return s.Normalizer.Normalize(ctx, sd)
	}
	return sd, nil
}
