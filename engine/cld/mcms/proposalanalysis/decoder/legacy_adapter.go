package decoder

import (
	"encoding/json"
	"strings"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// decodedTimelockProposal adapts legacy experimental/analyzer report to our interface
type decodedTimelockProposal struct {
	batches types.DecodedBatchOperations
}

func newDecodedTimelockProposal(
	report *experimentalanalyzer.ProposalReport,
	proposal *mcms.TimelockProposal,
) *decodedTimelockProposal {
	ops := proposal.Operations
	batches := make(types.DecodedBatchOperations, 0, len(report.Batches))

	for batchIdx, batch := range report.Batches {
		decodedBatch := &decodedBatchOperation{
			chainSelector: batch.ChainSelector,
			chainName:     batch.ChainName,
		}

		if batchIdx < len(ops) {
			for opIdx, op := range batch.Operations {
				if opIdx >= len(ops[batchIdx].Transactions) {
					continue
				}

				tx := ops[batchIdx].Transactions[opIdx]

				if len(op.Calls) == 0 {
					decodedBatch.calls = append(decodedBatch.calls, newUndecodedCall(tx))
					continue
				}

				for _, call := range op.Calls {
					decodedBatch.calls = append(decodedBatch.calls, newDecodedCall(call, tx))
				}
			}
		}

		batches = append(batches, decodedBatch)
	}

	return &decodedTimelockProposal{batches: batches}
}

func (d *decodedTimelockProposal) BatchOperations() types.DecodedBatchOperations { return d.batches }

type decodedBatchOperation struct {
	chainSelector uint64
	chainName     string
	calls         types.DecodedCalls
}

func (d *decodedBatchOperation) ChainSelector() uint64 { return d.chainSelector }
func (d *decodedBatchOperation) ChainName() string     { return d.chainName }
func (d *decodedBatchOperation) Calls() types.DecodedCalls {
	return d.calls
}

type decodedCall struct {
	to               string
	name             string
	inputs           types.DecodedParameters
	outputs          types.DecodedParameters
	data             []byte
	additionalFields json.RawMessage
	contractType     string
	contractVersion  string
}

func newDecodedCall(call *experimentalanalyzer.DecodedCall, tx mcmstypes.Transaction) *decodedCall {
	return &decodedCall{
		to:               call.Address,
		name:             methodName(call.Method),
		inputs:           convertNamedFields(call.Inputs),
		outputs:          convertNamedFields(call.Outputs),
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
		contractType:     tx.ContractType,
	}
}

func methodName(method string) string {
	m := strings.TrimSpace(method)
	m = strings.TrimPrefix(m, "function ")
	if idx := strings.Index(m, "("); idx > 0 {
		m = m[:idx]
	}
	return strings.TrimSpace(m)
}

func newUndecodedCall(tx mcmstypes.Transaction) *decodedCall {
	return &decodedCall{
		to:               tx.To,
		name:             "[undecoded]",
		inputs:           nil,
		outputs:          nil,
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
		contractType:     tx.ContractType,
	}
}

func (d *decodedCall) To() string                        { return d.to }
func (d *decodedCall) Name() string                      { return d.name }
func (d *decodedCall) Inputs() types.DecodedParameters   { return d.inputs }
func (d *decodedCall) Outputs() types.DecodedParameters  { return d.outputs }
func (d *decodedCall) Data() []byte                      { return d.data }
func (d *decodedCall) AdditionalFields() json.RawMessage { return d.additionalFields }
func (d *decodedCall) ContractType() string              { return d.contractType }
func (d *decodedCall) ContractVersion() string           { return d.contractVersion }

type decodedParameter struct {
	name    string
	value   any
	display any
	ptype   string
}

func (d *decodedParameter) Name() string { return d.name }
func (d *decodedParameter) Value() any   { return d.value }
func (d *decodedParameter) DisplayValue() any {
	return d.display
}
func (d *decodedParameter) Type() string { return d.ptype }

// convertNamedFields converts legacy experimental NamedFields to DecodedParameters
func convertNamedFields(fields []experimentalanalyzer.NamedField) types.DecodedParameters {
	params := make(types.DecodedParameters, len(fields))
	for i, field := range fields {
		paramType := ""
		if field.Value != nil {
			paramType = field.Value.GetType()
		}
		params[i] = &decodedParameter{
			name:    field.Name,
			value:   field.RawValue,
			display: field.Value,
			ptype:   paramType,
		}
	}
	return params
}
