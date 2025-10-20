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
		Inputs: []NamedDescriptor{
			{
				Name:  "to",
				Value: AddressDescriptor{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
			},
			{
				Name:  "amount",
				Value: SimpleDescriptor{Value: "1000000000000000000"},
			},
		},
		Outputs: []NamedDescriptor{},
	}

	ctx := NewArgumentContext(nil)
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
		Inputs: []NamedDescriptor{
			{
				Name:  "account",
				Value: AddressDescriptor{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
			},
		},
		Outputs: []NamedDescriptor{
			{
				Name:  "balance",
				Value: SimpleDescriptor{Value: "5000000000000000000"},
			},
		},
	}

	ctx := NewArgumentContext(nil)
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
		Inputs:  []NamedDescriptor{},
		Outputs: []NamedDescriptor{},
	}

	// Create context with address mapping
	addresses := deployment.AddressesByChain{
		1: {
			"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
		},
	}
	ctx := NewArgumentContext(addresses)
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
		Inputs:  []NamedDescriptor{},
		Outputs: []NamedDescriptor{},
	}

	ctx := NewArgumentContext(nil)
	output := renderer.RenderDecodedCall(call, ctx)

	assert.Contains(t, output, "**Address:** `0x1234567890123456789012345678901234567890`")
	assert.Contains(t, output, "**Method:** `execute()`")
	assert.NotContains(t, output, "**Inputs:**")
	assert.NotContains(t, output, "**Outputs:**")
}

// Tests for MarkdownRenderer descriptor handling methods

func TestMarkdownRenderer_SummarizeArgument_SimpleDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	// Short string
	summary, details := renderer.summarizeDescriptor("name", SimpleDescriptor{Value: "short"}, ctx)
	assert.Equal(t, "`short`", summary)
	assert.Empty(t, details)

	// Long string
	longValue := "this is a very long string that should trigger details section because it exceeds the 80 character limit"
	summary, details = renderer.summarizeDescriptor("longName", SimpleDescriptor{Value: longValue}, ctx)
	assert.Contains(t, summary, "`this is a very long string that should …cause it exceeds the 80 character limit` (len=")
	assert.Contains(t, details, "<details><summary>longName</summary>")
	assert.Contains(t, details, "```")
	assert.Contains(t, details, longValue)
}

func TestMarkdownRenderer_SummarizeArgument_AddressDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	summary, details := renderer.summarizeDescriptor("addr", AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"}, ctx)
	assert.Equal(t, "`0x1234567890123456789012345678901234567890`", summary)
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_ChainSelectorDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	summary, details := renderer.summarizeDescriptor("chain", ChainSelectorDescriptor{Value: 1}, ctx)
	assert.Contains(t, summary, "`1 (<chain unknown>)`")
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_BytesDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	// Small bytes
	smallBytes := []byte{0x01, 0x02, 0x03}
	summary, details := renderer.summarizeDescriptor("small", BytesDescriptor{Value: smallBytes}, ctx)
	assert.Contains(t, summary, "bytes(len=3):")
	assert.Contains(t, details, "<details><summary>small</summary>")
	assert.Contains(t, details, "```")
	assert.Contains(t, details, "0x010203")

	// Large bytes
	largeBytes := make([]byte, 50)
	for i := range largeBytes {
		largeBytes[i] = byte(i)
	}
	summary, details = renderer.summarizeDescriptor("large", BytesDescriptor{Value: largeBytes}, ctx)
	assert.Contains(t, summary, "bytes(len=50):")
	assert.Contains(t, details, "<details><summary>large</summary>")
}

func TestMarkdownRenderer_SummarizeArgument_ArrayDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	// Empty array
	summary, details := renderer.summarizeDescriptor("empty", ArrayDescriptor{Elements: []Descriptor{}}, ctx)
	assert.Equal(t, "[]", summary)
	assert.Empty(t, details)

	// Non-empty array
	elements := []Descriptor{
		SimpleDescriptor{Value: "item1"},
		SimpleDescriptor{Value: "item2"},
		SimpleDescriptor{Value: "item3"},
	}
	summary, details = renderer.summarizeDescriptor("array", ArrayDescriptor{Elements: elements}, ctx)
	assert.Contains(t, summary, "array[3]")
	assert.Contains(t, details, "<details><summary>array</summary>")
	assert.Contains(t, details, "```")
}

func TestMarkdownRenderer_SummarizeArgument_StructDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	fields := []NamedDescriptor{
		{Name: "field1", Value: SimpleDescriptor{Value: "value1"}},
		{Name: "field2", Value: SimpleDescriptor{Value: "value2"}},
	}
	summary, details := renderer.summarizeDescriptor("struct", StructDescriptor{Fields: fields}, ctx)
	assert.Contains(t, summary, "struct{2 fields}")
	assert.Contains(t, details, "<details><summary>struct</summary>")
	assert.Contains(t, details, "```")
}

func TestMarkdownRenderer_SummarizeArgument_NamedDescriptor(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	named := NamedDescriptor{
		Name:  "param",
		Value: SimpleDescriptor{Value: "value"},
	}
	summary, details := renderer.summarizeDescriptor("named", named, ctx)
	assert.Contains(t, summary, "`param: value`")
	assert.Empty(t, details)
}

func TestMarkdownRenderer_SummarizeArgument_DefaultCase(t *testing.T) {
	t.Parallel()

	renderer := NewMarkdownRenderer()
	ctx := NewArgumentContext(nil)

	// Test with a descriptor that doesn't match any specific case
	// This should fall through to the default case
	customDesc := &customDescriptor{value: "custom value"}
	summary, details := renderer.summarizeDescriptor("custom", customDesc, ctx)
	assert.Contains(t, summary, "`custom value`")
	assert.Empty(t, details)
}

// Helper type for testing default case
type customDescriptor struct {
	value string
}

func (c *customDescriptor) Describe(ctx *DescriptorContext) string {
	return c.value
}

// Tests for helper functions

func TestArrayPreview(t *testing.T) {
	t.Parallel()

	ctx := NewArgumentContext(nil)

	// Empty array
	elements := []Descriptor{}
	preview := arrayPreview(elements, ctx)
	assert.Empty(t, preview)

	// Small array
	elements = []Descriptor{
		SimpleDescriptor{Value: "item1"},
		SimpleDescriptor{Value: "item2"},
	}
	preview = arrayPreview(elements, ctx)
	assert.Contains(t, preview, ": [item1, item2]")

	// Large array (should truncate)
	elements = []Descriptor{
		SimpleDescriptor{Value: "item1"},
		SimpleDescriptor{Value: "item2"},
		SimpleDescriptor{Value: "item3"},
		SimpleDescriptor{Value: "item4"},
		SimpleDescriptor{Value: "item5"},
	}
	preview = arrayPreview(elements, ctx)
	assert.Contains(t, preview, ": [item1, item2, item3, … (+2)]")
}

func TestCompactValue(t *testing.T) {
	t.Parallel()

	ctx := NewArgumentContext(nil)

	// Test different descriptor types
	assert.Equal(t, "0x1234567890123456789012345678901234567890", compactValue(AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"}, ctx))
	assert.Contains(t, compactValue(ChainSelectorDescriptor{Value: 1}, ctx), "1 (<chain unknown>)")
	assert.Contains(t, compactValue(BytesDescriptor{Value: []byte{0x01, 0x02}}, ctx), "0x0102")
	assert.Equal(t, "short", compactValue(SimpleDescriptor{Value: "short"}, ctx))
	assert.Equal(t, "struct", compactValue(StructDescriptor{Fields: []NamedDescriptor{}}, ctx))
	assert.Contains(t, compactValue(ArrayDescriptor{Elements: []Descriptor{}}, ctx), "array[0]")
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
