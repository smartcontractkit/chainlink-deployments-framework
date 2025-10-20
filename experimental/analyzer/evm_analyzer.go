package analyzer

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeEVMTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	chainFamily, err := chainsel.GetSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain family for selector %v: %w", chainSelector, err)
	}
	if chainFamily != chainsel.FamilyEVM {
		return nil, fmt.Errorf("unsupported chain family (%v)", chainFamily)
	}

	decoder := NewTxCallDecoder(nil)

	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		decodedTxs[i], _, _, err = AnalyzeEVMTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze transaction %d: %w", i, err)
		}
	}

	return decodedTxs, nil
}

func AnalyzeEVMTransaction(
	ctx ProposalContext, decoder *EVMTxCallDecoder, chainSelector uint64, mcmsTx types.Transaction,
) (*DecodedCall, *abi.ABI, string, error) {
	evmRegistry := ctx.GetEVMRegistry()
	if evmRegistry == nil {
		return nil, nil, "", errors.New("EVM registry is not available")
	}
	abi, abiStr, err := evmRegistry.GetABIByAddress(chainSelector, mcmsTx.To)
	if err != nil {
		return nil, nil, "", err
	}

	analyzeResult, err := decoder.Decode(mcmsTx.To, abi, mcmsTx.Data)
	if err != nil {
		return nil, nil, "", fmt.Errorf("error analyzing operation: %w", err)
	}

	return analyzeResult, abi, abiStr, nil
}
