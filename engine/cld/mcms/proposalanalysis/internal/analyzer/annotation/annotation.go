package annotation

type Annotation interface {
	Name() string
	Type() string
	Value() any
	AnalyzerID() string
}

type Annotations []Annotation

type annotation struct {
	name       string
	atype      string
	value      any
	analyzerID string
}

var _ Annotation = &annotation{}

func (a *annotation) Name() string       { return a.name }
func (a *annotation) Type() string       { return a.atype }
func (a *annotation) Value() any         { return a.value }
func (a *annotation) AnalyzerID() string { return a.analyzerID }

// New creates a new annotation with the given name, type, and value.
func New(name, atype string, value any) Annotation {
	return &annotation{
		name:  name,
		atype: atype,
		value: value,
	}
}

// NewWithAnalyzer creates a new annotation tagged with the ID of the
// analyzer that produced it.
func NewWithAnalyzer(name, atype string, value any, analyzerID string) Annotation {
	return &annotation{
		name:       name,
		atype:      atype,
		value:      value,
		analyzerID: analyzerID,
	}
}
