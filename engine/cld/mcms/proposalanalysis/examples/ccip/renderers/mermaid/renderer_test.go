package mermaid

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

func render(t *testing.T, proposal *renderer.AnalyzedProposalNode) string {
	t.Helper()

	var buf bytes.Buffer
	err := NewMermaidRenderer().RenderTo(&buf, renderer.RenderRequest{Domain: "ccip", EnvironmentName: "mainnet"}, proposal)
	require.NoError(t, err)

	return buf.String()
}

func call(addr, method, contractType, version string, anns ...analyzer.Annotation) *renderer.AnalyzedCallNode {
	c := renderer.NewAnalyzedCallNode(addr, method, nil, nil, nil, contractType, version, nil)
	if len(anns) > 0 {
		c.AddAnnotations(anns...)
	}

	return c
}

func batch(chainSel uint64, calls ...*renderer.AnalyzedCallNode) *renderer.AnalyzedBatchOperationNode {
	ac := make(renderer.AnalyzedCalls, len(calls))
	for i, c := range calls {
		ac[i] = c
	}

	return renderer.NewAnalyzedBatchOperationNode(chainSel, ac)
}

func TestGolden_Mermaid(t *testing.T) {
	t.Parallel()

	proposal := renderer.NewAnalyzedProposalNode(renderer.AnalyzedBatchOperations{
		batch(5009297550715157269,
			call("0x1111111111111111111111111111111111111111", "setRateLimiterConfig", "OnRamp", "v1.5.0"),
			call("0x2222222222222222222222222222222222222222", "transfer", "ERC20", ""),
		),
		batch(13264668187771770619,
			call("0x3333333333333333333333333333333333333333", "pause", "", ""),
		),
	})

	out := render(t, proposal)
	expected, err := os.ReadFile(filepath.Join("testdata", "golden_mermaid.txt"))
	require.NoError(t, err)
	require.Equal(t, string(expected), out)
}

func TestMermaid_EmptyProposal(t *testing.T) {
	t.Parallel()

	out := render(t, renderer.NewAnalyzedProposalNode(nil))
	assert.Contains(t, out, "graph TD")
}

func TestMermaid_MultipleBatchesSameChain(t *testing.T) {
	t.Parallel()

	proposal := renderer.NewAnalyzedProposalNode(renderer.AnalyzedBatchOperations{
		batch(5009297550715157269, call("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "applyChainUpdates", "BurnMintTokenPool", "1.5.1")),
		batch(5009297550715157269, call("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "acceptOwnership", "TokenAdminRegistry", "1.5.0")),
		batch(5009297550715157269, call("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "setPool", "TokenAdminRegistry", "1.5.0")),
	})

	out := render(t, proposal)
	assert.Equal(t, 1, strings.Count(out, `subgraph ethereum_mainnet ["ethereum-mainnet"]`), "chain subgraph should appear exactly once")
	assert.Contains(t, out, "1. applyChainUpdates")
	assert.Contains(t, out, "2. acceptOwnership")
	assert.Contains(t, out, "3. setPool")
}

func TestMermaid_CrossChainEdges(t *testing.T) {
	t.Parallel()

	proposal := renderer.NewAnalyzedProposalNode(renderer.AnalyzedBatchOperations{
		batch(9027416829622342829,
			call("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "applyChainUpdates", "BurnMintTokenPool", "1.5.1",
				analyzer.NewAnnotation("ccip.token.symbol", "string", "SolvBTC"),
				analyzer.NewAnnotation("ccip.chain_update", "chain_update", ccip.ChainUpdateValue{RemoteChainSelector: 6433500567565415381, Label: "avalanche-mainnet (6433500567565415381) added"}),
			),
		),
		batch(6433500567565415381,
			call("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "applyChainUpdates", "BurnMintTokenPool", "1.5.1",
				analyzer.NewAnnotation("ccip.token.symbol", "string", "SolvBTC"),
				analyzer.NewAnnotation("ccip.chain_update", "chain_update", ccip.ChainUpdateValue{RemoteChainSelector: 9027416829622342829, Label: "sei-mainnet (9027416829622342829) added"}),
			),
		),
	})

	out := render(t, proposal)
	assert.Contains(t, out, `subgraph sei_mainnet ["sei-mainnet"]`)
	assert.Contains(t, out, `subgraph avalanche_mainnet ["avalanche-mainnet"]`)
	assert.Contains(t, out, "SolvBTC")
	assert.Contains(t, out, "chain update")
	assert.Contains(t, out, ":::pool")
}
