package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAnalyzedProposalNode(t *testing.T) {
	t.Parallel()

	batch := NewAnalyzedBatchOperationNode(111, nil)
	node := NewAnalyzedProposalNode(AnalyzedBatchOperations{batch})

	require.Len(t, node.BatchOperations(), 1)
	require.Equal(t, uint64(111), node.BatchOperations()[0].ChainSelector())
}

func TestNewAnalyzedBatchOperationNode(t *testing.T) {
	t.Parallel()

	call := NewAnalyzedCallNode("0xabc", "doThing", nil, nil, []byte{0x01}, "Foo", "1.0.0")
	node := NewAnalyzedBatchOperationNode(5009297550715157269, AnalyzedCalls{call})

	require.Equal(t, uint64(5009297550715157269), node.ChainSelector())
	require.Len(t, node.Calls(), 1)
	require.Equal(t, "doThing", node.Calls()[0].Name())
}

func TestNewAnalyzedCallNode(t *testing.T) {
	t.Parallel()

	in := NewAnalyzedParameterNode("amount", "uint256", 42)
	out := NewAnalyzedParameterNode("ok", "bool", true)
	node := NewAnalyzedCallNode(
		"0xabc",
		"transfer",
		AnalyzedParameters{in},
		AnalyzedParameters{out},
		[]byte{0xaa, 0xbb},
		"Token",
		"v1",
	)

	require.Equal(t, "0xabc", node.To())
	require.Equal(t, "transfer", node.Name())
	require.Len(t, node.Inputs(), 1)
	require.Len(t, node.Outputs(), 1)
	require.Equal(t, []byte{0xaa, 0xbb}, node.Data())
	require.Equal(t, "Token", node.ContractType())
	require.Equal(t, "v1", node.ContractVersion())
	require.Equal(t, map[string]any{}, node.AdditionalFields())
}

func TestNewAnalyzedParameterNode(t *testing.T) {
	t.Parallel()

	node := NewAnalyzedParameterNode("recipient", "address", "0xabc")

	require.Equal(t, "recipient", node.Name())
	require.Equal(t, "address", node.Type())
	require.Equal(t, "0xabc", node.Value())
}
