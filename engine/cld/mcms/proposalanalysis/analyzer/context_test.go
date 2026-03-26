package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

func TestNewExecutionContextNode(t *testing.T) {
	t.Parallel()

	domain := cldfdomain.NewDomain("/tmp/domains", "mcms")
	blockChains := chain.NewBlockChains(nil)
	dataStore := datastore.NewMemoryDataStore().Seal()

	ctx := NewExecutionContextNode(domain, "staging", blockChains, dataStore)

	require.Equal(t, domain, ctx.Domain())
	require.Equal(t, "staging", ctx.EnvironmentName())
	require.Equal(t, blockChains, ctx.BlockChains())
	require.Equal(t, dataStore, ctx.DataStore())
}

func TestNewBatchOperationAnalyzerContextNode(t *testing.T) {
	t.Parallel()

	proposal := &testDecodedProposal{}

	ctx := NewBatchOperationAnalyzerContextNode(proposal)

	require.Equal(t, proposal, ctx.Proposal())
}

func TestNewCallAnalyzerContextNode(t *testing.T) {
	t.Parallel()

	proposal := &testDecodedProposal{}
	operation := &testDecodedBatchOperation{}

	ctx := NewCallAnalyzerContextNode(proposal, operation)

	require.Equal(t, proposal, ctx.Proposal())
	require.Equal(t, operation, ctx.BatchOperation())
}

func TestNewParameterAnalyzerContextNode(t *testing.T) {
	t.Parallel()

	proposal := &testDecodedProposal{}
	operation := &testDecodedBatchOperation{}
	call := &testDecodedCall{}

	ctx := NewParameterAnalyzerContextNode(proposal, operation, call)

	require.Equal(t, proposal, ctx.Proposal())
	require.Equal(t, operation, ctx.BatchOperation())
	require.Equal(t, call, ctx.Call())
}

var (
	_ decoder.DecodedTimelockProposal = (*testDecodedProposal)(nil)
	_ decoder.DecodedBatchOperation   = (*testDecodedBatchOperation)(nil)
	_ decoder.DecodedCall             = (*testDecodedCall)(nil)
)

type testDecodedProposal struct{}

func (p *testDecodedProposal) BatchOperations() decoder.DecodedBatchOperations {
	return nil
}

type testDecodedBatchOperation struct{}

func (b *testDecodedBatchOperation) ChainSelector() uint64 {
	return 0
}

func (b *testDecodedBatchOperation) Calls() decoder.DecodedCalls {
	return nil
}

type testDecodedCall struct{}

func (c *testDecodedCall) To() string {
	return "0x0"
}

func (c *testDecodedCall) Name() string {
	return "noop"
}

func (c *testDecodedCall) Inputs() decoder.DecodedParameters {
	return nil
}

func (c *testDecodedCall) Outputs() decoder.DecodedParameters {
	return nil
}

func (c *testDecodedCall) Data() []byte {
	return nil
}

func (c *testDecodedCall) AdditionalFields() json.RawMessage {
	return nil
}

func (c *testDecodedCall) ContractType() string {
	return ""
}

func (c *testDecodedCall) ContractVersion() string {
	return ""
}
