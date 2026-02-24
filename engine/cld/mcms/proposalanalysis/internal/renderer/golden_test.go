package renderer

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"
)

func goldenProposal() *stubProposal {
	targetParam := &stubParam{
		name:  "target",
		typ:   "address",
		value: "0xAbCdEf1234567890abcdef1234567890abcdef12",
	}
	targetParam.AddAnnotations(
		annotation.ValueTypeAnnotation("ethereum.address"),
		annotation.New("label", "string", "destination contract"),
	)

	amountParam := &stubParam{
		name:  "amount",
		typ:   "uint256",
		value: big.NewInt(1000000000000000000),
	}
	amountParam.AddAnnotations(annotation.ValueTypeAnnotation("ethereum.uint256"))

	enabledParam := &stubParam{name: "enabled", typ: "bool", value: true}

	call1 := &stubCall{
		to:              "0x1111111111111111111111111111111111111111",
		name:            "setRateLimiterConfig",
		contractType:    "OnRamp",
		contractVersion: "v1.5.0",
		inputs:          analyzer.AnalyzedParameters{targetParam, amountParam, enabledParam},
	}
	call1.AddAnnotations(
		annotation.SeverityAnnotation(annotation.SeverityWarning),
		annotation.RiskAnnotation(annotation.RiskHigh),
		annotation.New("ccip.lane", "string", "ethereum -> arbitrum"),
	)

	call2 := &stubCall{
		to:           "0x2222222222222222222222222222222222222222",
		name:         "transfer",
		contractType: "ERC20",
		inputs: analyzer.AnalyzedParameters{
			&stubParam{name: "to", typ: "address", value: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
			&stubParam{name: "value", typ: "uint256", value: big.NewInt(500)},
		},
	}

	batch1 := &stubBatchOp{
		chainSelector: 5009297550715157269,
		calls:         analyzer.AnalyzedCalls{call1, call2},
	}
	batch1.AddAnnotations(annotation.New("batch.note", "string", "first batch"))

	batch2 := &stubBatchOp{
		chainSelector: 13264668187771770619,
		calls: analyzer.AnalyzedCalls{
			&stubCall{
				to:   "0x3333333333333333333333333333333333333333",
				name: "pause",
			},
		},
	}

	return &stubProposal{
		batches: analyzer.AnalyzedBatchOperations{batch1, batch2},
	}
}

func TestGolden_Markdown(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)

	out, err := r.RenderToString(
		RenderRequest{Domain: "ccip", EnvironmentName: "mainnet"},
		goldenProposal(),
	)
	require.NoError(t, err)

	golden := filepath.Join("testdata", "golden_markdown.md")
	expected, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.Equal(t, string(expected), out)
}
