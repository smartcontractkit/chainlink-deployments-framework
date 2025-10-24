package analyzer

import (
	"github.com/smartcontractkit/mcms"
)

func DescribeTimelockProposal(ctx ProposalContext, proposal *mcms.TimelockProposal) (string, error) {
	report, err := BuildTimelockReport(ctx, proposal)
	if err != nil {
		return "", err
	}

	// Create fields context for address annotations
	// For timelock proposals with multiple batches, we'll use the first batch's chain selector
	var chainSelector uint64
	if len(proposal.Operations) > 0 {
		chainSelector = uint64(proposal.Operations[0].ChainSelector)
	}
	fieldCtx := ctx.FieldsContext(chainSelector)

	return ctx.GetRenderer().RenderTimelockProposal(report, fieldCtx), nil
}

func DescribeProposal(ctx ProposalContext, proposal *mcms.Proposal) (string, error) {
	report, err := BuildProposalReport(ctx, proposal)
	if err != nil {
		return "", err
	}

	// Create fields context for address annotations
	// For proposals with multiple operations, we'll use the first operation's chain selector
	var chainSelector uint64
	if len(proposal.Operations) > 0 {
		chainSelector = uint64(proposal.Operations[0].ChainSelector)
	}
	fieldCtx := ctx.FieldsContext(chainSelector)

	return ctx.GetRenderer().RenderProposal(report, fieldCtx), nil
}
