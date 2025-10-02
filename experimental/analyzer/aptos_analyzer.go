package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/chainlink-aptos/bindings"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"
)

func AnalyzeAptosTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*proposalutils.DecodedCall, error) {
	decoder := mcmsaptossdk.NewDecoder()
	decodedTxs := make([]*proposalutils.DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeAptosTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Aptos transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeAptosTransaction(ctx ProposalContext, decoder *mcmsaptossdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*proposalutils.DecodedCall, error) {
	var additionalFields mcmsaptossdk.AdditionalFields
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Aptos additional fields: %w", err)
	}
	functionInfo := bindings.GetFunctionInfo(additionalFields.PackageName, additionalFields.ModuleName, additionalFields.Function)
	decodedOp, err := decoder.Decode(mcmsTx, functionInfo.String())
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single missing method
		errStr := fmt.Errorf("failed to decode Aptos transaction: %w", err)

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
