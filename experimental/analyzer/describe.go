package analyzer

import (
	"context"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func DescribeTimelockProposal(ctx context.Context, proposalCtx ProposalContext, env deployment.Environment, proposal *mcms.TimelockProposal) (string, error) {
	report, err := BuildTimelockReport(ctx, proposalCtx, env, proposal)
	if err != nil {
		return "", err
	}

	// Create fields context for address annotations
	// For timelock proposals with multiple batches, we'll use the first batch's chain selector
	var chainSelector uint64
	if len(proposal.Operations) > 0 {
		chainSelector = uint64(proposal.Operations[0].ChainSelector)
	}
	fieldCtx := proposalCtx.FieldsContext(chainSelector)

	return proposalCtx.GetRenderer().RenderTimelockProposal(report, fieldCtx), nil
}

func DescribeProposal(ctx context.Context, proposalContext ProposalContext, env deployment.Environment, proposal *mcms.Proposal) (string, error) {
	report, err := BuildProposalReport(ctx, proposalContext, env, proposal)
	if err != nil {
		return "", err
	}

	// Create fields context for address annotations
	// For proposals with multiple operations, we'll use the first operation's chain selector
	var chainSelector uint64
	if len(proposal.Operations) > 0 {
		chainSelector = uint64(proposal.Operations[0].ChainSelector)
	}
	fieldCtx := proposalContext.FieldsContext(chainSelector)

	return proposalContext.GetRenderer().RenderProposal(report, fieldCtx), nil
}
