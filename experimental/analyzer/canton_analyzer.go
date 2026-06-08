package analyzer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmscantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeCantonTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	decoder := mcmscantonsdk.NewDecoder()
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeCantonTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze Canton transaction %d: %w", i, err)
		}
		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

// AnalyzeCantonTransaction decodes a single Canton MCMS transaction. Each transaction already
// describes the target call (Daml choice or factory deploy) via its AdditionalFields, so it is decoded directly
func AnalyzeCantonTransaction(ctx ProposalContext, decoder *mcmscantonsdk.Decoder, chainSelector uint64, mcmsTx types.Transaction) (*DecodedCall, error) {
	contractType, contractVersion := resolveContractInfo(ctx, chainSelector, mcmsTx)

	var additionalFields mcmscantonsdk.AdditionalFields
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Canton additional fields: %w", err)
	}

	// Pass the resolved contract type as the fallback contract-type key
	// The decoder prefers the entity name parsed from AdditionalFields.TargetTemplateID when present.
	decodedOp, err := decoder.Decode(mcmsTx, contractType)
	if err != nil {
		errStr := fmt.Errorf("failed to decode Canton transaction %q: %w", additionalFields.FunctionName, err)

		return &DecodedCall{
			Address:         mcmsTx.To,
			Method:          errStr.Error(),
			ContractType:    contractType,
			ContractVersion: contractVersion,
		}, nil
	}

	namedArgs, err := cantonToNamedFields(decodedOp)
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

// cantonToNamedFields is like toNamedFields but uses cantonFieldValue so that nested Daml records
// (returned as map[string]any by the Canton decoder's toDisplayArg) become StructField.
// Kept here rather than in utils.go to avoid leaking Canton-specific logic into the shared utility.
func cantonToNamedFields(decodedOp mcmssdk.DecodedOperation) ([]NamedField, error) {
	args := decodedOp.Args()
	keys := decodedOp.Keys()
	if len(keys) != len(args) {
		return nil, fmt.Errorf("mismatched keys and arguments length: %d keys, %d arguments", len(keys), len(args))
	}
	namedArgs := make([]NamedField, len(args))
	for i := range args {
		namedArgs[i] = NamedField{
			Name:     keys[i],
			Value:    cantonFieldValue(args[i]),
			RawValue: args[i],
		}
	}

	return namedArgs, nil
}

// cantonFieldValue is like getFieldValue but also converts map[string]any to StructField.
// Canton decoded args use map[string]any for nested Daml records (via toDisplayArg in the Canton
// decoder). This is Canton-scoped: other chains (e.g. TON) also return map[string]any in some
// decoded fields but rely on the default fmt.Sprintf("%v", ...) rendering via getFieldValue.
func cantonFieldValue(argument any) FieldValue {
	if m, ok := argument.(map[string]any); ok {
		return mapToStructField(m)
	}
	// For slices, recurse so nested maps within arrays are also converted.
	// Recurse into slices/arrays so nested maps within them are also converted — but not []byte,
	// which must fall through to getFieldValue so it renders as BytesField (hex-preview) rather
	// than ArrayField with individual byte elements.
	if _, isByteSlice := argument.([]byte); !isByteSlice {
		if kind := reflect.TypeOf(argument); kind != nil {
			if kind.Kind() == reflect.Array || kind.Kind() == reflect.Slice {
				array := ArrayField{}
				v := reflect.ValueOf(argument)
				for i := range v.Len() {
					array.Elements = append(array.Elements, cantonFieldValue(v.Index(i).Interface()))
				}

				return array
			}
		}
	}

	return getFieldValue(argument)
}

// mapToStructField converts a map[string]any to a StructField with sorted keys.
func mapToStructField(m map[string]any) StructField {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fields := make([]NamedField, 0, len(m))
	for _, k := range keys {
		fields = append(fields, NamedField{
			Name:  k,
			Value: cantonFieldValue(m[k]),
		})
	}

	return StructField{Fields: fields}
}
