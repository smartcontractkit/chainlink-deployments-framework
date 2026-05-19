package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
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
// On decode failure, this function returns a DecodedCall with the error in the Method field
// instead of returning an error. This allows the proposal to continue processing even if
// a single transaction fails to decode.
func AnalyzeTONTransaction(ctx ProposalContext, decoder sdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	contractType, contractVersion := resolveContractInfo(ctx, chainSelector, mcmsTx)

	var typeErr string
	fullyQualifiedName := func() string {
		var additionalFields ton.AdditionalFields
		if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
			typeErr = fmt.Sprintf("failed to unmarshal TON additional fields: %s", err)
			return ""
		}

		fullyQualifiedName := string(additionalFields.ContractTypeFull)
		// If ContractVersion is provided, append it to the fully qualified name to ensure the decoder uses the correct version.
		// If it is skipped, the decoder will use the latest version available for the contract type.
		// Note: we don't use contractVersion from resolveContractInfo because that only represents the short type used by the datastore.
		if mcmsTx.ContractVersion != nil {
			fullyQualifiedName += "@" + mcmsTx.ContractVersion.String()
		}

		return fullyQualifiedName
	}()

	decodedOp, err := decoder.Decode(mcmsTx, fullyQualifiedName)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single transaction decode failure.
		// Instead, put the error message in the Method field so it's visible in the report.
		errStr := "failed to decode TON transaction: " + err.Error()
		if typeErr != "" {
			errStr += "; additionally" + typeErr
		}

		return &DecodedCall{
			Address:         mcmsTx.To,
			Method:          errStr,
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
