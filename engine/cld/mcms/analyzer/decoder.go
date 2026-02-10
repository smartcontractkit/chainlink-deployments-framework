package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	expanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type decodedCall struct {
	contractType     string
	address          string
	methodName       string
	inputs           []DecodedParameter
	data             []byte
	additionalFields json.RawMessage
}

func (d *decodedCall) ContractType() string              { return d.contractType }
func (d *decodedCall) Name() string                      { return d.methodName }
func (d *decodedCall) To() string                        { return d.address }
func (d *decodedCall) Inputs() []DecodedParameter        { return d.inputs }
func (d *decodedCall) Data() []byte                      { return d.data }
func (d *decodedCall) AdditionalFields() json.RawMessage { return d.additionalFields }

func decodeProposal(
	ctx context.Context,
	ectx *ExecutionContext,
	proposal *mcms.TimelockProposal,
) (*DecodedTimelockProposal, error) {
	if ectx.proposalCtx == nil {
		return nil, fmt.Errorf("ProposalContext is required for decoding")
	}

	report, err := expanalyzer.BuildTimelockReport(ctx, ectx.proposalCtx, ectx.Env, proposal)
	if err != nil {
		return nil, fmt.Errorf("decode proposal: %w", err)
	}

	decoded := &DecodedTimelockProposal{}

	for batchIdx, batch := range report.Batches {
		chainSel := batch.ChainSelector
		decodedBatch := DecodedBatchOperation{
			ChainSelector: chainSel,
			ChainName:     batch.ChainName,
		}

		for opIdx, op := range batch.Operations {
			tx := proposal.Operations[batchIdx].Transactions[opIdx]

			if len(op.Calls) == 0 {
				decodedBatch.Calls = append(decodedBatch.Calls, newUndecodedCall(tx))

				continue
			}

			for _, expDecoded := range op.Calls {
				call := convertDecodedCall(expDecoded, tx)
				decodedBatch.Calls = append(decodedBatch.Calls, call)
			}
		}

		decoded.BatchOperations = append(decoded.BatchOperations, decodedBatch)
	}

	return decoded, nil
}

func convertDecodedCall(
	expDecoded *expanalyzer.DecodedCall,
	tx mcmstypes.Transaction,
) DecodedCall {
	inputs := make([]DecodedParameter, len(expDecoded.Inputs))

	for i, field := range expDecoded.Inputs {
		inputs[i] = DecodedParameter{
			Name:         field.Name,
			Value:        field.RawValue,
			DisplayValue: field.Value,
		}
	}

	return &decodedCall{
		contractType:     tx.ContractType,
		address:          tx.To,
		methodName:       expDecoded.MethodName(),
		inputs:           inputs,
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
	}
}

func newUndecodedCall(tx mcmstypes.Transaction) DecodedCall {
	log.Printf("WARN: undecoded call to %s on contract type %q", tx.To, tx.ContractType)

	return &decodedCall{
		contractType:     tx.ContractType,
		address:          tx.To,
		methodName:       "[undecoded]",
		data:             tx.Data,
		additionalFields: tx.AdditionalFields,
	}
}
