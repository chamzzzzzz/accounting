package normalizer

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/chamzzzzzz/accounting/sourcedocument"
)

func TestNormalize_WithSampleOCRAndExpectedOutput(t *testing.T) {
	t.Run("generic layout regression", func(t *testing.T) {
		input := fixtureGenericLayoutInput()
		expected := fixtureGenericLayoutExpected()

		n := &Normalizer{}
		out, err := n.Normalize(context.Background(), input)
		if err != nil {
			t.Fatalf("normalize source document: %v", err)
		}

		assertNoLocation(t, out.Annotations)
		assertAnnotationsEqual(t, out.Annotations, expected)
	})

	t.Run("nil source document", func(t *testing.T) {
		n := &Normalizer{}
		out, err := n.Normalize(context.Background(), nil)
		if err != nil {
			t.Fatalf("normalize nil source: %v", err)
		}
		if out != nil {
			t.Fatalf("expected nil output for nil source")
		}
	})

	t.Run("no located annotations preserve text annotations", func(t *testing.T) {
		src := fixturePlainAnnotationsInput()
		n := &Normalizer{}
		out, err := n.Normalize(context.Background(), src)
		if err != nil {
			t.Fatalf("normalize source document: %v", err)
		}
		want := fixturePlainAnnotationsExpected()
		assertAnnotationsEqual(t, out.Annotations, want)
	})
}

func TestNormalize_WithUserProvidedFiles(t *testing.T) {
	inputPath := os.Getenv("NORMALIZER_TEST_INPUT")
	expectPath := os.Getenv("NORMALIZER_TEST_EXPECT_OUTPUT")
	if inputPath == "" || expectPath == "" {
		t.Skip("set NORMALIZER_TEST_INPUT and NORMALIZER_TEST_EXPECT_OUTPUT to enable this test")
	}

	input := loadSourceDocumentFromFile(t, inputPath)
	expected := loadSourceDocumentFromFile(t, expectPath)

	n := &Normalizer{}
	out, err := n.Normalize(context.Background(), input)
	if err != nil {
		t.Fatalf("normalize source document: %v", err)
	}

	assertNoLocation(t, out.Annotations)
	assertAnnotationsEqual(t, out.Annotations, expected.Annotations)
}

func fixtureGenericLayoutInput() *sourcedocument.SourceDocument {
	return &sourcedocument.SourceDocument{
		Annotations: []*sourcedocument.Annotation{
			{Text: "17:39", Location: &sourcedocument.Location{X: 87, Y: 2424, W: 133, H: 61}},
			{Text: "noise_chunk_100", Location: &sourcedocument.Location{X: 897, Y: 2428, W: 224, H: 56}},
			{Text: "<", Location: &sourcedocument.Location{X: 33, Y: 2292, W: 58, H: 77}},
			{Text: "free_text_a", Location: &sourcedocument.Location{X: 474, Y: 2303, W: 217, H: 59}},
			{Text: "status_text_a", Location: &sourcedocument.Location{X: 474, Y: 1891, W: 220, H: 62}},
			{Text: "field_b:87.92", Location: &sourcedocument.Location{X: 412, Y: 1773, W: 345, H: 55}},
			{Text: "field_e", Location: &sourcedocument.Location{X: 62, Y: 1575, W: 172, H: 47}},
			{Text: "field_d", Location: &sourcedocument.Location{X: 62, Y: 1438, W: 173, H: 49}},
			{Text: "field_c", Location: &sourcedocument.Location{X: 62, Y: 1302, W: 173, H: 52}},
			{Text: "field_f", Location: &sourcedocument.Location{X: 62, Y: 1169, W: 173, H: 49}},
			{Text: "field_a", Location: &sourcedocument.Location{X: 66, Y: 1034, W: 169, H: 51}},
			{Text: "field_g", Location: &sourcedocument.Location{X: 62, Y: 897, W: 88, H: 47}},
			{Text: "valueE", Location: &sourcedocument.Location{X: 893, Y: 1574, W: 213, H: 48}},
			{Text: "valueD(0001)", Location: &sourcedocument.Location{X: 772, Y: 1438, W: 331, H: 51}},
			{Text: "valueC(0002)", Location: &sourcedocument.Location{X: 772, Y: 1302, W: 331, H: 51}},
			{Text: "ID-20260330-0001", Location: &sourcedocument.Location{X: 629, Y: 1170, W: 481, H: 47}},
			{Text: "2026-03-30 17:38:12", Location: &sourcedocument.Location{X: 680, Y: 1034, W: 426, H: 51}},
			{Text: "free_text_b", Location: &sourcedocument.Location{X: 390, Y: 890, W: 717, H: 55}},
		},
	}
}

func fixtureGenericLayoutExpected() []*sourcedocument.Annotation {
	return []*sourcedocument.Annotation{
		{Label: "field_a", Text: "2026-03-30 17:38:12"},
		{Label: "field_b", Text: "87.92"},
		{Text: "status_text_a"},
		{Label: "field_g", Text: "free_text_b"},
		{Label: "field_f", Text: "ID-20260330-0001"},
		{Label: "field_c", Text: "valueC(0002)"},
		{Label: "field_d", Text: "valueD(0001)"},
		{Label: "field_e", Text: "valueE"},
		{Text: "<"},
		{Text: "free_text_a"},
		{Text: "17:39"},
		{Text: "noise_chunk_100"},
	}
}

func fixturePlainAnnotationsInput() *sourcedocument.SourceDocument {
	return &sourcedocument.SourceDocument{Annotations: []*sourcedocument.Annotation{{Label: "a", Text: "  b  "}, {Text: "  c  "}}}
}

func fixturePlainAnnotationsExpected() []*sourcedocument.Annotation {
	return []*sourcedocument.Annotation{{Label: "a", Text: "b"}, {Text: "c"}}
}

func loadSourceDocumentFromFile(t *testing.T, path string) *sourcedocument.SourceDocument {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	var doc sourcedocument.SourceDocument
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("decode json file %q: %v", path, err)
	}
	return &doc
}

func assertNoLocation(t *testing.T, annotations []*sourcedocument.Annotation) {
	t.Helper()
	for i, ann := range annotations {
		if ann != nil && ann.Location != nil {
			t.Fatalf("output annotation #%d still has location", i)
		}
	}
}

func assertAnnotationsEqual(t *testing.T, got, want []*sourcedocument.Annotation) {
	t.Helper()
	actualRows := normalize(got)
	wantRows := normalize(want)
	if len(actualRows) != len(wantRows) {
		t.Fatalf("annotation count mismatch: got %d want %d", len(actualRows), len(wantRows))
	}
	for i := range wantRows {
		if actualRows[i] != wantRows[i] {
			t.Fatalf("annotation mismatch at %d: got %q want %q", i, actualRows[i], wantRows[i])
		}
	}
}

func normalize(annotations []*sourcedocument.Annotation) []string {
	rows := make([]string, 0, len(annotations))
	for _, ann := range annotations {
		if ann == nil {
			continue
		}
		rows = append(rows, ann.Label+"\t"+ann.Text)
	}
	sort.Strings(rows)
	return rows
}
