package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

func TestNewRenderer_TextFormat(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)
	require.NotNil(t, renderer)
	require.NotNil(t, renderer.tmpl)
	assert.Equal(t, FormatText, renderer.Format())
}

func TestNewRendererWithFormat_HTML(t *testing.T) {
	renderer, err := NewRendererWithFormat(FormatHTML)
	require.NoError(t, err)
	require.NotNil(t, renderer)
	assert.Equal(t, FormatHTML, renderer.Format())
}

func TestRenderer_Render_EmptyProposal_Text(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "ANALYZED PROPOSAL")
	assert.Contains(t, output, "No batch operations found")
}

func TestRenderer_Render_EmptyProposal_HTML(t *testing.T) {
	renderer, err := NewRendererWithFormat(FormatHTML)
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "ANALYZED PROPOSAL")
	assert.Contains(t, output, "No batch operations found")
}

func TestRenderer_Render_WithAnnotations(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	// Create parameter with formatting annotation
	param := &analyzedParameter{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				FormatterAnnotation("ethereum.address"),
				ImportantAnnotation(true),
			},
		},
		decodedParameter: mockDecodedParameter{
			name:  "recipient",
			ptype: "address",
			value: "1234567890abcdef1234567890abcdef12345678",
		},
	}

	call := &analyzedCall{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				SeverityAnnotation("warning"),
				RiskAnnotation("medium"),
			},
		},
		decodedCall: mockDecodedCall{
			name: "transfer",
		},
		inputs:  []analyzer.AnalyzedParameter{param},
		outputs: []analyzer.AnalyzedParameter{},
	}

	batchOp := &analyzedBatchOperation{
		annotated:             &annotated{},
		decodedBatchOperation: mockDecodedBatchOperation{},
		calls:                 []analyzer.AnalyzedCall{call},
	}

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: []analyzer.AnalyzedBatchOperation{batchOp},
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)

	// Check that annotations affect rendering
	assert.Contains(t, output, "transfer")
	assert.Contains(t, output, "⚠")  // warning symbol
	assert.Contains(t, output, "🟡")  // medium risk symbol
	assert.Contains(t, output, "⭐")  // important marker
	assert.Contains(t, output, "0x") // ethereum address formatter
}

func TestRenderer_Render_HTML_WithAnnotations(t *testing.T) {
	renderer, err := NewRendererWithFormat(FormatHTML)
	require.NoError(t, err)

	// Create parameter with important annotation
	param := &analyzedParameter{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				ImportantAnnotation(true),
				EmojiAnnotation("💰"),
			},
		},
		decodedParameter: mockDecodedParameter{
			name:  "amount",
			ptype: "uint256",
			value: "1000000000000000000",
		},
	}

	call := &analyzedCall{
		annotated: &annotated{
			annotations: []analyzer.Annotation{
				ImportantAnnotation(true),
			},
		},
		decodedCall: mockDecodedCall{
			name: "mint",
		},
		inputs:  []analyzer.AnalyzedParameter{param},
		outputs: []analyzer.AnalyzedParameter{},
	}

	batchOp := &analyzedBatchOperation{
		annotated:             &annotated{},
		decodedBatchOperation: mockDecodedBatchOperation{},
		calls:                 []analyzer.AnalyzedCall{call},
	}

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: []analyzer.AnalyzedBatchOperation{batchOp},
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)

	// Check HTML specific formatting
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "mint")
	assert.Contains(t, output, "⭐")                   // important marker
	assert.Contains(t, output, "💰")                   // emoji
	assert.Contains(t, output, "class=\"important\"") // important class
}

func TestFormatParameterValue_EthereumAddress(t *testing.T) {
	param := &analyzedParameter{
		decodedParameter: mockDecodedParameter{
			value: "1234567890abcdef1234567890abcdef12345678",
		},
	}

	result := formatParameterValue(param, "ethereum.address")
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", result)
}

func TestFormatParameterValue_EthereumUint256(t *testing.T) {
	param := &analyzedParameter{
		decodedParameter: mockDecodedParameter{
			value: "1000000000",
		},
	}

	result := formatParameterValue(param, "ethereum.uint256")
	assert.Equal(t, "1,000,000,000", result)
}

func TestFormatParameterValue_Hex(t *testing.T) {
	param := &analyzedParameter{
		decodedParameter: mockDecodedParameter{
			value: []byte{0x12, 0x34, 0x56, 0x78},
		},
	}

	result := formatParameterValue(param, "hex")
	assert.Contains(t, result, "0x")
}

func TestFormatParameterValue_Truncate(t *testing.T) {
	param := &analyzedParameter{
		decodedParameter: mockDecodedParameter{
			value: "this is a very long string that should be truncated",
		},
	}

	result := formatParameterValue(param, "truncate:20")
	assert.Equal(t, "this is a very lo...", result)
	assert.Equal(t, 20, len(result))
}

func TestTemplateFunc_GetAnnotation(t *testing.T) {
	funcs := templateFuncs()
	getAnnotation := funcs["getAnnotation"].(func(analyzer.Annotated, string) analyzer.Annotation)

	annotated := &annotated{
		annotations: []analyzer.Annotation{
			ImportantAnnotation(true),
			EmojiAnnotation("🔥"),
		},
	}

	// Test getting existing annotation
	ann := getAnnotation(annotated, "render.important")
	require.NotNil(t, ann)
	assert.Equal(t, "render.important", ann.Name())
	assert.Equal(t, true, ann.Value())

	// Test getting non-existing annotation
	ann = getAnnotation(annotated, "non.existent")
	assert.Nil(t, ann)

	// Test with nil annotated
	ann = getAnnotation(nil, "render.important")
	assert.Nil(t, ann)
}

func TestTemplateFunc_GetAnnotationValue(t *testing.T) {
	funcs := templateFuncs()
	getAnnotationValue := funcs["getAnnotationValue"].(func(analyzer.Annotated, string) interface{})

	annotated := &annotated{
		annotations: []analyzer.Annotation{
			FormatterAnnotation("ethereum.address"),
		},
	}

	// Test getting existing annotation value
	val := getAnnotationValue(annotated, "render.formatter")
	assert.Equal(t, "ethereum.address", val)

	// Test getting non-existing annotation value
	val = getAnnotationValue(annotated, "non.existent")
	assert.Nil(t, val)
}

func TestTemplateFunc_HasAnnotation(t *testing.T) {
	funcs := templateFuncs()
	hasAnnotation := funcs["hasAnnotation"].(func(analyzer.Annotated, string) bool)

	annotated := &annotated{
		annotations: []analyzer.Annotation{
			ImportantAnnotation(true),
		},
	}

	// Test existing annotation
	assert.True(t, hasAnnotation(annotated, "render.important"))

	// Test non-existing annotation
	assert.False(t, hasAnnotation(annotated, "non.existent"))

	// Test with nil
	assert.False(t, hasAnnotation(nil, "render.important"))
}

func TestNewRendererWithTemplates(t *testing.T) {
	customTemplates := map[string]string{
		"proposal": `{{define "proposal"}}Custom Proposal{{end}}`,
	}

	renderer, err := NewRendererWithTemplates(FormatText, customTemplates)
	require.NoError(t, err)

	proposal := &analyzedProposal{
		annotated:       &annotated{},
		decodedProposal: mockDecodedTimelockProposal{},
		batchOperations: nil,
	}

	output, err := renderer.Render(proposal)
	require.NoError(t, err)
	assert.Contains(t, output, "Custom Proposal")
}
