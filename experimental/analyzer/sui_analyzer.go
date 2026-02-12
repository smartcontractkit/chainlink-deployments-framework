package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/chainlink-sui/bindings/generated"
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeSuiTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	decoder := mcmssuisdk.NewDecoder()
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeSuiTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Sui transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeSuiTransaction(ctx ProposalContext, decoder *mcmssuisdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	var additionalFields mcmssuisdk.AdditionalFields
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Sui additional fields: %w", err)
	}

	// Return the method name directly for MCMS transactions, since the inner transactions will be decoded separately
	if additionalFields.ModuleName == "mcms" {
		methodName := fmt.Sprintf("%s::%s", additionalFields.ModuleName, additionalFields.Function)
		return &DecodedCall{
			Address: mcmsTx.To,
			Method:  methodName,
			Inputs:  []NamedField{},
			Outputs: []NamedField{},
		}, nil
	}

	functionInfo, ok := generated.FunctionInfoByModule[additionalFields.ModuleName]
	if !ok {
		// Don't return an error to not block the whole proposal decoding because of a single missing method
		errStr := fmt.Errorf("no function info found for module %s on chain selector %d", additionalFields.ModuleName, chainSelector)

		return &DecodedCall{
			Address: mcmsTx.To,
			Method:  errStr.Error(),
		}, nil
	}

	decodedOp, err := decoder.Decode(mcmsTx, functionInfo)
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
