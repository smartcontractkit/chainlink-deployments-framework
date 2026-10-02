package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// stellarAdditionalFields is the Stellar arm of a transaction's AdditionalFields, as stamped by
// the MCMS Stellar encoder. Both members are pointers so an absent field is distinguishable from
// a zero value.
type stellarAdditionalFields struct {
	Family          *string `json:"family"`
	EncodingVersion *uint32 `json:"encodingVersion"`
}

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

	if reason := stellarUnsupportedEncoding(mcmsTx.AdditionalFields); reason != "" {
		return &DecodedCall{
			Address:         mcmsTx.To,
			Method:          reason,
			ContractType:    contractType,
			ContractVersion: contractVersion,
		}, nil
	}

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

	names := stellarArgNames(function, len(args))
	inputs := make([]NamedField, 0, len(args))
	for i, arg := range args {
		inputs = append(inputs, NamedField{
			Name:     names[i],
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

// stellarTimelockArgNames gives the parameter names of the RBACTimelock entrypoints, taken from
// contracts/timelock/src/lib.rs. Naming them is not cosmetic: the UPF timelock conversion replaces
// FunctionArgs["calls"] with the expanded batch (upf.go), so an outer call whose batch argument is
// named argN keeps the raw encoded batch alongside a second, expanded "calls" entry instead of
// replacing it. The framework already hard-codes these function names in the timelock batch
// checkers, so their signatures are equally fixed knowledge.
var stellarTimelockArgNames = map[string][]string{
	"schedule_batch":         {"caller", "calls", "predecessor", "salt", "delay"},
	"bypasser_execute_batch": {"caller", "calls"},
	"execute_batch":          {"calls", "predecessor", "salt"},
	"cancel":                 {"caller", "id"},
}

// stellarArgNames returns the field names to use for count arguments of function: the known
// parameter names for the timelock entrypoints, and positional arg0, arg1, ... otherwise.
//
// A known function whose argument count does not match its recorded signature falls back to
// positional names, so a future contract revision mislabels nothing.
func stellarArgNames(function string, count int) []string {
	if known, ok := stellarTimelockArgNames[function]; ok && len(known) == count {
		return known
	}

	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("arg%d", i)
	}

	return names
}

// stellarUnsupportedEncoding reports why a transaction must not be decoded with this analyzer's
// assumptions, or "" when decoding may proceed.
//
// AdditionalFields is advisory: a missing or unparseable value falls back to best-effort decoding,
// since a hand-written proposal may omit it. But an explicit family or encodingVersion this
// analyzer does not implement must never be decoded anyway — a future wire format could still
// parse as an ScVal vector and show a reviewer a confident, wrong method name.
func stellarUnsupportedEncoding(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var fields stellarAdditionalFields
	if err := json.Unmarshal(raw, &fields); err != nil {
		return ""
	}

	if fields.Family != nil && *fields.Family != chainsel.FamilyStellar {
		return fmt.Sprintf("unexpected transaction family %q on a Stellar chain", *fields.Family)
	}

	if fields.EncodingVersion != nil && *fields.EncodingVersion != bindings.SorobanInvokeEncodingVersion {
		return fmt.Sprintf(
			"unsupported Stellar MCMS encoding version %d: this analyzer decodes version %d",
			*fields.EncodingVersion, bindings.SorobanInvokeEncodingVersion,
		)
	}

	return ""
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
			name := stellarUniqueFieldName(stellarMapKeyName(entry.Key, i), i, used)
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

// stellarUniqueFieldName returns base when it is unused, and otherwise probes base#<n> until it
// finds a free name. Probing is required rather than a single base#<index> attempt: a suffixed
// candidate can itself collide with another key's name, as with the keys U32(1),
// Symbol("U32(1)#2") and Symbol("U32(1)"), where the third would otherwise reuse the second's
// name. UPF serializes a StructField into a map[string]any keyed by NamedField.Name, so any
// collision silently drops an entry.
//
// The loop terminates: each rejected candidate matches a distinct already-used name, so it runs
// at most len(used)+1 times.
func stellarUniqueFieldName(base string, index int, used map[string]struct{}) string {
	if _, taken := used[base]; !taken {
		return base
	}

	for n := index; ; n++ {
		candidate := fmt.Sprintf("%s#%d", base, n)
		if _, taken := used[candidate]; !taken {
			return candidate
		}
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
