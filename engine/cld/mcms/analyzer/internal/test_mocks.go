package internal

import (
	"encoding/json"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

// Mock implementations for testing - shared across test files

type mockDecodedParameter struct {
	name  string
	ptype string
	value any
}

func (m mockDecodedParameter) Name() string { return m.name }
func (m mockDecodedParameter) Type() string { return m.ptype }
func (m mockDecodedParameter) Value() any   { return m.value }

type mockDecodedCall struct {
	name    string
	inputs  analyzer.DecodedParameters
	outputs analyzer.DecodedParameters
}

func (m mockDecodedCall) Name() string                        { return m.name }
func (m mockDecodedCall) ContractType() string                { return "" }
func (m mockDecodedCall) ContractVersion() string             { return "" }
func (m mockDecodedCall) To() string                          { return "" }
func (m mockDecodedCall) Inputs() analyzer.DecodedParameters  { return m.inputs }
func (m mockDecodedCall) Outputs() analyzer.DecodedParameters { return m.outputs }
func (m mockDecodedCall) Data() []byte                        { return nil }
func (m mockDecodedCall) AdditionalFields() json.RawMessage   { return nil }

type mockDecodedBatchOperation struct {
	calls analyzer.DecodedCalls
}

func (m mockDecodedBatchOperation) ChainSelector() uint64        { return 0 }
func (m mockDecodedBatchOperation) Calls() analyzer.DecodedCalls { return m.calls }

type mockDecodedTimelockProposal struct {
	batchOps analyzer.DecodedBatchOperations
}

func (m mockDecodedTimelockProposal) BatchOperations() analyzer.DecodedBatchOperations {
	return m.batchOps
}
