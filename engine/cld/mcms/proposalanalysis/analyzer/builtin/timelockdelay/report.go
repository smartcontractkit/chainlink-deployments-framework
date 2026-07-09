package timelockdelay

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

const (
	// ValidatorID identifies the built-in timelock delay proposal analyzer.
	ValidatorID = "cld.proposal.timelock_delay"
	// ReportName is the structured report annotation emitted by ValidatorID.
	ReportName = "cld.builtin.timelock_delay.report"
)

// ChainMinDelay captures on-chain minDelay for one timelock chain.
type ChainMinDelay struct {
	ChainSelector uint64
	ChainName     string
	MinDelay      string
	Address       string
}

// Report is the structured output of Validator.
type Report struct {
	ProposalDelay  string
	ChainMinDelays []ChainMinDelay
	Validation     string
	Severity       analyzer.Severity
}

func reportAnnotations(report Report) analyzer.Annotations {
	return analyzer.Annotations{
		analyzer.NewAnnotation(ReportName, "struct", report),
	}
}
