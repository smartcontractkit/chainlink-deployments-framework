package renderer

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

func goldenProposal() *analyzer.AnalyzedProposalNode {
	targetParam := analyzer.NewAnalyzedParameterNode(
		"target", "address", "0xAbCdEf1234567890abcdef1234567890abcdef12",
	)
	targetParam.AddAnnotations(
		annotation.ValueTypeAnnotation("ethereum.address"),
		annotation.New("label", "string", "destination contract"),
	)

	amountParam := analyzer.NewAnalyzedParameterNode(
		"amount", "uint256", big.NewInt(1000000000000000000),
	)
	amountParam.AddAnnotations(annotation.ValueTypeAnnotation("ethereum.uint256"))

	enabledParam := analyzer.NewAnalyzedParameterNode("enabled", "bool", true)

	call1 := analyzer.NewAnalyzedCallNode(
		"0x1111111111111111111111111111111111111111", "setRateLimiterConfig",
		analyzer.AnalyzedParameters{targetParam, amountParam, enabledParam},
		nil, nil, "OnRamp", "v1.5.0", nil,
	)
	call1.AddAnnotations(
		annotation.SeverityAnnotation(annotation.SeverityWarning),
		annotation.RiskAnnotation(annotation.RiskHigh),
		annotation.New("ccip.lane", "string", "ethereum -> arbitrum"),
		annotation.DiffAnnotation("outboundRateLimit", big.NewInt(0), big.NewInt(1000000), "ethereum.uint256"),
	)

	call2 := analyzer.NewAnalyzedCallNode(
		"0x2222222222222222222222222222222222222222", "transfer",
		analyzer.AnalyzedParameters{
			analyzer.NewAnalyzedParameterNode("to", "address", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
			analyzer.NewAnalyzedParameterNode("value", "uint256", big.NewInt(500)),
		},
		nil, nil, "ERC20", "", nil,
	)

	batch1 := analyzer.NewAnalyzedBatchOperationNode(
		5009297550715157269, analyzer.AnalyzedCalls{call1, call2},
	)
	batch1.AddAnnotations(annotation.New("batch.note", "string", "first batch"))

	batch2 := analyzer.NewAnalyzedBatchOperationNode(
		13264668187771770619, analyzer.AnalyzedCalls{
			analyzer.NewAnalyzedCallNode(
				"0x3333333333333333333333333333333333333333", "pause",
				nil, nil, nil, "", "", nil,
			),
		},
	)

	return analyzer.NewAnalyzedProposalNode(
		analyzer.AnalyzedBatchOperations{batch1, batch2},
	)
}

func TestGolden_Markdown(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)

	out := renderToString(t, r,
		RenderRequest{Domain: "ccip", EnvironmentName: "mainnet"},
		goldenProposal(),
	)

	golden := filepath.Join("testdata", "golden_markdown.md")
	expected, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.Equal(t, string(expected), out)
}
