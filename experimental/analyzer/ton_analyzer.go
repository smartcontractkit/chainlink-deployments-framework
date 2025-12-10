package analyzer

import (
	"fmt"

	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeTONTransactions(ctx ProposalContext, txs []types.Transaction) ([]*DecodedCall, error) {
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeTONTransaction(ctx, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze TON transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeTONTransaction(_ ProposalContext, mcmsTx types.Transaction) (*DecodedCall, error) {
	decoder := ton.NewDecoder()
	decodedOp, err := decoder.Decode(mcmsTx, mcmsTx.ContractType)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single transaction decode failure
		errStr := fmt.Errorf("failed to decode TON transaction: %w", err)

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
