package tokenpool

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

func TestCompareRateLimiterConfig(t *testing.T) {
	t.Parallel()

	t.Run("capacity change produces diff annotation", func(t *testing.T) {
		t.Parallel()

		current := token_pool.RateLimiterTokenBucket{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}
		proposed := token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(2_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}

		anns := compareRateLimiterConfig(current, proposed, 6, "outbound to BSC", "USDC")
		require.Len(t, anns, 1)
		assert.Equal(t, analyzer.AnnotationDiffName, anns[0].Name())

		dv, ok := anns[0].Value().(analyzer.DiffValue)
		require.True(t, ok)
		assert.Equal(t, "outbound to BSC capacity", dv.Field)
		assert.Equal(t, "1000000 USDC (1,000,000,000,000, decimals=6)", dv.Old)
		assert.Equal(t, "2000000 USDC (2,000,000,000,000, decimals=6)", dv.New)
	})

	t.Run("rate change produces diff annotation", func(t *testing.T) {
		t.Parallel()

		current := token_pool.RateLimiterTokenBucket{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}
		proposed := token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(200_000_000),
		}

		anns := compareRateLimiterConfig(current, proposed, 6, "inbound from BSC", "USDC")
		require.Len(t, anns, 1)
		assert.Equal(t, analyzer.AnnotationDiffName, anns[0].Name())

		dv, ok := anns[0].Value().(analyzer.DiffValue)
		require.True(t, ok)
		assert.Equal(t, "inbound from BSC rate", dv.Field)
	})

	t.Run("enable toggle produces rate limiter annotation", func(t *testing.T) {
		t.Parallel()

		current := token_pool.RateLimiterTokenBucket{
			IsEnabled: false,
			Capacity:  big.NewInt(0),
			Rate:      big.NewInt(0),
		}
		proposed := token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(0),
			Rate:      big.NewInt(0),
		}

		anns := compareRateLimiterConfig(current, proposed, 18, "outbound to Arbitrum", "LINK")
		require.Len(t, anns, 1)
		assert.Equal(t, "ccip.rate_limiter", anns[0].Name())
		assert.Contains(t, anns[0].Value(), "enabled")
	})

	t.Run("zero capacity produces warning and high risk", func(t *testing.T) {
		t.Parallel()

		current := token_pool.RateLimiterTokenBucket{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}
		proposed := token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(0),
			Rate:      big.NewInt(100_000_000),
		}

		anns := compareRateLimiterConfig(current, proposed, 6, "outbound to BSC", "USDC")
		require.Len(t, anns, 3)
		assert.Equal(t, analyzer.AnnotationDiffName, anns[0].Name())
		assert.Equal(t, analyzer.AnnotationSeverityName, anns[1].Name())
		assert.Equal(t, analyzer.AnnotationRiskName, anns[2].Name())
	})

	t.Run("no changes produces empty annotations", func(t *testing.T) {
		t.Parallel()

		current := token_pool.RateLimiterTokenBucket{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}
		proposed := token_pool.RateLimiterConfig{
			IsEnabled: true,
			Capacity:  big.NewInt(1_000_000_000_000),
			Rate:      big.NewInt(100_000_000),
		}

		anns := compareRateLimiterConfig(current, proposed, 6, "outbound to BSC", "USDC")
		assert.Empty(t, anns)
	})
}

func TestFormatRichAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   *big.Int
		decimals uint8
		symbol   string
		expected string
	}{
		{"nil", nil, 18, "ETH", "0"},
		{"zero", big.NewInt(0), 18, "ETH", "0"},
		{
			"18 decimals with fraction",
			new(big.Int).Mul(big.NewInt(25), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)),
			18, "SolvBTC",
			"2.5 SolvBTC (2,500,000,000,000,000,000, decimals=18)",
		},
		{
			"6 decimals whole number",
			big.NewInt(1_000_000), 6, "USDC",
			"1 USDC (1,000,000, decimals=6)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatRichAmount(tt.amount, tt.decimals, tt.symbol))
		})
	}
}

func TestGoldenMarkdown(t *testing.T) {
	t.Parallel()

	proposal := buildGoldenProposal()
	md, err := renderer.NewMarkdownRenderer()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, md.RenderTo(&buf, renderer.RenderRequest{
		Domain:          "ccip",
		EnvironmentName: "mainnet",
	}, proposal))

	golden := filepath.Join("testdata", "golden_markdown.md")
	expected, err := os.ReadFile(golden)
	require.NoError(t, err, "golden file missing — create it with the current output")

	assert.Equal(t, string(expected), buf.String())
}

func buildGoldenProposal() analyzer.AnalyzedProposal {
	call1 := analyzer.NewAnalyzedCallNode(
		"0x1234567890abcdef1234567890abcdef12345678",
		"applyChainUpdates",
		analyzer.AnalyzedParameters{
			analyzer.NewAnalyzedParameterNode("remoteChainSelectorsToRemove", "uint64[]", nil),
			analyzer.NewAnalyzedParameterNode("chainsToAdd", "tuple[]", "(decoded)"),
		},
		nil,
		nil,
		"LockReleaseTokenPool",
		"1.5.1",
		nil,
	)

	call1.AddAnnotations(
		analyzer.NewAnnotation("ccip.token.symbol", "string", "USDC"),
		analyzer.NewAnnotation("ccip.token.decimals", "uint8", uint8(6)),
		analyzer.NewAnnotation("ccip.token.address", "string", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		analyzer.NewAnnotation("ccip.chain_update", "string", "bsc-mainnet (11344663589394136015) added"),
		analyzer.DiffAnnotation(
			"outbound to bsc-mainnet capacity",
			"1 USDC (1,000,000, decimals=6)", "2 USDC (2,000,000, decimals=6)", "",
		),
		analyzer.DiffAnnotation(
			"outbound to bsc-mainnet rate",
			"0.0001 USDC (100, decimals=6)", "0.0002 USDC (200, decimals=6)", "",
		),
		analyzer.NewAnnotation("ccip.rate_limiter", "string", "inbound from bsc-mainnet: rate limiter enabled"),
		analyzer.DiffAnnotation(
			"inbound from bsc-mainnet capacity",
			"0", "0.5 USDC (500,000, decimals=6)", "",
		),
		analyzer.DiffAnnotation(
			"inbound from bsc-mainnet rate",
			"0", "0.00005 USDC (50, decimals=6)", "",
		),
	)

	call2 := analyzer.NewAnalyzedCallNode(
		"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		"applyChainUpdates",
		analyzer.AnalyzedParameters{
			analyzer.NewAnalyzedParameterNode("remoteChainSelectorsToRemove", "uint64[]", "[ 3734025351759498498 ]"),
			analyzer.NewAnalyzedParameterNode("chainsToAdd", "tuple[]", nil),
		},
		nil,
		nil,
		"BurnMintTokenPool",
		"1.5.1",
		nil,
	)

	call2.AddAnnotations(
		analyzer.NewAnnotation("ccip.chain_update", "string", "avalanche-mainnet (6433500567565415381) removed"),
		analyzer.SeverityAnnotation(analyzer.SeverityWarning),
		analyzer.RiskAnnotation(analyzer.RiskMedium),
	)

	batchOp := analyzer.NewAnalyzedBatchOperationNode(
		5009297550715157269,
		analyzer.AnalyzedCalls{call1, call2},
	)

	prop := analyzer.NewAnalyzedProposalNode(
		analyzer.AnalyzedBatchOperations{batchOp},
	)

	return prop
}
