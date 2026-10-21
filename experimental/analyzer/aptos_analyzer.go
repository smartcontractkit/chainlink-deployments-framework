package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/smartcontractkit/chainlink-aptos/bindings"
	aptosbind "github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

// AptosFunctionInfoLookup returns the function info JSON of one Move function, in the
// chainlink-aptos bindings format, or "" when it does not know the function.
type AptosFunctionInfoLookup func(packageName, moduleName, function string) string

// getAptosFunctionInfo tries the context's lookups in order, then the chainlink-aptos bindings.
func getAptosFunctionInfo(ctx ProposalContext, packageName, moduleName, function string) (string, error) {
	var lookups []AptosFunctionInfoLookup
	if c, ok := ctx.(interface {
		aptosFunctionInfoLookups() []AptosFunctionInfoLookup
	}); ok {
		lookups = c.aptosFunctionInfoLookups()
	}

	var errs []error
	for _, lookup := range lookups {
		info := lookup(packageName, moduleName, function)
		if info == "" {
			continue
		}
		infos, err := aptosbind.ParseFunctionInfo(info)
		if err != nil {
			errs = append(errs, fmt.Errorf("aptos function info lookup for %s::%s::%s: %w", packageName, moduleName, function, err))
			continue
		}
		if len(infos) == 0 {
			continue
		}
		if hasFunctionInfo(infos, packageName, moduleName, function) {
			return info, nil
		}
		errs = append(errs, fmt.Errorf("aptos function info lookup returned entries that do not name %s::%s::%s", packageName, moduleName, function))
	}

	// chainlink-aptos returns a zero entry for an unknown function, so match the key.
	fallback := bindings.GetFunctionInfo(packageName, moduleName, function)
	if !hasFunctionInfo(fallback, packageName, moduleName, function) && len(errs) > 0 {
		return "", errors.Join(errs...)
	}

	return fallback.String(), nil
}

func hasFunctionInfo(infos aptosbind.FunctionInfos, packageName, moduleName, function string) bool {
	return slices.ContainsFunc(infos, func(info aptosbind.FunctionInfo) bool {
		return info.Package == packageName && info.Module == moduleName && info.Name == function
	})
}

func AnalyzeAptosTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	decoder := mcmsaptossdk.NewDecoder()
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeAptosTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Aptos transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeAptosTransaction(ctx ProposalContext, decoder *mcmsaptossdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	contractType, contractVersion := resolveContractInfo(ctx, chainSelector, mcmsTx)

	var additionalFields mcmsaptossdk.AdditionalFields
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Aptos additional fields: %w", err)
	}
	decodedOp, err := decodeAptosTransaction(ctx, decoder, mcmsTx, additionalFields)
	if err != nil {
		// Don't return an error to not block the whole proposal decoding because of a single missing method
		errStr := fmt.Errorf("failed to decode Aptos transaction: %w", err)

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

func decodeAptosTransaction(ctx ProposalContext, decoder *mcmsaptossdk.Decoder, mcmsTx types.Transaction, fields mcmsaptossdk.AdditionalFields) (sdk.DecodedOperation, error) {
	functionInfo, err := getAptosFunctionInfo(ctx, fields.PackageName, fields.ModuleName, fields.Function)
	if err != nil {
		return nil, err
	}

	return decoder.Decode(mcmsTx, functionInfo)
}
