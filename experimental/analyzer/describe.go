package analyzer

import (
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func DescribeTimelockProposal(ctx ProposalContext, proposal *mcms.TimelockProposal) (string, error) {
	report, err := BuildTimelockReport(ctx, proposal)
	if err != nil {
		return "", err
	}

	return NewMarkdownRenderer().RenderTimelock(report), nil
}

func DescribeProposal(ctx ProposalContext, proposal *mcms.Proposal) (string, error) {
	report, err := BuildProposalReport(ctx, proposal)
	if err != nil {
		return "", err
	}

	return NewMarkdownRenderer().RenderProposal(report), nil
}

func describeBatchOperations(ctx ProposalContext, batches []types.BatchOperation) ([][]string, error) {
	describedBatches := make([][]string, len(batches))
	for batchIdx, batch := range batches {
		chainSel := uint64(batch.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		describedBatches[batchIdx] = make([]string, len(batch.Transactions))
		switch family {
		case chainsel.FamilyEVM:
			describedTxs, err := AnalyzeEVMTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.DescriptorContext(chainSel))
			}
		case chainsel.FamilySolana:
			describedTxs, err := AnalyzeSolanaTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.DescriptorContext(chainSel))
			}
		case chainsel.FamilyAptos:
			describedTxs, err := AnalyzeAptosTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.DescriptorContext(chainSel))
			}
		case chainsel.FamilySui:
			describedTxs, err := AnalyzeSuiTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.DescriptorContext(chainSel))
			}
		default:
			for callIdx := range batch.Transactions {
				describedBatches[batchIdx][callIdx] = family + " transaction decoding is not supported"
			}
		}
	}

	return describedBatches, nil
}

func describeOperations(ctx ProposalContext, operations []types.Operation) ([]string, error) {
	describedOperations := make([]string, len(operations))
	for callIdx, operation := range operations {
		chainSel := uint64(operation.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}

		switch family {
		case chainsel.FamilyEVM:
			describedTransaction, err := AnalyzeEVMTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.DescriptorContext(uint64(operation.ChainSelector)))

		case chainsel.FamilySolana:
			describedTransaction, err := AnalyzeSolanaTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.DescriptorContext(uint64(operation.ChainSelector)))

		case chainsel.FamilyAptos:
			describedTransaction, err := AnalyzeAptosTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.DescriptorContext(uint64(operation.ChainSelector)))
		case chainsel.FamilySui:
			describedTransaction, err := AnalyzeSuiTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.DescriptorContext(uint64(operation.ChainSelector)))

		default:
			describedOperations[callIdx] = family + " transaction decoding is not supported"
		}
	}

	return describedOperations, nil
}
