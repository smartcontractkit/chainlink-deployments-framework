package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestMarkdownRenderer_RenderDecodedCall_SimpleCall(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "transfer(address,uint256)",
		Inputs: []NamedField{
			{
				Name:  "to",
				Value: AddressField{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
			},
			{
				Name:  "amount",
				Value: SimpleField{Value: "1000000000000000000"},
			},
		},
		Outputs: []NamedField{},
	}

	ctx := NewFieldContext(nil)
	output := renderer.RenderDecodedCall(call, ctx)

	assert.Contains(t, output, "**Address:** `0x1234567890123456789012345678901234567890`")
	assert.Contains(t, output, "**Method:** `transfer(address,uint256)`")
	assert.Contains(t, output, "**Inputs:**")
	assert.Contains(t, output, "- `to`: `0xabcdefabcdefabcdefabcdefabcdefabcdefabcd`")
	assert.Contains(t, output, "- `amount`: `1000000000000000000`")
	assert.NotContains(t, output, "**Outputs:**")
}

func TestMarkdownRenderer_RenderDecodedCall_WithOutputs(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "balanceOf(address)",
		Inputs: []NamedField{
			{
				Name:  "account",
				Value: AddressField{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
			},
		},
		Outputs: []NamedField{
			{
				Name:  "balance",
				Value: SimpleField{Value: "5000000000000000000"},
			},
		},
	}

	ctx := NewFieldContext(nil)
	output := renderer.RenderDecodedCall(call, ctx)

	assert.Contains(t, output, "**Inputs:**")
	assert.Contains(t, output, "- `account`: `0xabcdefabcdefabcdefabcdefabcdefabcdefabcd`")
	assert.Contains(t, output, "**Outputs:**")
	assert.Contains(t, output, "- `balance`: `5000000000000000000`")
}

func TestMarkdownRenderer_RenderDecodedCall_WithAddressAnnotation(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "transfer(address,uint256)",
		Inputs:  []NamedField{},
		Outputs: []NamedField{},
	}

	// Create context with address mapping
	addresses := deployment.AddressesByChain{
		1: {
			"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
		},
	}
	ctx := NewFieldContext(addresses)
	output := renderer.RenderDecodedCall(call, ctx)

	assert.Contains(t, output, "**Address:** `0x1234567890123456789012345678901234567890`")
	assert.Contains(t, output, "<sub><i>address of Token 1.0.0 from 1</i></sub>")
}

func TestMarkdownRenderer_RenderDecodedCall_EmptyInputsOutputs(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "execute()",
		Inputs:  []NamedField{},
		Outputs: []NamedField{},
	}

	ctx := NewFieldContext(nil)
	output := renderer.RenderDecodedCall(call, ctx)

	assert.Contains(t, output, "**Address:** `0x1234567890123456789012345678901234567890`")
	assert.Contains(t, output, "**Method:** `execute()`")
	assert.NotContains(t, output, "**Inputs:**")
	assert.NotContains(t, output, "**Outputs:**")
}

// Tests for MarkdownRenderer descriptor handling methods

func TestMarkdownRenderer_SummarizeArgument_SimpleField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Short string
	summary, details := renderer.summarizeField("name", SimpleField{Value: "short"}, ctx)
	assert.Equal(t, "`short`", summary)
	assert.Empty(t, details)

	// Long string
	longValue := "this is a very long string that should trigger details section because it exceeds the 80 character limit"
	summary, details = renderer.summarizeField("longName", SimpleField{Value: longValue}, ctx)
	assert.Contains(t, summary, "`this is a very long string that should …cause it exceeds the 80 character limit` (len=")
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	assert.Contains(t, details, longValue)
}

func TestMarkdownRenderer_SummarizeArgument_AddressField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	summary, details := renderer.summarizeField("addr", AddressField{Value: "0x1234567890123456789012345678901234567890"}, ctx)
	assert.Equal(t, "`0x1234567890123456789012345678901234567890`", summary)
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_ChainSelectorField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	summary, details := renderer.summarizeField("chain", ChainSelectorField{Value: 1}, ctx)
	assert.Contains(t, summary, "`1 (<chain unknown>)`")
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_BytesField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Small bytes
	smallBytes := []byte{0x01, 0x02, 0x03}
	summary, details := renderer.summarizeField("small", BytesField{Value: smallBytes}, ctx)
	assert.Contains(t, summary, "bytes(len=3):")
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	assert.Contains(t, details, "0x010203")

	// Large bytes
	largeBytes := make([]byte, 50)
	for i := range largeBytes {
		largeBytes[i] = byte(i)
	}
	summary, details = renderer.summarizeField("large", BytesField{Value: largeBytes}, ctx)
	assert.Contains(t, summary, "bytes(len=50):")
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
}

func TestMarkdownRenderer_SummarizeArgument_ArrayField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Empty array
	summary, details := renderer.summarizeField("empty", ArrayField{Elements: []FieldValue{}}, ctx)
	assert.Equal(t, "array[0]: []", summary)
	assert.Empty(t, details)

	// Non-empty array
	elements := []FieldValue{
		SimpleField{Value: "item1"},
		SimpleField{Value: "item2"},
		SimpleField{Value: "item3"},
	}
	summary, details = renderer.summarizeField("array", ArrayField{Elements: elements}, ctx)
	assert.Contains(t, summary, "array[3]")
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
}

func TestMarkdownRenderer_SummarizeArgument_StructField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	fields := []NamedField{
		{Name: "field1", Value: SimpleField{Value: "value1"}},
		{Name: "field2", Value: SimpleField{Value: "value2"}},
	}
	summary, details := renderer.summarizeField("struct", StructField{Fields: fields}, ctx)
	assert.Contains(t, summary, "struct{ 2 fields }")
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	// Test that details contain actual field content, not just the summary
	assert.Contains(t, details, "field1: value1")
	assert.Contains(t, details, "field2: value2")
}

func TestMarkdownRenderer_SummarizeArgument_ArrayField_Details(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	elements := []FieldValue{
		SimpleField{Value: "item1"},
		SimpleField{Value: "item2"},
		SimpleField{Value: "item3"},
	}
	array := ArrayField{Elements: elements}
	summary, details := renderer.summarizeField("array", array, ctx)

	// Test summary
	assert.Contains(t, summary, "array[3]")

	// Test that details contain actual array content, not just the summary
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	assert.Contains(t, details, "item1")
	assert.Contains(t, details, "item2")
	assert.Contains(t, details, "item3")
	assert.NotContains(t, details, "array[3]: [item1, item2, item3]")
}

func TestMarkdownRenderer_SummarizeArgument_StructField_Empty(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Test with an empty struct (no fields)
	emptyStruct := StructField{Fields: []NamedField{}}
	summary, details := renderer.summarizeField("emptyStruct", emptyStruct, ctx)

	// Test summary
	assert.Contains(t, summary, "struct{ 0 fields }")

	// Test that details show the appropriate message for empty struct
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	assert.Contains(t, details, "struct with 0 fields (no field data available)")
}

func TestMarkdownRenderer_StructDetail(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Create a struct that represents the real-world case from the user's output
	// This should have actual field content, not just "struct{ 2 fields }"
	fields := []NamedField{
		{Name: "ChainSelector", Value: SimpleField{Value: "3017758115101368649"}},
		{Name: "ChainConfig", Value: SimpleField{Value: "0x7b226761735072696365446576696174696f6e505042223a2234303030303030303030222c2264614761735072696365446576696174696f6e505042223a2234303030303030303030222c226f7074696d6973746963436f6e6669726d6174696f6e73223a312c22636861696e466565446576696174696f6e44697361626c6564223a66616c73657d"}},
	}
	structField := StructField{Fields: fields}
	summary, details := renderer.summarizeField("chainConfigAdds", structField, ctx)

	assert.Contains(t, summary, "struct{ 2 fields }")

	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")

	// details should contain actual field content
	assert.Contains(t, details, "ChainSelector: 3017758115101368649")
	assert.Contains(t, details, "ChainConfig: 0x7b226761735072696365446576696174696f6e505042223a2234303030303030303030222c2264614761735072696365446576696174696f6e505042223a2234303030303030303030222c226f7074696d6973746963436f6e6669726d6174696f6e73223a312c22636861696e466565446576696174696f6e44697361626c6564223a66616c73657d")

	// Details should NOT contain just the summary
	assert.NotContains(t, details, "struct{ 2 fields }")
}

func TestMarkdownRenderer_ArrayField_WithStructElement(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Test array with struct element - this matches the real data structure
	// ArrayField containing 1 StructField with nested fields
	nestedStructFields := []NamedField{
		{Name: "Readers", Value: ArrayField{Elements: []FieldValue{}}},
		{Name: "FChain", Value: SimpleField{Value: "someValue"}},
		{Name: "Config", Value: BytesField{Value: []byte{0x01, 0x02, 0x03}}},
	}
	nestedStruct := StructField{Fields: nestedStructFields}

	outerStructFields := []NamedField{
		{Name: "ChainSelector", Value: ChainSelectorField{Value: 3017758115101368649}},
		{Name: "ChainConfig", Value: nestedStruct},
	}
	outerStruct := StructField{Fields: outerStructFields}

	// Create ArrayField with 1 StructField element
	arrayField := ArrayField{
		Elements: []FieldValue{outerStruct},
	}

	summary, details := renderer.summarizeField("chainConfigAdds", arrayField, ctx)

	// Summary should show array with struct
	assert.Contains(t, summary, "array[1]: [struct]")

	// Details should show FULL nested content in GitHub Flavored Markdown collapsible section
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")

	// Should show the outer struct fields
	assert.Contains(t, details, "ChainSelector: 3017758115101368649")
	assert.Contains(t, details, "ChainConfig:")

	// Should show the nested struct fields
	assert.Contains(t, details, "Readers:")
	assert.Contains(t, details, "FChain: someValue")
	assert.Contains(t, details, "Config:")

	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")

	// Details should NOT contain just the summary
	assert.NotContains(t, details, "array[1]: [struct]")
}

func TestMarkdownRenderer_StructField_EmptyFields(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Test struct with no fields - should show appropriate message
	structField := StructField{Fields: []NamedField{}}
	summary, details := renderer.summarizeField("emptyStruct", structField, ctx)

	// Summary should show field count
	assert.Contains(t, summary, "struct{ 0 fields }")

	// Details should show appropriate message for empty struct
	assert.Contains(t, details, "<details>")
	assert.Contains(t, details, "<pre><code>")
	assert.Contains(t, details, "</code></pre>")
	assert.Contains(t, details, "</details>")
	assert.Contains(t, details, "struct with 0 fields (no field data available)")
}

func TestMarkdownRenderer_SummarizeArgument_NamedField(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	named := NamedField{
		Name:  "param",
		Value: SimpleField{Value: "value"},
	}
	summary, details := renderer.summarizeField("named", named, ctx)
	assert.Contains(t, summary, "`param: value`")
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_DefaultCase(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Test with a descriptor that doesn't match any specific case
	// This should fall through to the default case
	customDesc := &customFieldValue{value: "custom value"}
	summary, details := renderer.summarizeField("custom", customDesc, ctx)
	assert.Contains(t, summary, "`&{custom value}`")
	assert.Empty(t, details)
}

func TestMarkdownRenderer_YamlField_PrettyPrinting(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewFieldContext(nil)

	// Test YAML field with complex data that should be pretty-printed
	yamlData := map[string]interface{}{
		"items": []string{
			"3Yrg9E4ySAeRezgQY99NNarAmFLtixapga9MZb6y2dt3",
			"GVBN6ikQrqJBWQtiS1i8ejRhBN8HcpCAhb6abaktsprG",
			"B1xCxcjXiNTZpfUb9tZZgEAYFX84X7Y9R5JX9gSiNLG5",
		},
		"config": map[string]interface{}{
			"writable": true,
			"signer":   false,
		},
	}

	yamlField := YamlField{Value: yamlData}
	summary, details := renderer.summarizeField("AccountMetaSlice", yamlField, ctx)

	// Summary should show the compact YAML version
	assert.Contains(t, summary, "config:")
	assert.Contains(t, summary, "signer: false")

	// Details should show pretty-printed YAML with proper indentation
	assert.Contains(t, details, "<details><pre><code>")
	assert.Contains(t, details, "</code></pre></details>")

	// Should contain the YAML structure with proper formatting and html espaced strings &#45 for dashes "-"
	assert.Contains(t, details, "items:")
	assert.Contains(t, details, "&#45; 3Yrg9E4ySAeRezgQY99NNarAmFLtixapga9MZb6y2dt3")
	assert.Contains(t, details, "&#45; GVBN6ikQrqJBWQtiS1i8ejRhBN8HcpCAhb6abaktsprG")
	assert.Contains(t, details, "config:")
	assert.Contains(t, details, "writable: true")
	assert.Contains(t, details, "signer: false")
}

// Helper type for testing default case
type customFieldValue struct {
	value string
}

func (c *customFieldValue) GetType() string {
	return "customFieldValue"
}

func (c *customFieldValue) Describe(ctx *FieldContext) string {
	return c.value
}

// Tests for helper functions

func TestArrayPreview(t *testing.T) {
	t.Parallel()

	ctx := NewFieldContext(nil)

	// Empty array
	elements := []FieldValue{}
	preview := arrayPreview(elements, ctx)
	assert.Empty(t, preview)

	// Small array
	elements = []FieldValue{
		SimpleField{Value: "item1"},
		SimpleField{Value: "item2"},
	}
	preview = arrayPreview(elements, ctx)
	assert.Contains(t, preview, ": [item1, item2]")

	// Large array (should truncate)
	elements = []FieldValue{
		SimpleField{Value: "item1"},
		SimpleField{Value: "item2"},
		SimpleField{Value: "item3"},
		SimpleField{Value: "item4"},
		SimpleField{Value: "item5"},
	}
	preview = arrayPreview(elements, ctx)
	assert.Contains(t, preview, ": [item1, item2, item3, … (+2)]")
}

func TestCompactValue(t *testing.T) {
	t.Parallel()

	ctx := NewFieldContext(nil)

	// Test different descriptor types
	assert.Equal(t, "0x1234567890123456789012345678901234567890", compactValue(AddressField{Value: "0x1234567890123456789012345678901234567890"}, ctx))
	assert.Contains(t, compactValue(ChainSelectorField{Value: 1}, ctx), "1")
	assert.Contains(t, compactValue(BytesField{Value: []byte{0x01, 0x02}}, ctx), "0x0102")
	assert.Equal(t, "short", compactValue(SimpleField{Value: "short"}, ctx))
	assert.Equal(t, "struct", compactValue(StructField{Fields: []NamedField{}}, ctx))
	assert.Contains(t, compactValue(ArrayField{Elements: []FieldValue{}}, ctx), "array[0]")
}

func TestHexPreview(t *testing.T) {
	t.Parallel()

	// Small bytes
	small := []byte{0x01, 0x02, 0x03}
	preview := hexPreview(small, 16)
	assert.Equal(t, "0x010203", preview)

	// Large bytes
	large := make([]byte, 50)
	for i := range large {
		large[i] = byte(i)
	}
	preview = hexPreview(large, 4)
	assert.Contains(t, preview, "0x00010203")
	assert.Contains(t, preview, "…")
}

func TestTruncateMiddle(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "short", truncateMiddle("short", 10))

	// Long string
	long := "this is a very long string that should be truncated in the middle"
	truncated := truncateMiddle(long, 20)
	assert.LessOrEqual(t, len(truncated), 21) // Allow for the 3-byte ellipsis
	assert.Contains(t, truncated, "…")

	// Very short max length
	assert.Equal(t, "ab", truncateMiddle("abcdef", 2))
}
