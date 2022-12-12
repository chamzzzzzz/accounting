package scanner

import (
	"fmt"
	"github.com/chamzzzzzz/accounting/sourcedocument/analyzer"
	"github.com/chamzzzzzz/accounting/sourcedocument/recognizer"
	"github.com/chamzzzzzz/accounting/sourcedocument/types"
)

type Option struct {
	Recognizers           []*recognizer.Recognizer
	Analyzers             []*analyzer.Analyzer
	RecognizerDriverNames []string
	RecognizerParamNames  []string
	AnalyzerDriverNames   []string
	AnalyzerParamNames    []string
}

type Scanner struct {
	Recognizers []*recognizer.Recognizer
	Analyzers   []*analyzer.Analyzer
}

func NewScanner(option *Option) (*Scanner, error) {
	if option == nil {
		option = &Option{
			RecognizerDriverNames: recognizer.Drivers(),
			AnalyzerDriverNames:   analyzer.Drivers(),
		}
	}

	scanner := &Scanner{
		Recognizers: option.Recognizers,
		Analyzers:   option.Analyzers,
	}

	pnlen := len(option.RecognizerParamNames)
	for i, dn := range option.RecognizerDriverNames {
		pn := ""
		if pnlen > i {
			pn = option.RecognizerParamNames[i]
		}
		recognizer, err := recognizer.Open(dn, pn)
		if err != nil {
			return nil, err
		}
		scanner.Recognizers = append(scanner.Recognizers, recognizer)
	}

	pnlen = len(option.AnalyzerParamNames)
	for i, dn := range option.AnalyzerDriverNames {
		pn := ""
		if pnlen > i {
			pn = option.AnalyzerParamNames[i]
		}
		analyzer, err := analyzer.Open(dn, pn)
		if err != nil {
			return nil, err
		}
		scanner.Analyzers = append(scanner.Analyzers, analyzer)
	}
	return scanner, nil
}

func (scanner *Scanner) Scan(file string) (*types.SourceDocument, error) {
	if len(scanner.Recognizers) == 0 {
		return nil, fmt.Errorf("no recognizer")
	}
	if len(scanner.Analyzers) == 0 {
		return nil, fmt.Errorf("no analyzer")
	}

	var err error
	var source *types.Source
	for _, recognizer := range scanner.Recognizers {
		if source, err = recognizer.Recognize(file); err == nil && source != nil {
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("recognize error")
	}

	for _, analyzer := range scanner.Analyzers {
		if sourcedocument, err := analyzer.Analyze(source); err == nil && sourcedocument != nil {
			return sourcedocument, err
		}
	}
	return nil, fmt.Errorf("analyze error")
}
