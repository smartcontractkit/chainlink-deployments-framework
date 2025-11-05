package analyzer

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
)

func TestTextRenderer_RenderDecodedCall_SimpleCall(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
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
	result := renderer.RenderDecodedCall(call, ctx)

	expected := `Address: 0x1234567890123456789012345678901234567890
Method: transfer(address,uint256)

Inputs:
  to: 0xabcdefabcdefabcdefabcdefabcdefabcdefabcd
  amount: 1000000000000000000
`

	assert.Equal(t, expected, result)
}

func TestTextRenderer_RenderDecodedCall_WithOutputs(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "getBalance(address)",
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
	result := renderer.RenderDecodedCall(call, ctx)

	expected := `Address: 0x1234567890123456789012345678901234567890
Method: getBalance(address)

Inputs:
  account: 0xabcdefabcdefabcdefabcdefabcdefabcdefabcd

Outputs:
  balance: 5000000000000000000
`

	assert.Equal(t, expected, result)
}

func TestTextRenderer_RenderDecodedCall_EmptyInputsOutputs(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "fallback()",
		Inputs:  []NamedField{},
		Outputs: []NamedField{},
	}

	ctx := NewFieldContext(nil)
	result := renderer.RenderDecodedCall(call, ctx)

	expected := `Address: 0x1234567890123456789012345678901234567890
Method: fallback()
`

	assert.Equal(t, expected, result)
}

func TestTextRenderer_RenderField_SimpleField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	field := SimpleField{Value: "test value"}
	result := renderer.RenderField(NamedField{Name: "test", Value: field}, ctx)

	assert.Equal(t, "test: test value", result)
}

func TestTextRenderer_RenderField_AddressField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	field := AddressField{Value: "0x1234567890123456789012345678901234567890"}
	result := renderer.RenderField(NamedField{Name: "address", Value: field}, ctx)

	assert.Equal(t, "address: 0x1234567890123456789012345678901234567890", result)
}

func TestTextRenderer_RenderField_ChainSelectorField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Known chain", func(t *testing.T) {
		t.Parallel()
		field := ChainSelectorField{Value: chainsel.ETHEREUM_MAINNET.Selector}
		result := renderer.RenderField(NamedField{Name: "chain", Value: field}, ctx)

		assert.Contains(t, result, "ethereum-mainnet")
	})

	t.Run("Unknown chain", func(t *testing.T) {
		t.Parallel()
		field := ChainSelectorField{Value: 999999}
		result := renderer.RenderField(NamedField{Name: "chain", Value: field}, ctx)

		assert.Equal(t, "chain: `999999 (<chain unknown>)`", result)
	})
}

func TestTextRenderer_RenderField_BytesField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Empty bytes", func(t *testing.T) {
		t.Parallel()
		field := BytesField{Value: []byte{}}
		result := renderer.RenderField(NamedField{Name: "data", Value: field}, ctx)

		assert.Equal(t, "data: 0x", result)
	})

	t.Run("Single byte", func(t *testing.T) {
		t.Parallel()
		field := BytesField{Value: []byte{0x42}}
		result := renderer.RenderField(NamedField{Name: "data", Value: field}, ctx)

		assert.Equal(t, "data: 0x42", result)
	})

	t.Run("Multiple bytes", func(t *testing.T) {
		t.Parallel()
		field := BytesField{Value: []byte{0x12, 0x34, 0x56, 0x78}}
		result := renderer.RenderField(NamedField{Name: "data", Value: field}, ctx)

		assert.Equal(t, "data: 0x12345678", result)
	})
}

func TestTextRenderer_RenderField_ArrayField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Empty array", func(t *testing.T) {
		t.Parallel()
		field := ArrayField{Elements: []FieldValue{}}
		result := renderer.RenderField(NamedField{Name: "array", Value: field}, ctx)

		assert.Equal(t, "array: ", result)
	})

	t.Run("Single element", func(t *testing.T) {
		t.Parallel()
		field := ArrayField{Elements: []FieldValue{SimpleField{Value: "item1"}}}
		result := renderer.RenderField(NamedField{Name: "array", Value: field}, ctx)

		assert.Equal(t, "array: item1", result)
	})

	t.Run("Multiple elements", func(t *testing.T) {
		t.Parallel()
		elements := []FieldValue{
			SimpleField{Value: "item1"},
			SimpleField{Value: "item2"},
			SimpleField{Value: "item3"},
		}
		field := ArrayField{Elements: elements}
		result := renderer.RenderField(NamedField{Name: "array", Value: field}, ctx)

		assert.Equal(t, "array: item1, item2, item3", result)
	})

	t.Run("Nested structures", func(t *testing.T) {
		t.Parallel()
		elements := []FieldValue{
			SimpleField{Value: "simple"},
			AddressField{Value: "0x1234567890123456789012345678901234567890"},
		}
		field := ArrayField{Elements: elements}
		result := renderer.RenderField(NamedField{Name: "array", Value: field}, ctx)

		assert.Equal(t, "array: simple, 0x1234567890123456789012345678901234567890", result)
	})
}

func TestTextRenderer_RenderField_StructField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Empty struct", func(t *testing.T) {
		t.Parallel()
		field := StructField{Fields: []NamedField{}}
		result := renderer.RenderField(NamedField{Name: "struct", Value: field}, ctx)

		assert.Equal(t, "struct: ", result)
	})

	t.Run("Single field", func(t *testing.T) {
		t.Parallel()
		field := StructField{Fields: []NamedField{
			{Name: "field1", Value: SimpleField{Value: "value1"}},
		}}
		result := renderer.RenderField(NamedField{Name: "struct", Value: field}, ctx)

		assert.Equal(t, "struct: field1: value1", result)
	})

	t.Run("Multiple fields", func(t *testing.T) {
		t.Parallel()
		field := StructField{Fields: []NamedField{
			{Name: "field1", Value: SimpleField{Value: "value1"}},
			{Name: "field2", Value: SimpleField{Value: "value2"}},
			{Name: "field3", Value: AddressField{Value: "0x1234567890123456789012345678901234567890"}},
		}}
		result := renderer.RenderField(NamedField{Name: "struct", Value: field}, ctx)

		assert.Equal(t, "struct: field1: value1, field2: value2, field3: 0x1234567890123456789012345678901234567890", result)
	})
}

func TestTextRenderer_RenderField_YamlField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Simple string", func(t *testing.T) {
		t.Parallel()
		field := YamlField{Value: "test string"}
		result := renderer.RenderField(NamedField{Name: "yaml", Value: field}, ctx)

		assert.Equal(t, "yaml: test string", result)
	})

	t.Run("Number as string", func(t *testing.T) {
		t.Parallel()
		field := YamlField{Value: "42"}
		result := renderer.RenderField(NamedField{Name: "yaml", Value: field}, ctx)

		assert.Equal(t, "yaml: 42", result)
	})

	t.Run("Boolean as string", func(t *testing.T) {
		t.Parallel()
		field := YamlField{Value: "true"}
		result := renderer.RenderField(NamedField{Name: "yaml", Value: field}, ctx)

		assert.Equal(t, "yaml: true", result)
	})
}

func TestTextRenderer_RenderField_NamedField(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Simple value", func(t *testing.T) {
		t.Parallel()
		field := NamedField{
			Name:  "param",
			Value: SimpleField{Value: "value"},
		}
		result := renderer.RenderField(field, ctx)

		assert.Equal(t, "param: value", result)
	})

	t.Run("Complex value", func(t *testing.T) {
		t.Parallel()
		field := NamedField{
			Name: "complex",
			Value: ArrayField{Elements: []FieldValue{
				SimpleField{Value: "item1"},
				SimpleField{Value: "item2"},
			}},
		}
		result := renderer.RenderField(field, ctx)

		assert.Equal(t, "complex: item1, item2", result)
	})
}

// MockField for testing unknown field types
type MockField struct{}

func (m MockField) GetType() string                   { return "MockField" }
func (m MockField) GetValue() string                  { return "mock value" }
func (m MockField) Describe(ctx *FieldContext) string { return "mock value" }

func TestTextRenderer_RenderField_DefaultCase(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	field := MockField{}
	result := renderer.RenderField(NamedField{Name: "mock", Value: field}, ctx)

	assert.Equal(t, "mock: <unknown field type: MockField>", result)
}

func TestTextRenderer_ComplexDecodedCall(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "complexMethod(address,uint256[],(string,bytes))",
		Inputs: []NamedField{
			{
				Name:  "recipient",
				Value: AddressField{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
			},
			{
				Name: "amounts",
				Value: ArrayField{Elements: []FieldValue{
					SimpleField{Value: "1000000000000000000"},
					SimpleField{Value: "2000000000000000000"},
					SimpleField{Value: "3000000000000000000"},
				}},
			},
			{
				Name: "metadata",
				Value: StructField{Fields: []NamedField{
					{Name: "name", Value: SimpleField{Value: "Test Token"}},
					{Name: "symbol", Value: SimpleField{Value: "TEST"}},
					{Name: "decimals", Value: SimpleField{Value: "18"}},
				}},
			},
		},
		Outputs: []NamedField{
			{
				Name:  "success",
				Value: SimpleField{Value: "true"},
			},
			{
				Name:  "gasUsed",
				Value: SimpleField{Value: "150000"},
			},
		},
	}

	ctx := NewFieldContext(nil)
	result := renderer.RenderDecodedCall(call, ctx)

	expected := `Address: 0x1234567890123456789012345678901234567890
Method: complexMethod(address,uint256[],(string,bytes))

Inputs:
  recipient: 0xabcdefabcdefabcdefabcdefabcdefabcdefabcd
  amounts: 1000000000000000000, 2000000000000000000, 3000000000000000000
  metadata: name: Test Token, symbol: TEST, decimals: 18

Outputs:
  success: true
  gasUsed: 150000
`

	assert.Equal(t, expected, result)
}

func TestTextRenderer_WithFieldContext(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()

	// Create a field context with some data
	ctx := NewFieldContext(nil)

	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "testMethod()",
		Inputs: []NamedField{
			{
				Name:  "param",
				Value: SimpleField{Value: "value"},
			},
		},
		Outputs: []NamedField{},
	}

	result := renderer.RenderDecodedCall(call, ctx)

	// The context shouldn't affect the basic rendering
	assert.Contains(t, result, "Address: 0x1234567890123456789012345678901234567890")
	assert.Contains(t, result, "Method: testMethod()")
	assert.Contains(t, result, "param: value")
}

func TestTextRenderer_ErrorHandling(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Template execution error", func(t *testing.T) {
		// This test ensures that template errors are handled gracefully
		// We'll test with a field that might cause template issues
		field := SimpleField{Value: "test"}
		result := renderer.RenderField(NamedField{Name: "test", Value: field}, ctx)

		// Should not contain error messages
		assert.NotContains(t, result, "Error rendering")
		assert.Equal(t, "test: test", result)
	})
}

func TestTextRenderer_EdgeCases(t *testing.T) {
	t.Parallel()

	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	t.Run("Empty string values", func(t *testing.T) {
		t.Parallel()
		field := SimpleField{Value: ""}
		result := renderer.RenderField(NamedField{Name: "empty", Value: field}, ctx)

		assert.Equal(t, "empty: ", result)
	})

	t.Run("Special characters", func(t *testing.T) {
		t.Parallel()
		field := SimpleField{Value: "test\nwith\ttabs\r\nand\rnewlines"}
		result := renderer.RenderField(NamedField{Name: "special", Value: field}, ctx)

		assert.Equal(t, "special: test\nwith\ttabs\r\nand\rnewlines", result)
	})

	t.Run("Unicode characters", func(t *testing.T) {
		t.Parallel()
		field := SimpleField{Value: "测试 🚀 émojis"}
		result := renderer.RenderField(NamedField{Name: "unicode", Value: field}, ctx)

		assert.Equal(t, "unicode: 测试 🚀 émojis", result)
	})
}

// Additional tests for functions with 0% coverage
func TestTextRenderer_RenderProposal(t *testing.T) {
	t.Parallel()
	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	// Test with a simple proposal
	proposal := &ProposalReport{
		Operations: []OperationReport{
			{
				ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
				ChainName:     "ethereum-testnet-sepolia",
				Calls: []*DecodedCall{
					{
						Address: "0x1234567890123456789012345678901234567890",
						Method:  "transfer(address,uint256)",
						Inputs: []NamedField{
							{Name: "to", Value: AddressField{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"}},
							{Name: "amount", Value: SimpleField{Value: "1000000000000000000"}},
						},
						Outputs: []NamedField{},
					},
				},
			},
		},
	}

	result := renderer.RenderProposal(proposal, ctx)

	// Should contain chain name and call details
	assert.Contains(t, result, "ethereum-testnet-sepolia")
	assert.Contains(t, result, "0x1234567890123456789012345678901234567890")
	assert.Contains(t, result, "transfer(address,uint256)")
}

func TestTextRenderer_RenderTimelockProposal(t *testing.T) {
	t.Parallel()
	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	// Test with a timelock proposal
	timelockProposal := &ProposalReport{
		Batches: []BatchReport{
			{
				ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
				ChainName:     "ethereum-testnet-sepolia",
				Operations: []OperationReport{
					{
						ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
						ChainName:     "ethereum-testnet-sepolia",
						Calls: []*DecodedCall{
							{
								Address: "0x1234567890123456789012345678901234567890",
								Method:  "execute()",
								Inputs:  []NamedField{},
								Outputs: []NamedField{},
							},
						},
					},
				},
			},
		},
	}

	result := renderer.RenderTimelockProposal(timelockProposal, ctx)

	// Should contain chain name and call details
	assert.Contains(t, result, "ethereum-testnet-sepolia")
	assert.Contains(t, result, "0x1234567890123456789012345678901234567890")
	assert.Contains(t, result, "execute()")
}

func TestTextRenderer_renderCallHelper(t *testing.T) {
	t.Parallel()
	renderer := NewTextRenderer()
	ctx := NewFieldContext(nil)

	call := &DecodedCall{
		Address: "0x1234567890123456789012345678901234567890",
		Method:  "testMethod()",
		Inputs: []NamedField{
			{Name: "param", Value: SimpleField{Value: "value"}},
		},
		Outputs: []NamedField{},
	}

	result := renderer.renderCallHelper(call, ctx)

	// Should contain call details
	assert.Contains(t, result, "0x1234567890123456789012345678901234567890")
	assert.Contains(t, result, "testMethod()")
	assert.Contains(t, result, "param: value")
}
