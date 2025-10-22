package analyzer

import (
	"strings"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
)

// BuildProposalReport assembles a ProposalReport from a single-proposal input.
func BuildProposalReport(ctx ProposalContext, p *mcms.Proposal) (*ProposalReport, error) {
	rpt := &ProposalReport{Operations: make([]OperationReport, len(p.Operations))}
	for i, op := range p.Operations {
		chainSel := uint64(op.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		chainName, _ := GetChainNameBySelector(chainSel)

		var calls []*DecodedCall
		switch family {
		case chainsel.FamilyEVM:
			dec, err := AnalyzeEVMTransactions(ctx, chainSel, []types.Transaction{op.Transaction})
			if err != nil {
				return nil, err
			}
			calls = dec
		case chainsel.FamilySolana:
			dec, err := AnalyzeSolanaTransactions(ctx, chainSel, []types.Transaction{op.Transaction})
			if err != nil {
				return nil, err
			}
			calls = dec
		case chainsel.FamilyAptos:
			dec, err := AnalyzeAptosTransactions(ctx, chainSel, []types.Transaction{op.Transaction})
			if err != nil {
				return nil, err
			}
			calls = dec
		case chainsel.FamilySui:
			dec, err := AnalyzeSuiTransactions(ctx, chainSel, []types.Transaction{op.Transaction})
			if err != nil {
				return nil, err
			}
			calls = dec
		default:
			calls = []*DecodedCall{}
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
func BuildTimelockReport(ctx ProposalContext, p *mcms.TimelockProposal) (*ProposalReport, error) {
	rpt := &ProposalReport{Batches: make([]BatchReport, len(p.Operations))}
	for i, batch := range p.Operations {
		chainSel := uint64(batch.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		chainName, _ := GetChainNameBySelector(chainSel)

		ops := make([]OperationReport, len(batch.Transactions))
		switch family {
		case chainsel.FamilyEVM:
			dec, err := AnalyzeEVMTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for j := range dec {
				ops[j] = OperationReport{
					ChainSelector: chainSel,
					ChainName:     chainNameOrUnknown(chainName),
					Family:        family,
					Calls:         []*DecodedCall{dec[j]},
				}
			}
		case chainsel.FamilySolana:
			dec, err := AnalyzeSolanaTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for j := range dec {
				ops[j] = OperationReport{
					ChainSelector: chainSel,
					ChainName:     chainNameOrUnknown(chainName),
					Family:        family,
					Calls:         []*DecodedCall{dec[j]},
				}
			}
		case chainsel.FamilyAptos:
			dec, err := AnalyzeAptosTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for j := range dec {
				ops[j] = OperationReport{
					ChainSelector: chainSel,
					ChainName:     chainNameOrUnknown(chainName),
					Family:        family,
					Calls:         []*DecodedCall{dec[j]},
				}
			}
		case chainsel.FamilySui:
			dec, err := AnalyzeSuiTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for j := range dec {
				ops[j] = OperationReport{
					ChainSelector: chainSel,
					ChainName:     chainNameOrUnknown(chainName),
					Family:        family,
					Calls:         []*DecodedCall{dec[j]},
				}
			}
		default:
			for j := range batch.Transactions {
				ops[j] = OperationReport{
					ChainSelector: chainSel,
					ChainName:     chainNameOrUnknown(chainName),
					Family:        family,
					Calls:         nil,
				}
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

func chainNameOrUnknown(n string) string {
	if n == "" || strings.TrimSpace(n) == "" {
		return "<chain unknown>"
	}

	return n
}
