package analyzer

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// AnalyzeTONTransactions decodes a slice of TON transactions and returns their decoded representations.
func AnalyzeTONTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	decoder := ton.NewDecoder(bindings.Registry)
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeTONTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze TON transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

// AnalyzeTONTransaction decodes a single TON transaction using the MCMS TON decoder.
//
// Unlike Aptos/Sui analyzers, this function does not unmarshal AdditionalFields because
// the TON decoder only requires tx.Data (BOC cell) and tx.ContractType (metadata).
// AdditionalFields in TON is only used by the encoder/timelock_converter for the Value field.
//
// On decode failure, this function returns a DecodedCall with the error in the Method field
// instead of returning an error. This allows the proposal to continue processing even if
// a single transaction fails to decode.
func AnalyzeTONTransaction(ctx ProposalContext, decoder sdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	contractType, contractVersion := resolveContractInfo(ctx, chainSelector, mcmsTx)

	decodedOp, err := decoder.Decode(mcmsTx, mcmsTx.ContractType)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single transaction decode failure.
		// Instead, put the error message in the Method field so it's visible in the report.
		errStr := fmt.Errorf("failed to decode TON transaction: %w", err)

		return &DecodedCall{
			Address:         mcmsTx.To,
			Method:          errStr.Error(),
			ContractType:    contractType,
			ContractVersion: contractVersion,
		}, nil
	}

	namedArgs, err := toNamedFields(decodedOp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert decoded operation to named arguments: %w", err)
	}

	return &DecodedCall{
		Address:         mcmsTx.To,
		Method:          decodedOp.MethodName(),
		Inputs:          namedArgs,
		Outputs:         []NamedField{},
		ContractType:    contractType,
		ContractVersion: contractVersion,
	}, nil
}
