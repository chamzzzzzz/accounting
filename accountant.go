package accounting

import "fmt"

type Accountant struct {
	Recognizer *Recognizer
	Analyzers  []Analyzer
}

func (a *Accountant) Recognize(file string) (*Source, error) {
	return a.Recognizer.Recognize(file)
}

func (a *Accountant) Review(source *Source) (*SourceDocument, error) {
	for _, analyzer := range a.Analyzers {
		doc, err := analyzer.Analyze(source)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("no analyzer can analyze the source")
}
