package decoder

import (
	"encoding/json"
	"strings"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const undecodedCallName = "[undecoded]"

// adaptTimelockProposal converts a ProposalReport produced by the experimental
// analyzer into a DecodedTimelockProposal.
func adaptTimelockProposal(
	report *experimentalanalyzer.ProposalReport,
	proposal *mcms.TimelockProposal,
) DecodedTimelockProposal {
	batches := make(DecodedBatchOperations, len(report.Batches))

	for i, batch := range report.Batches {
		batches[i] = &decodedBatchOperation{
			chainSelector: batch.ChainSelector,
			calls:         adaptBatchCalls(batch, proposal.Operations[i]),
		}
	}

	return &decodedTimelockProposal{batches: batches}
}

// adaptBatchCalls merges the decoded operations from a batch report with the
// raw transactions from the original proposal.
func adaptBatchCalls(
	batch experimentalanalyzer.BatchReport,
	rawBatch mcmstypes.BatchOperation,
) DecodedCalls {
	calls := make(DecodedCalls, 0, len(batch.Operations))

	for i, op := range batch.Operations {
		tx := rawBatch.Transactions[i]

		if len(op.Calls) == 0 {
			calls = append(calls, newUndecodedCall(tx))
			continue
		}

		for _, call := range op.Calls {
			calls = append(calls, newAdaptedCall(call, tx))
		}
	}

	return calls
}

// newAdaptedCall converts an experimental DecodedCall into the canonical
// decodedCall.
func newAdaptedCall(call *experimentalanalyzer.DecodedCall, tx mcmstypes.Transaction) *decodedCall {
	return &decodedCall{
		to:               call.Address,
		name:             cleanMethodName(call.Method),
		inputs:           adaptNamedFields(call.Inputs),
		outputs:          adaptNamedFields(call.Outputs),
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
		contractType:     call.ContractType,
		contractVersion:  call.ContractVersion,
	}
}

// newUndecodedCall creates a decodedCall for a transaction that the
// experimental analyzer could not decode.
func newUndecodedCall(tx mcmstypes.Transaction) *decodedCall {
	return &decodedCall{
		to:               tx.To,
		name:             undecodedCallName,
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
		contractType:     tx.ContractType,
	}
}

// adaptNamedFields converts experimental NamedField values into
// DecodedParameters.
func adaptNamedFields(fields []experimentalanalyzer.NamedField) DecodedParameters {
	if len(fields) == 0 {
		return nil
	}

	params := make(DecodedParameters, len(fields))
	for i, field := range fields {
		ptype := field.TypeName
		if ptype == "" && field.Value != nil {
			ptype = field.Value.GetType()
		}

		value := any(field.Value)
		if value == nil {
			value = field.RawValue
		}

		params[i] = &decodedParameter{
			name:     field.Name,
			ptype:    ptype,
			value:    value,
			rawValue: field.RawValue,
		}
	}

	return params
}

// cleanMethodName strips function signature down to just
// the method name.
func cleanMethodName(method string) string {
	m := strings.TrimSpace(method)
	m = strings.TrimPrefix(m, "function ")

	if idx := strings.Index(m, "("); idx > 0 {
		m = m[:idx]
	}

	return strings.TrimSpace(m)
}

var _ DecodedTimelockProposal = (*decodedTimelockProposal)(nil)

type decodedTimelockProposal struct {
	batches DecodedBatchOperations
}

func (d *decodedTimelockProposal) BatchOperations() DecodedBatchOperations {
	return d.batches
}

var _ DecodedBatchOperation = (*decodedBatchOperation)(nil)

type decodedBatchOperation struct {
	chainSelector uint64
	calls         DecodedCalls
}

func (d *decodedBatchOperation) ChainSelector() uint64 { return d.chainSelector }
func (d *decodedBatchOperation) Calls() DecodedCalls   { return d.calls }

var _ DecodedCall = (*decodedCall)(nil)

type decodedCall struct {
	to               string
	name             string
	inputs           DecodedParameters
	outputs          DecodedParameters
	data             []byte
	additionalFields json.RawMessage
	contractType     string
	contractVersion  string
}

func (d *decodedCall) To() string                        { return d.to }
func (d *decodedCall) Name() string                      { return d.name }
func (d *decodedCall) Inputs() DecodedParameters         { return d.inputs }
func (d *decodedCall) Outputs() DecodedParameters        { return d.outputs }
func (d *decodedCall) Data() []byte                      { return d.data }
func (d *decodedCall) AdditionalFields() json.RawMessage { return d.additionalFields }
func (d *decodedCall) ContractType() string              { return d.contractType }
func (d *decodedCall) ContractVersion() string           { return d.contractVersion }

var _ DecodedParameter = (*decodedParameter)(nil)

type decodedParameter struct {
	name     string
	ptype    string
	value    any
	rawValue any
}

func (d *decodedParameter) Name() string  { return d.name }
func (d *decodedParameter) Type() string  { return d.ptype }
func (d *decodedParameter) Value() any    { return d.value }
func (d *decodedParameter) RawValue() any { return d.rawValue }
