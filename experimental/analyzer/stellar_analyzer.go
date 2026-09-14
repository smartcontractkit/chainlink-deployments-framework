package analyzer

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// AnalyzeStellarTransactions decodes a slice of Stellar transactions and returns their decoded
// representations.
func AnalyzeStellarTransactions(
	ctx ProposalContext, chainSelector uint64, txs []types.Transaction,
) ([]*DecodedCall, error) {
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeStellarTransaction(ctx, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Stellar transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

// AnalyzeStellarTransaction decodes a single Stellar transaction.
//
// The transaction data is the MCMS Soroban invoke payload: XDR for
// ScVal::Vec([Symbol(function), ...args]), which the Stellar MCMS proposal transformer splits
// into StellarOp.function / StellarOp.args_xdr before submission. Decoding is driven entirely by
// the payload, so no per-contract ABI registry is needed; the trade-off is that arguments are
// positional (arg0, arg1, ...) rather than named.
//
// On decode failure, this function returns a DecodedCall with the error in the Method field
// instead of returning an error. This allows the proposal to continue processing even if
// a single transaction fails to decode.
func AnalyzeStellarTransaction(
	ctx ProposalContext, chainSelector uint64, mcmsTx types.Transaction,
) (*DecodedCall, error) {
	contractType, contractVersion := resolveContractInfo(ctx, chainSelector, mcmsTx)

	function, args, err := bindings.DecodeSorobanInvokePayload(mcmsTx.Data)
	if err != nil {
		//nolint:nilerr // Surface the failure in the report rather than blocking the whole proposal.
		return &DecodedCall{
			Address:         mcmsTx.To,
			Method:          "failed to decode Stellar transaction: " + err.Error(),
			ContractType:    contractType,
			ContractVersion: contractVersion,
		}, nil
	}

	inputs := make([]NamedField, 0, len(args))
	for i, arg := range args {
		inputs = append(inputs, NamedField{
			Name:     fmt.Sprintf("arg%d", i),
			TypeName: stellarScValTypeName(arg),
			Value:    stellarScValField(arg),
			RawValue: arg,
		})
	}

	return &DecodedCall{
		Address:         mcmsTx.To,
		Method:          function,
		Inputs:          inputs,
		Outputs:         []NamedField{},
		ContractType:    contractType,
		ContractVersion: contractVersion,
	}, nil
}

// stellarScValField converts a Soroban value into the field representation used by the renderers.
// Vectors and maps recurse so nested arguments stay inspectable.
func stellarScValField(val xdr.ScVal) FieldValue {
	//nolint:exhaustive // default case renders the remaining scalar types via ScVal.String
	switch val.Type {
	case xdr.ScValTypeScvAddress:
		addr, err := val.Address.String()
		if err != nil {
			return SimpleField{Value: fmt.Sprintf("invalid address: %s", err)}
		}

		return AddressField{Value: addr}

	case xdr.ScValTypeScvBytes:
		if val.Bytes == nil {
			return BytesField{}
		}

		return BytesField{Value: []byte(*val.Bytes)}

	case xdr.ScValTypeScvVec:
		if val.Vec == nil || *val.Vec == nil {
			return ArrayField{}
		}
		elements := make([]FieldValue, 0, len(**val.Vec))
		for _, element := range **val.Vec {
			elements = append(elements, stellarScValField(element))
		}

		return ArrayField{Elements: elements}

	case xdr.ScValTypeScvMap:
		if val.Map == nil || *val.Map == nil {
			return StructField{}
		}
		entries := **val.Map
		fields := make([]NamedField, 0, len(entries))
		used := make(map[string]struct{}, len(entries))
		for i, entry := range entries {
			name := stellarMapKeyName(entry.Key, i)
			if _, taken := used[name]; taken {
				// Distinct keys must never share a name: UPF serializes a StructField into a
				// map[string]any keyed by NamedField.Name, so a collision drops an entry.
				name = fmt.Sprintf("%s#%d", name, i)
			}
			used[name] = struct{}{}

			fields = append(fields, NamedField{
				Name:     name,
				TypeName: stellarScValTypeName(entry.Val),
				Value:    stellarScValField(entry.Val),
				RawValue: entry.Val,
			})
		}

		return StructField{Fields: fields}

	default:
		// ScVal.String covers the scalar types, including the 128/256-bit integers.
		return SimpleField{Value: val.String()}
	}
}

// stellarMapKeyName renders a Soroban map key as a field name.
//
// Map keys are arbitrary ScVals, not just symbols. Symbol keys — what contract-generated structs
// use — keep their bare text so reports stay readable. Every other key type is qualified with its
// type, because ScVal.String omits it: U32(1), String("1") and Symbol("1") all render as "1" and
// would otherwise collide into a single field name.
func stellarMapKeyName(key xdr.ScVal, index int) string {
	if sym, ok := key.GetSym(); ok {
		if name := string(sym); name != "" {
			return name
		}

		return fmt.Sprintf("key%d", index)
	}

	rendered := key.String()
	if rendered == "" {
		return fmt.Sprintf("key%d", index)
	}

	return fmt.Sprintf("%s(%s)", stellarScValTypeName(key), rendered)
}

// stellarScValTypeName renders the Soroban type of a value without the XDR enum prefix, so
// reports show "Address" rather than "ScValTypeScvAddress".
func stellarScValTypeName(val xdr.ScVal) string {
	const prefix = "ScValTypeScv"

	name := val.Type.String()
	if len(name) > len(prefix) && name[:len(prefix)] == prefix {
		return name[len(prefix):]
	}

	return name
}
