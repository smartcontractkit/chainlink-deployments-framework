package renderer

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

func newTestProposal() *analyzer.AnalyzedProposalNode {
	return analyzer.NewAnalyzedProposalNode(
		analyzer.AnalyzedBatchOperations{
			analyzer.NewAnalyzedBatchOperationNode(5009297550715157269, analyzer.AnalyzedCalls{
				analyzer.NewAnalyzedCallNode(
					"0x1111111111111111111111111111111111111111", "transfer",
					analyzer.AnalyzedParameters{
						analyzer.NewAnalyzedParameterNode("amount", "uint256", big.NewInt(1000)),
						analyzer.NewAnalyzedParameterNode("recipient", "address", "0xabcdef1234567890abcdef1234567890abcdef12"),
					},
					nil, nil, "ERC20", "v1.0.0", nil,
				),
			}),
		},
	)
}

func renderToString(t *testing.T, r *TemplateRenderer, req RenderRequest, proposal analyzer.AnalyzedProposal) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, r.RenderTo(&buf, req, proposal))

	return buf.String()
}

func minimalTemplates(proposalTemplate string) map[string]string {
	return map[string]string{
		"proposal":       proposalTemplate,
		"batchOperation": `{{ define "batchOperation" }}{{ end }}`,
		"call":           `{{ define "call" }}{{ end }}`,
		"parameter":      `{{ define "parameter" }}{{ end }}`,
		"annotations":    `{{ define "annotations" }}{{ end }}`,
	}
}

func TestNewMarkdownRenderer(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	assert.Equal(t, IDMarkdown, r.ID())

	out := renderToString(t, r, RenderRequest{Domain: "ccip", EnvironmentName: "mainnet"}, newTestProposal())

	assert.Contains(t, out, "## Proposal — ccip (mainnet)")
	assert.Contains(t, out, "<details>")
	assert.Contains(t, out, "Batch 1")
	assert.Contains(t, out, "5009297550715157269")
	assert.Contains(t, out, "**ERC20 v1.0.0**")
	assert.Contains(t, out, "`transfer`")
	assert.Contains(t, out, "- [ ]")
	assert.Contains(t, out, "0x1111..1111")
	assert.Contains(t, out, "**`amount`**")
	assert.Contains(t, out, "**`recipient`**")
}

func TestNoTitleWithoutRequest(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)

	out := renderToString(t, r, RenderRequest{}, newTestProposal())

	assert.NotContains(t, out, "## Proposal")
	assert.Contains(t, out, "<details>")
}

func TestRenderEmptyProposal(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, analyzer.NewAnalyzedProposalNode(nil))
	assert.Contains(t, out, "No batch operations")
}

func TestRenderWithAnnotations(t *testing.T) {
	t.Parallel()

	call := analyzer.NewAnalyzedCallNode(
		"0x1111111111111111111111111111111111111111", "setConfig",
		nil, nil, nil, "Router", "", nil,
	)
	call.AddAnnotations(
		annotation.New("ccip.lane", "string", "ethereum -> arbitrum"),
		annotation.New("description", "string", "update router config"),
	)

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(123, analyzer.AnalyzedCalls{call}),
	})

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, proposal)

	assert.Contains(t, out, "ccip.lane: ethereum -> arbitrum")
	assert.Contains(t, out, "description: update router config")
	assert.Contains(t, out, "_Annotations:_")
}

func TestRenderTo_PassesRenderRequest(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer(
		WithTemplates(minimalTemplates(`{{ define "proposal" }}{{ .Request.Domain }}-{{ .Request.EnvironmentName }}{{ end }}`)),
	)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.RenderTo(&buf, RenderRequest{Domain: "test", EnvironmentName: "staging"}, newTestProposal())
	require.NoError(t, err)
	assert.Equal(t, "test-staging", buf.String())
}

func TestWithTemplateFuncs(t *testing.T) {
	t.Parallel()

	customFuncs := template.FuncMap{
		"shout": func(s string) string { return strings.ToUpper(s) },
	}

	r, err := NewMarkdownRenderer(
		WithTemplateFuncs(customFuncs),
		WithTemplates(minimalTemplates(`{{ define "proposal" }}{{ shout "hello" }}{{ end }}`)),
	)
	require.NoError(t, err)

	out := renderToString(t, r, RenderRequest{}, newTestProposal())
	assert.Equal(t, "HELLO", out)
}

func TestMissingTemplateDefinitionsError(t *testing.T) {
	t.Parallel()

	_, err := NewMarkdownRenderer(
		WithTemplates(map[string]string{
			"proposal": `{{ define "proposal" }}only{{ end }}`,
		}),
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "missing required template definitions")
}

func TestWithTemplates_InMemoryOverride(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer(
		WithTemplates(minimalTemplates(`{{ define "proposal" }}custom: {{len .Proposal.BatchOperations}} batches{{ end }}`)),
	)
	require.NoError(t, err)

	out := renderToString(t, r, RenderRequest{}, newTestProposal())
	assert.Equal(t, "custom: 1 batches", out)
}

func TestWithTemplateDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.tmpl"), []byte(`{{ define "proposal" }}dir: {{len .Proposal.BatchOperations}}{{ end }}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "batchOperation.tmpl"), []byte(`{{ define "batchOperation" }}{{ end }}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "call.tmpl"), []byte(`{{ define "call" }}{{ end }}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "parameter.tmpl"), []byte(`{{ define "parameter" }}{{ end }}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "annotations.tmpl"), []byte(`{{ define "annotations" }}{{ end }}`), 0o600))

	r, err := NewMarkdownRenderer(WithTemplateDir(dir))
	require.NoError(t, err)

	out := renderToString(t, r, RenderRequest{}, newTestProposal())
	assert.Equal(t, "dir: 1", out)
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, "<nil>"},
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"bytes empty", []byte{}, "0x"},
		{"bytes", []byte{0xde, 0xad, 0xbe, 0xef}, "0xdeadbeef"},
		{"big.Int", big.NewInt(123456789), "123456789"},
		{"big.Int nil", (*big.Int)(nil), "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, formatValue(tt.input))
		})
	}
}

func TestFormatValue_MapsAndSlices(t *testing.T) {
	t.Parallel()

	t.Run("map is JSON-pretty-printed", func(t *testing.T) {
		t.Parallel()

		v := map[string]any{"enabled": true, "capacity": 1000}
		out := formatValue(v)
		assert.True(t, strings.HasPrefix(out, "{"), "expected JSON object, got: %s", out)
		assert.Contains(t, out, "enabled")
		assert.Contains(t, out, "capacity")
	})

	t.Run("slice is JSON-pretty-printed", func(t *testing.T) {
		t.Parallel()

		v := []any{"ethereum", "arbitrum", "polygon"}
		out := formatValue(v)
		assert.True(t, strings.HasPrefix(out, "["), "expected JSON array, got: %s", out)
		assert.Contains(t, out, "ethereum")
	})
}

func TestFindAnnotationValue(t *testing.T) {
	t.Parallel()

	anns := annotation.Annotations{
		annotation.New("severity", "string", "high"),
		annotation.New("label", "string", "important value"),
	}

	assert.Equal(t, "high", findAnnotationValue(anns, "severity"))
	assert.Equal(t, "important value", findAnnotationValue(anns, "label"))
	assert.Nil(t, findAnnotationValue(anns, "nonexistent"))
	assert.Nil(t, findAnnotationValue(nil, "severity"))
}

func TestIsFrameworkAnnotation(t *testing.T) {
	t.Parallel()

	assert.True(t, isFrameworkAnnotation("cld.severity"))
	assert.True(t, isFrameworkAnnotation("cld.risk"))
	assert.True(t, isFrameworkAnnotation("cld.value_type"))
	assert.True(t, isFrameworkAnnotation("cld.diff"))
	assert.False(t, isFrameworkAnnotation("ccip.lane"))
	assert.False(t, isFrameworkAnnotation(""))
}

func TestHasDisplayAnnotations(t *testing.T) {
	t.Parallel()

	assert.False(t, hasDisplayAnnotations(nil))
	assert.False(t, hasDisplayAnnotations(annotation.Annotations{
		annotation.ValueTypeAnnotation("ethereum.uint256"),
		annotation.SeverityAnnotation(annotation.SeverityWarning),
	}))
	assert.True(t, hasDisplayAnnotations(annotation.Annotations{
		annotation.ValueTypeAnnotation("ethereum.uint256"),
		annotation.New("ccip.note", "string", "visible"),
	}))
}

func TestFormatParam_WithValueTypeAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     any
		valueType string
		expected  string
	}{
		{name: "ethereum.address from short string", value: "abcd", valueType: "ethereum.address", expected: "0x000000000000000000000000000000000000abcd"},
		{name: "ethereum.uint256 with commas", value: big.NewInt(1000000), valueType: "ethereum.uint256", expected: "1,000,000"},
		{name: "hex from bytes", value: []byte{0xde, 0xad}, valueType: "hex", expected: "0xdead"},
		{name: "nil value", value: nil, valueType: "ethereum.address", expected: "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := analyzer.NewAnalyzedParameterNode("x", "test", tt.value)
			p.AddAnnotations(annotation.ValueTypeAnnotation(tt.valueType))
			assert.Equal(t, tt.expected, formatParam(p))
		})
	}
}

func TestTruncateAddress(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0x1234..5678", truncateAddress("0x1234567890abcdef1234567890abcdef12345678"))
	assert.Equal(t, "0xAbCd..ef12", truncateAddress("0xAbCdEf1234567890abcdef1234567890abcdef12"))
	assert.Equal(t, "7EqQ..ZCk", truncateAddress("7EqQdEULxWcraVx3mXKFjc84LhCkMGZCk"))
	assert.Equal(t, "0xaaaa", truncateAddress("0xaaaa"))
	assert.Equal(t, "short", truncateAddress("short"))
	assert.Empty(t, truncateAddress(""))
}

func TestFormatAsHex_UnsupportedTypeFallsBack(t *testing.T) {
	t.Parallel()

	out := formatAsHex(map[string]any{"foo": "bar"})
	assert.NotContains(t, out, "%!")
	assert.Contains(t, out, "foo")
}

func TestFormatEthereumUint256_NegativeNumber(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-1,000", formatEthereumUint256(big.NewInt(-1000)))
	assert.Equal(t, "-10,000,000", formatEthereumUint256("-10000000"))
}

func TestRenderSeverityAndRisk_Markdown(t *testing.T) {
	t.Parallel()

	call := analyzer.NewAnalyzedCallNode("0xaaaa", "dangerousMethod", nil, nil, nil, "Router", "", nil)
	call.AddAnnotations(
		annotation.SeverityAnnotation(annotation.SeverityWarning),
		annotation.RiskAnnotation(annotation.RiskHigh),
		annotation.New("ccip.lane", "string", "ethereum -> arbitrum"),
	)

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(1, analyzer.AnalyzedCalls{call}),
	})

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, proposal)

	assert.Contains(t, out, "⚠")
	assert.Contains(t, out, "**warning**")
	assert.Contains(t, out, "🔴")
	assert.Contains(t, out, "**high**")
	assert.Contains(t, out, "ccip.lane: ethereum -> arbitrum")
	assert.NotContains(t, out, "cld.severity:")
	assert.NotContains(t, out, "cld.risk:")
}

func TestWithTemplateFuncs_OverridesBuiltIn(t *testing.T) {
	t.Parallel()

	r, err := NewMarkdownRenderer(
		WithTemplateFuncs(template.FuncMap{
			"formatParam": func(_ analyzer.AnalyzedParameter) string { return "CUSTOM" },
		}),
	)
	require.NoError(t, err)

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(1, analyzer.AnalyzedCalls{
			analyzer.NewAnalyzedCallNode("0xaaaa", "test",
				analyzer.AnalyzedParameters{analyzer.NewAnalyzedParameterNode("x", "uint256", big.NewInt(42))},
				nil, nil, "", "", nil,
			),
		}),
	})

	out := renderToString(t, r, RenderRequest{}, proposal)
	assert.Contains(t, out, "CUSTOM")
	assert.NotContains(t, out, "42")
}

func TestRenderCallOutputs(t *testing.T) {
	t.Parallel()

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(1, analyzer.AnalyzedCalls{
			analyzer.NewAnalyzedCallNode("0xaaaa", "getConfig", nil,
				analyzer.AnalyzedParameters{
					analyzer.NewAnalyzedParameterNode("rate", "uint256", big.NewInt(500)),
					analyzer.NewAnalyzedParameterNode("active", "bool", true),
				},
				nil, "", "", nil,
			),
		}),
	})

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, proposal)

	assert.Contains(t, out, "**Outputs:**")
	assert.Contains(t, out, "**`rate`**")
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "**`active`**")
	assert.Contains(t, out, "true")
}

func TestRenderDiff_Markdown(t *testing.T) {
	t.Parallel()

	call := analyzer.NewAnalyzedCallNode("0xaaaa", "applyChainUpdates", nil, nil, nil, "TokenPool", "v1.5.0", nil)
	call.AddAnnotations(
		annotation.DiffAnnotation("outbound.capacity", big.NewInt(0), big.NewInt(1000000), "ethereum.uint256"),
		annotation.DiffAnnotation("inbound.rate", big.NewInt(100), big.NewInt(500), "ethereum.uint256"),
	)

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(1, analyzer.AnalyzedCalls{call}),
	})

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, proposal)

	assert.Contains(t, out, "**Changes:**")
	assert.Contains(t, out, "**outbound.capacity:** ~~0~~ -> **1,000,000**")
	assert.Contains(t, out, "**inbound.rate:** ~~100~~ -> **500**")
}

func TestDiffFuncmap(t *testing.T) {
	t.Parallel()

	t.Run("renderDiff with field and value type", func(t *testing.T) {
		t.Parallel()

		dv := annotation.DiffValue{Field: "rate", Old: big.NewInt(100), New: big.NewInt(200), ValueType: "ethereum.uint256"}
		assert.Equal(t, "**rate:** ~~100~~ -> **200**", renderDiff(dv))
	})

	t.Run("renderDiff without field", func(t *testing.T) {
		t.Parallel()

		dv := annotation.DiffValue{Old: "old", New: "new"}
		assert.Equal(t, "~~old~~ -> **new**", renderDiff(dv))
	})

	t.Run("renderDiff without value type skips type-specific formatting", func(t *testing.T) {
		t.Parallel()

		dv := annotation.DiffValue{Field: "rate", Old: big.NewInt(1000000), New: big.NewInt(2000000)}
		assert.Equal(t, "**rate:** ~~1000000~~ -> **2000000**", renderDiff(dv))
	})

	t.Run("diffAnnotations extracts only diff values", func(t *testing.T) {
		t.Parallel()

		anns := annotation.Annotations{
			annotation.SeverityAnnotation(annotation.SeverityWarning),
			annotation.DiffAnnotation("a", 1, 2, ""),
			annotation.New("ccip.lane", "string", "eth -> arb"),
			annotation.DiffAnnotation("b", 3, 4, ""),
		}
		diffs := diffAnnotations(anns)
		assert.Len(t, diffs, 2)
		assert.Equal(t, "a", diffs[0].Field)
		assert.Equal(t, "b", diffs[1].Field)
	})

	t.Run("diffAnnotations returns nil when no diffs", func(t *testing.T) {
		t.Parallel()

		anns := annotation.Annotations{
			annotation.SeverityAnnotation(annotation.SeverityInfo),
		}
		assert.Nil(t, diffAnnotations(anns))
	})
}

func TestParameterValueTypeNotRenderedAsAnnotation(t *testing.T) {
	t.Parallel()

	target := analyzer.NewAnalyzedParameterNode("target", "address", "abcd")
	target.AddAnnotations(
		annotation.ValueTypeAnnotation("ethereum.address"),
		annotation.New("label", "string", "destination contract"),
	)

	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(1, analyzer.AnalyzedCalls{
			analyzer.NewAnalyzedCallNode("0xaaaa", "setConfig",
				analyzer.AnalyzedParameters{target},
				nil, nil, "", "", nil,
			),
		}),
	})

	r, err := NewMarkdownRenderer()
	require.NoError(t, err)
	out := renderToString(t, r, RenderRequest{}, proposal)

	assert.Contains(t, out, "destination contract")
	assert.NotContains(t, out, "cld.value_type:")
}
