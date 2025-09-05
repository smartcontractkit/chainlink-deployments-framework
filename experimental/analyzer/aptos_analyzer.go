package analyzer

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/smartcontractkit/chainlink-aptos/bindings"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
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

func toNamedArguments(decodedOp mcmssdk.DecodedOperation) ([]proposalutils.NamedArgument, error) {
	args := decodedOp.Args()
	keys := decodedOp.Keys()
	if len(keys) != len(args) {
		return nil, fmt.Errorf("mismatched keys and arguments length: %d keys, %d arguments", len(keys), len(args))
	}
	namedArgs := make([]proposalutils.NamedArgument, len(args))
	for i := range args {
		namedArgs[i] = proposalutils.NamedArgument{
			Name:  keys[i],
			Value: getArgument(args[i]),
		}
	}

	return namedArgs, nil
}

func getArgument(argument any) proposalutils.Argument {
	var value proposalutils.Argument

	switch arg := argument.(type) {
	// Pretty-print byte arrays and addresses
	case []byte:
		value = proposalutils.BytesArgument{Value: arg}
	case aptos.AccountAddress:
		value = proposalutils.AddressArgument{Value: arg.StringLong()}
	case *aptos.AccountAddress:
		value = proposalutils.AddressArgument{Value: arg.StringLong()}
	default:
		//nolint:exhaustive // default case covers everything else
		switch reflect.TypeOf(arg).Kind() {
		// If the argument is a slice or array, iterate over every element individually
		case reflect.Array, reflect.Slice:
			array := proposalutils.ArrayArgument{}
			v := reflect.ValueOf(arg)
			for i := range v.Len() {
				array.Elements = append(array.Elements, getArgument(v.Index(i).Interface()))
			}
			value = array
		default:
			// Simply print the argument as-is
			value = proposalutils.SimpleArgument{Value: fmt.Sprintf("%v", arg)}
		}
	}

	return value
}
