package ocr

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ocr/normalizer"
	"github.com/chamzzzzzz/gocr"
)

type fakeRecognizer struct {
	result *gocr.Result
	err    error
}

func (f *fakeRecognizer) GetType() string { return "fake" }
func (f *fakeRecognizer) GetId() string   { return "fake" }
func (f *fakeRecognizer) GetOption() gocr.Option {
	return gocr.Option{Type: "fake", Id: "fake"}
}
func (f *fakeRecognizer) Recognize(_ context.Context, _ *gocr.Document) (*gocr.Result, error) {
	return f.result, f.err
}

func TestScan_SuccessCases(t *testing.T) {
	tests := []struct {
		name                string
		scanner             *Scanner
		wantAnnotationTexts []string
		wantHasLocation     bool
	}{
		{
			name:                "raw ocr without normalizer",
			scanner:             &Scanner{Recognizer: &fakeRecognizer{result: sampleOCRResult()}},
			wantAnnotationTexts: []string{"created_at", "2026-03-30 17:38:12"},
			wantHasLocation:     true,
		},
		{
			name: "configured normalizer spec affects behavior",
			scanner: func() *Scanner {
				spec := normalizer.DefaultSpec()
				spec.PairAcceptScore = 1.1
				return &Scanner{
					Recognizer: &fakeRecognizer{result: sampleOCRResult()},
					Normalizer: &normalizer.Normalizer{Spec: spec},
				}
			}(),
			wantAnnotationTexts: []string{"created_at", "2026-03-30 17:38:12"},
			wantHasLocation:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.scanner.Scan(context.Background(), &scanner.Document{Path: "unused"})
			if err != nil {
				t.Fatalf("scan error: %v", err)
			}
			if len(out.Annotations) != len(tc.wantAnnotationTexts) {
				t.Fatalf("annotation count mismatch: got %d want %d", len(out.Annotations), len(tc.wantAnnotationTexts))
			}
			for i, wantText := range tc.wantAnnotationTexts {
				if out.Annotations[i].Text != wantText {
					t.Fatalf("unexpected annotation text at %d: got %q want %q", i, out.Annotations[i].Text, wantText)
				}
				hasLocation := out.Annotations[i].Location != nil
				if hasLocation != tc.wantHasLocation {
					t.Fatalf("unexpected location presence at %d: got %v want %v", i, hasLocation, tc.wantHasLocation)
				}
			}
		})
	}
}

func TestScan_ErrorCases(t *testing.T) {
	t.Run("recognizer returns error", func(t *testing.T) {
		s := &Scanner{Recognizer: &fakeRecognizer{err: errors.New("recognizer failure")}}
		_, err := s.Scan(context.Background(), &scanner.Document{Path: "unused"})
		if err == nil || !strings.Contains(err.Error(), "recognizer failure") {
			t.Fatalf("expected recognizer failure, got: %v", err)
		}
	})

	t.Run("ocr service returns non-zero code", func(t *testing.T) {
		s := &Scanner{Recognizer: &fakeRecognizer{result: &gocr.Result{Code: "500", Message: "service error"}}}
		_, err := s.Scan(context.Background(), &scanner.Document{Path: "unused"})
		if err == nil {
			t.Fatalf("expected ocr code error")
		}
		if !strings.Contains(err.Error(), "ocr error: 500") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func sampleOCRResult() *gocr.Result {
	return &gocr.Result{
		Code: "0",
		Observations: []*gocr.Observation{
			{
				Text: "created_at",
				BoundingBox: &gocr.BoundingBox{
					Origin: gocr.Point{X: 66, Y: 1034},
					Size:   gocr.Size{Width: 169, Height: 51},
				},
			},
			{
				Text: "2026-03-30 17:38:12",
				BoundingBox: &gocr.BoundingBox{
					Origin: gocr.Point{X: 680, Y: 1034},
					Size:   gocr.Size{Width: 426, Height: 51},
				},
			},
		},
	}
}
