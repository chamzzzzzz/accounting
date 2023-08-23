package accounting

type Analyzer interface {
	Analyze(source *Source) (*SourceDocument, error)
}
