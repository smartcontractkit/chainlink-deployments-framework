package builtin

import (
	"slices"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

// ProposalSection links a built-in proposal analyzer to its markdown template.
type ProposalSection struct {
	AnalyzerID   string
	TemplateName string
	ReportName   string
}

// proposalSections lists built-in analyzers with dedicated proposal templates.
// Entries are defined at compile time in this package; do not mutate at runtime.
var proposalSections = []ProposalSection{}

// ProposalSections returns a copy of registered built-in proposal sections.
func ProposalSections() []ProposalSection {
	return slices.Clone(proposalSections)
}

// IsRegisteredReportName reports whether a built-in section is registered for name.
func IsRegisteredReportName(name string) bool {
	for _, section := range proposalSections {
		if section.ReportName == name {
			return true
		}
	}

	return false
}

// IsHandledByBuiltinSection reports whether ann is rendered by a registered
// built-in section (matching both report name and analyzer ID when set).
func IsHandledByBuiltinSection(ann annotation.Annotation) bool {
	if ann == nil || !annotation.IsBuiltinReportName(ann.Name()) {
		return false
	}

	for _, section := range proposalSections {
		if ann.Name() != section.ReportName {
			continue
		}
		if section.AnalyzerID != "" && ann.AnalyzerID() != section.AnalyzerID {
			continue
		}

		return true
	}

	return false
}

// FindReport returns the structured report value for a proposal section, if present.
func FindReport(anns annotation.Annotations, section ProposalSection) any {
	for _, ann := range anns {
		if ann.Name() != section.ReportName {
			continue
		}
		if section.AnalyzerID != "" && ann.AnalyzerID() != section.AnalyzerID {
			continue
		}

		return ann.Value()
	}

	return nil
}
