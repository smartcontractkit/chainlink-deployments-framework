package analyzer

import (
	"context"
	"strings"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// BuildProposalReport assembles a ProposalReport from a single-proposal input.
func BuildProposalReport(ctx context.Context, proposalContext ProposalContext, env deployment.Environment, p *mcms.Proposal) (*ProposalReport, error) {
	rpt := &ProposalReport{Operations: make([]OperationReport, len(p.Operations))}
	for i, op := range p.Operations {
		chainSel := uint64(op.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		chainName, _ := GetChainNameBySelector(chainSel)

		calls, err := analyzeTransactions(ctx, proposalContext, env, family, chainSel, []types.Transaction{op.Transaction})
		if err != nil {
			return nil, err
		}

		rpt.Operations[i] = OperationReport{
			ChainSelector: chainSel,
			ChainName:     chainNameOrUnknown(chainName),
			Family:        family,
			Calls:         calls,
		}
	}

	return rpt, nil
}

// BuildTimelockReport assembles a ProposalReport for timelock-style proposals with batches.
func BuildTimelockReport(ctx context.Context, proposalCtx ProposalContext, env deployment.Environment, p *mcms.TimelockProposal) (*ProposalReport, error) {
	rpt := &ProposalReport{Batches: make([]BatchReport, len(p.Operations))}
	for i, batch := range p.Operations {
		chainSel := uint64(batch.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		chainName, _ := GetChainNameBySelector(chainSel)

		dec, err := analyzeTransactions(ctx, proposalCtx, env, family, chainSel, batch.Transactions)
		if err != nil {
			return nil, err
		}

		ops := make([]OperationReport, len(batch.Transactions))
		for j := range batch.Transactions {
			var calls []*DecodedCall
			if j < len(dec) && dec[j] != nil {
				calls = []*DecodedCall{dec[j]}
			}
			ops[j] = OperationReport{
				ChainSelector: chainSel,
				ChainName:     chainNameOrUnknown(chainName),
				Family:        family,
				Calls:         calls,
			}
		}

		rpt.Batches[i] = BatchReport{
			ChainSelector: chainSel,
			ChainName:     chainNameOrUnknown(chainName),
			Family:        family,
			Operations:    ops,
		}
	}

	return rpt, nil
}

// analyzeTransactions dispatches to the appropriate chain-family analyzer.
func analyzeTransactions(ctx context.Context, proposalCtx ProposalContext, env deployment.Environment, family string, chainSel uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	switch family {
	case chainsel.FamilyEVM:
		return AnalyzeEVMTransactions(ctx, proposalCtx, env, chainSel, txs)
	case chainsel.FamilySolana:
		return AnalyzeSolanaTransactions(proposalCtx, chainSel, txs)
	case chainsel.FamilyAptos:
		return AnalyzeAptosTransactions(proposalCtx, chainSel, txs)
	case chainsel.FamilySui:
		return AnalyzeSuiTransactions(proposalCtx, chainSel, txs)
	case chainsel.FamilyTon:
		return AnalyzeTONTransactions(proposalCtx, chainSel, txs)
	default:
		return []*DecodedCall{}, nil
	}
}

func chainNameOrUnknown(n string) string {
	if n == "" || strings.TrimSpace(n) == "" {
		return "<chain unknown>"
	}

	return n
}
