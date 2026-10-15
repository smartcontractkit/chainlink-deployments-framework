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

type aptosFunctionInfoLookup func(packageName, moduleName, function string) (string, error)

// Generic so callers can pass a bindings GetFunctionInfo from another module without a converter.
func newAptosFunctionInfoLookup[T ~[]E, E any](getFunctionInfo func(packageModuleFunction ...string) T) aptosFunctionInfoLookup {
	return func(packageName, moduleName, function string) (string, error) {
		found := getFunctionInfo(packageName, moduleName, function)
		if len(found) == 0 {
			return "", nil
		}
		raw, err := json.Marshal(found)
		if err != nil {
			return "", fmt.Errorf("failed to marshal Aptos function info: %w", err)
		}
		infos, err := aptosbind.ParseFunctionInfo(string(raw))
		if err != nil {
			return "", fmt.Errorf("failed to parse Aptos function info: %w", err)
		}
		for _, info := range infos {
			if info.Package != packageName || info.Module != moduleName || info.Name != function {
				return "", fmt.Errorf("aptos function info lookup returned an entry that does not match %s::%s::%s", packageName, moduleName, function)
			}
		}

		return infos.String(), nil
	}
}

// getAptosFunctionInfo tries the context's lookups in order, then the chainlink-aptos bindings.
func getAptosFunctionInfo(ctx ProposalContext, packageName, moduleName, function string) (string, error) {
	var lookups []aptosFunctionInfoLookup
	if c, ok := ctx.(interface {
		aptosFunctionInfoLookups() []aptosFunctionInfoLookup
	}); ok {
		lookups = c.aptosFunctionInfoLookups()
	}

	var errs []error
	for _, lookup := range lookups {
		info, err := lookup(packageName, moduleName, function)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if info != "" {
			return info, nil
		}
	}

	// chainlink-aptos returns a zero entry for an unknown function, so match the key.
	fallback := bindings.GetFunctionInfo(packageName, moduleName, function)
	resolved := slices.ContainsFunc(fallback, func(info aptosbind.FunctionInfo) bool {
		return info.Package == packageName && info.Module == moduleName && info.Name == function
	})
	if !resolved && len(errs) > 0 {
		return "", errors.Join(errs...)
	}

	return fallback.String(), nil
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
