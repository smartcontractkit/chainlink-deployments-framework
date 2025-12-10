package analyzer

import (
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeTONTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	decoder := ton.NewDecoder()
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeTONTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Sui transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeTONTransaction(ctx ProposalContext, decoder sdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	decodedOp, err := decoder.Decode(mcmsTx, mcmsTx.ContractType)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single transaction decode failure
		errStr := fmt.Errorf("failed to decode Sui transaction: %w", err)

		return &DecodedCall{
			Address: mcmsTx.To,
			Method:  errStr.Error(),
		}, nil
	}
	namedArgs, err := toNamedFields(decodedOp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert decoded operation to named arguments: %w", err)
	}

	return &DecodedCall{
		Address: mcmsTx.To,
		Method:  decodedOp.MethodName(),
		Inputs:  namedArgs,
		Outputs: []NamedField{},
	}, nil
}
