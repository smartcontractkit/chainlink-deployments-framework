package decoder

import (
	"encoding/json"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// decodedTimelockProposal adapts legacy experimental/analyzer report to our interface
type decodedTimelockProposal struct {
	report *experimentalanalyzer.ProposalReport
}

func (d *decodedTimelockProposal) BatchOperations() types.DecodedBatchOperations {
	batches := make(types.DecodedBatchOperations, len(d.report.Batches))
	for i, batch := range d.report.Batches {
		batches[i] = &decodedBatchOperation{
			batch: batch,
		}
	}
	return batches
}

// decodedBatchOperation adapts legacy experimental batch report
type decodedBatchOperation struct {
	batch experimentalanalyzer.BatchReport
}

func (d *decodedBatchOperation) ChainSelector() uint64 {
	return d.batch.ChainSelector
}

func (d *decodedBatchOperation) Calls() types.DecodedCalls {
	// Flatten all calls from all operations in the batch
	var allCalls types.DecodedCalls
	for _, op := range d.batch.Operations {
		for _, call := range op.Calls {
			allCalls = append(allCalls, &decodedCall{call: call})
		}
	}
	return allCalls
}

// decodedCall adapts legacy experimental decoded call
type decodedCall struct {
	call *experimentalanalyzer.DecodedCall
}

func (d *decodedCall) To() string {
	return d.call.Address
}

func (d *decodedCall) Name() string {
	return d.call.Method
}

func (d *decodedCall) Inputs() types.DecodedParameters {
	return convertNamedFields(d.call.Inputs)
}

func (d *decodedCall) Outputs() types.DecodedParameters {
	return convertNamedFields(d.call.Outputs)
}

func (d *decodedCall) Data() []byte {
	// Not directly available in legacy experimental analyzer, return empty
	return []byte{}
}

func (d *decodedCall) AdditionalFields() json.RawMessage {
	// Not directly available in legacy experimental analyzer, return empty
	return json.RawMessage("{}")
}

// decodedParameter adapts legacy experimental named field
type decodedParameter struct {
	field experimentalanalyzer.NamedField
}

func (d *decodedParameter) Name() string {
	return d.field.Name
}

func (d *decodedParameter) Value() any {
	return convertFieldValue(d.field.Value)
}

// convertNamedFields converts legacy experimental NamedFields to DecodedParameters
func convertNamedFields(fields []experimentalanalyzer.NamedField) types.DecodedParameters {
	params := make(types.DecodedParameters, len(fields))
	for i, field := range fields {
		params[i] = &decodedParameter{field: field}
	}
	return params
}

// convertFieldValue recursively converts legacy experimental FieldValue to simple types
func convertFieldValue(fv experimentalanalyzer.FieldValue) any {
	if fv == nil {
		return nil
	}

	// Try to render the field value to a string
	// The legacy experimental analyzer's FieldValue interface doesn't expose internal structure,
	// so we use the rendering method
	return fv.GetType()
}
