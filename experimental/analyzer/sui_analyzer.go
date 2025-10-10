package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/chainlink-sui/bindings/generated"
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"
)

func AnalyzeSuiTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*proposalutils.DecodedCall, error) {
	decoder := mcmssuisdk.NewDecoder()
	decodedTxs := make([]*proposalutils.DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeSuiTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Sui transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeSuiTransaction(ctx ProposalContext, decoder *mcmssuisdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*proposalutils.DecodedCall, error) {
	var additionalFields mcmssuisdk.AdditionalFields
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Sui additional fields: %w", err)
	}

	functionInfo := generated.FunctionInfoByModule[additionalFields.ModuleName]
	decodedOp, err := decoder.Decode(mcmsTx, functionInfo)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single missing method
		errStr := fmt.Errorf("failed to decode Sui transaction: %w", err)

		return &proposalutils.DecodedCall{
			Address: mcmsTx.To,
			Method:  errStr.Error(),
		}, nil
	}
	namedArgs, err := toNamedArguments(decodedOp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert decoded operation to named arguments: %w", err)
	}

	return &proposalutils.DecodedCall{
		Address: mcmsTx.To,
		Method:  decodedOp.MethodName(),
		Inputs:  namedArgs,
		Outputs: []proposalutils.NamedArgument{},
	}, nil
}
