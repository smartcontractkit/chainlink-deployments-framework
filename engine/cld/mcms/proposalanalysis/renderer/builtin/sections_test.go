package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

func TestIsRegisteredReportName(t *testing.T) {
	t.Parallel()

	assert.False(t, IsRegisteredReportName("cld.builtin.example.report"))
}

func TestFindReport(t *testing.T) {
	t.Parallel()

	section := ProposalSection{
		AnalyzerID:   "cld.proposal.example",
		TemplateName: "builtinExample",
		ReportName:   "cld.builtin.example.report",
	}
	type exampleReport struct {
		Message string
	}
	report := exampleReport{Message: "ok"}
	anns := annotation.Annotations{
		annotation.NewWithAnalyzer(
			section.ReportName,
			"struct",
			report,
			section.AnalyzerID,
		),
	}

	got, ok := FindReport(anns, section).(exampleReport)
	require.True(t, ok)
	assert.Equal(t, report, got)

	assert.Nil(t, FindReport(anns, ProposalSection{
		AnalyzerID:   "other",
		TemplateName: section.TemplateName,
		ReportName:   section.ReportName,
	}))

	annsWithoutAnalyzerID := annotation.Annotations{
		annotation.New(section.ReportName, "struct", report),
	}
	assert.Nil(t, FindReport(annsWithoutAnalyzerID, section))
}
