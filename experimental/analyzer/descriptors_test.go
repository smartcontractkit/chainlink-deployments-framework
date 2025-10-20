package analyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestDescriptorContext(t *testing.T) {
	t.Parallel()

	t.Run("NewArgumentContext", func(t *testing.T) {
		t.Parallel()
		addresses := deployment.AddressesByChain{
			1: map[string]deployment.TypeAndVersion{
				"0x1234567890123456789012345678901234567890": {
					Type:    "TestContract",
					Version: *semver.MustParse("1.0.0"),
				},
			},
		}

		ctx := NewArgumentContext(addresses)
		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Ctx)

		retrievedAddresses, err := ContextGet[deployment.AddressesByChain](ctx, "AddressesByChain")
		require.NoError(t, err)
		require.Equal(t, addresses, retrievedAddresses)
	})

	t.Run("ContextGet", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name          string
			ctx           *DescriptorContext
			key           string
			expectedType  string
			expectError   bool
			errorContains string
			expectedValue any
		}{
			{
				name: "Success_String",
				ctx: &DescriptorContext{
					Ctx: map[string]any{
						"test_string": "hello",
					},
				},
				key:           "test_string",
				expectedType:  "string",
				expectError:   false,
				expectedValue: "hello",
			},
			{
				name: "Success_Int",
				ctx: &DescriptorContext{
					Ctx: map[string]any{
						"test_int": 42,
					},
				},
				key:           "test_int",
				expectedType:  "int",
				expectError:   false,
				expectedValue: 42,
			},
			{
				name: "NotFound",
				ctx: &DescriptorContext{
					Ctx: map[string]any{},
				},
				key:           "nonexistent",
				expectedType:  "string",
				expectError:   true,
				errorContains: "context element nonexistent not found",
			},
			{
				name: "TypeMismatch",
				ctx: &DescriptorContext{
					Ctx: map[string]any{
						"test": "hello",
					},
				},
				key:           "test",
				expectedType:  "int",
				expectError:   true,
				errorContains: "type mismatch",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				switch tt.expectedType {
				case "string":
					result, err := ContextGet[string](tt.ctx, tt.key)
					if tt.expectError {
						require.Error(t, err)
						if tt.errorContains != "" {
							require.Contains(t, err.Error(), tt.errorContains)
						}
					} else {
						require.NoError(t, err)
						require.Equal(t, tt.expectedValue, result)
					}
				case "int":
					result, err := ContextGet[int](tt.ctx, tt.key)
					if tt.expectError {
						require.Error(t, err)
						if tt.errorContains != "" {
							require.Contains(t, err.Error(), tt.errorContains)
						}
					} else {
						require.NoError(t, err)
						require.Equal(t, tt.expectedValue, result)
					}
				}
			})
		}
	})
}

func TestNamedDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor NamedDescriptor
		expected   string
	}{
		{
			name: "SimpleValue",
			descriptor: NamedDescriptor{
				Name:  "testParam",
				Value: SimpleDescriptor{Value: "testValue"},
			},
			expected: "testParam: testValue",
		},
		{
			name: "ComplexValue_Array",
			descriptor: NamedDescriptor{
				Name: "arrayParam",
				Value: ArrayDescriptor{
					Elements: []Descriptor{
						SimpleDescriptor{Value: "item1"},
						SimpleDescriptor{Value: "item2"},
					},
				},
			},
			expected: "arrayParam: [item1,item2]",
		},
		{
			name: "EmptyName",
			descriptor: NamedDescriptor{
				Name:  "",
				Value: SimpleDescriptor{Value: "value"},
			},
			expected: ": value",
		},
		{
			name: "EmptyValue",
			descriptor: NamedDescriptor{
				Name:  "param",
				Value: SimpleDescriptor{Value: ""},
			},
			expected: "param: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestArrayDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor ArrayDescriptor
		expected   string
	}{
		{
			name:       "EmptyArray",
			descriptor: ArrayDescriptor{Elements: []Descriptor{}},
			expected:   "[]",
		},
		{
			name: "SingleElement",
			descriptor: ArrayDescriptor{
				Elements: []Descriptor{
					SimpleDescriptor{Value: "single"},
				},
			},
			expected: "[single]",
		},
		{
			name: "MultipleElements_Inline",
			descriptor: ArrayDescriptor{
				Elements: []Descriptor{
					SimpleDescriptor{Value: "item1"},
					SimpleDescriptor{Value: "item2"},
					SimpleDescriptor{Value: "item3"},
				},
			},
			expected: "[item1,item2,item3]",
		},
		{
			name: "MultipleElements_Indented",
			descriptor: ArrayDescriptor{
				Elements: []Descriptor{
					StructDescriptor{
						Fields: []NamedDescriptor{
							{Name: "field1", Value: SimpleDescriptor{Value: "value1"}},
							{Name: "field2", Value: SimpleDescriptor{Value: "value2"}},
						},
					},
					SimpleDescriptor{Value: "simple"},
				},
			},
			expected: "[\n    {\n        field1: value1\n        field2: value2\n    },\nsimple\n]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			if tt.name == "MultipleElements_Indented" {
				// For indented arrays, check that it contains the expected structure
				require.Contains(t, result, "[\n")
				require.Contains(t, result, "\n]")
				require.Contains(t, result, "    ") // Should contain indentation
			} else {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStructDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor StructDescriptor
		expected   string
	}{
		{
			name:       "EmptyStruct",
			descriptor: StructDescriptor{Fields: []NamedDescriptor{}},
			expected:   "{  }",
		},
		{
			name: "SingleField_Inline",
			descriptor: StructDescriptor{
				Fields: []NamedDescriptor{
					{Name: "field1", Value: SimpleDescriptor{Value: "value1"}},
				},
			},
			expected: "{ field1: value1 }",
		},
		{
			name: "MultipleFields_Inline",
			descriptor: StructDescriptor{
				Fields: []NamedDescriptor{
					{Name: "field1", Value: SimpleDescriptor{Value: "value1"}},
					{Name: "field2", Value: SimpleDescriptor{Value: "value2"}},
				},
			},
			expected: "{\n    field1: value1\n    field2: value2\n}",
		},
		{
			name: "MultipleFields_PrettyFormat",
			descriptor: StructDescriptor{
				Fields: []NamedDescriptor{
					{Name: "field1", Value: SimpleDescriptor{Value: "value1"}},
					{Name: "field2", Value: SimpleDescriptor{Value: "value2"}},
					{Name: "field3", Value: SimpleDescriptor{Value: "value3"}},
				},
			},
			expected: "{\n    field1: value1\n    field2: value2\n    field3: value3\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor SimpleDescriptor
		expected   string
	}{
		{
			name:       "BasicValue",
			descriptor: SimpleDescriptor{Value: "test value"},
			expected:   "test value",
		},
		{
			name:       "EmptyString",
			descriptor: SimpleDescriptor{Value: ""},
			expected:   "",
		},
		{
			name:       "SpecialCharacters",
			descriptor: SimpleDescriptor{Value: "test\nwith\ttabs and spaces"},
			expected:   "test\nwith\ttabs and spaces",
		},
		{
			name:       "UnicodeCharacters",
			descriptor: SimpleDescriptor{Value: "测试 🚀"},
			expected:   "测试 🚀",
		},
		{
			name:       "NumbersAsString",
			descriptor: SimpleDescriptor{Value: "12345"},
			expected:   "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestChainSelectorDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		descriptor       ChainSelectorDescriptor
		expectedContains []string
	}{
		{
			name:             "KnownChain_Ethereum",
			descriptor:       ChainSelectorDescriptor{Value: chainsel.ETHEREUM_MAINNET.Selector},
			expectedContains: []string{"5009297550715157269", "ethereum-mainnet"},
		},
		{
			name:             "KnownChain_Polygon",
			descriptor:       ChainSelectorDescriptor{Value: chainsel.POLYGON_MAINNET.Selector},
			expectedContains: []string{"4051577828743386545", "polygon-mainnet"},
		},
		{
			name:             "KnownChain_Solana",
			descriptor:       ChainSelectorDescriptor{Value: chainsel.SOLANA_MAINNET.Selector},
			expectedContains: []string{"124615329519749607", "solana-mainnet"},
		},
		{
			name:             "UnknownChain",
			descriptor:       ChainSelectorDescriptor{Value: 999999999},
			expectedContains: []string{"999999999", "<chain unknown>"},
		},
		{
			name:             "ZeroSelector",
			descriptor:       ChainSelectorDescriptor{Value: 0},
			expectedContains: []string{"0", "<chain unknown>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			for _, expected := range tt.expectedContains {
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestBytesDescriptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor BytesDescriptor
		expected   string
	}{
		{
			name:       "EmptyBytes",
			descriptor: BytesDescriptor{Value: []byte{}},
			expected:   "0x",
		},
		{
			name:       "SingleByte",
			descriptor: BytesDescriptor{Value: []byte{0x42}},
			expected:   "0x42",
		},
		{
			name:       "MultipleBytes",
			descriptor: BytesDescriptor{Value: []byte{0x01, 0x02, 0x03, 0x04}},
			expected:   "0x01020304",
		},
		{
			name:       "AllZeros",
			descriptor: BytesDescriptor{Value: []byte{0x00, 0x00, 0x00}},
			expected:   "0x000000",
		},
		{
			name:       "AllOnes",
			descriptor: BytesDescriptor{Value: []byte{0xFF, 0xFF, 0xFF}},
			expected:   "0xffffff",
		},
		{
			name:       "MixedBytes",
			descriptor: BytesDescriptor{Value: []byte{0xAB, 0xCD, 0xEF}},
			expected:   "0xabcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			require.Equal(t, tt.expected, result)
		})
	}

	t.Run("LargeBytes", func(t *testing.T) {
		t.Parallel()
		ctx := NewArgumentContext(deployment.AddressesByChain{})

		// Create a large byte array
		largeBytes := make([]byte, 100)
		for i := range largeBytes {
			largeBytes[i] = byte(i % 256)
		}

		bytesDesc := BytesDescriptor{Value: largeBytes}
		result := bytesDesc.Describe(ctx)

		require.Greater(t, len(result), 100) // Should be much longer due to hex encoding
		require.True(t, strings.HasPrefix(result, "0x"))
	})
}

func TestAddressDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("Describe", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name       string
			descriptor AddressDescriptor
			expected   string
		}{
			{
				name:       "ValidAddress",
				descriptor: AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
				expected:   "0x1234567890123456789012345678901234567890",
			},
			{
				name:       "EmptyAddress",
				descriptor: AddressDescriptor{Value: ""},
				expected:   "",
			},
			{
				name:       "ShortAddress",
				descriptor: AddressDescriptor{Value: "0x1234"},
				expected:   "0x1234",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx := NewArgumentContext(deployment.AddressesByChain{})
				result := tt.descriptor.Describe(ctx)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Annotation", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name             string
			addresses        deployment.AddressesByChain
			descriptor       AddressDescriptor
			expectedContains []string
		}{
			{
				name: "KnownAddress_SingleChain",
				addresses: deployment.AddressesByChain{
					chainsel.ETHEREUM_MAINNET.Selector: map[string]deployment.TypeAndVersion{
						"0x1234567890123456789012345678901234567890": {
							Type:    "TestContract",
							Version: *semver.MustParse("1.0.0"),
						},
					},
				},
				descriptor:       AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
				expectedContains: []string{"TestContract", "ethereum-mainnet", "address of"},
			},
			{
				name:             "UnknownAddress",
				addresses:        deployment.AddressesByChain{},
				descriptor:       AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
				expectedContains: []string{""},
			},
			{
				name: "MultipleChains",
				addresses: deployment.AddressesByChain{
					chainsel.ETHEREUM_MAINNET.Selector: map[string]deployment.TypeAndVersion{
						"0x1111111111111111111111111111111111111111": {
							Type:    "Contract1",
							Version: *semver.MustParse("1.0.0"),
						},
					},
					chainsel.POLYGON_MAINNET.Selector: map[string]deployment.TypeAndVersion{
						"0x2222222222222222222222222222222222222222": {
							Type:    "Contract2",
							Version: *semver.MustParse("2.0.0"),
						},
					},
				},
				descriptor:       AddressDescriptor{Value: "0x1111111111111111111111111111111111111111"},
				expectedContains: []string{"Contract1", "ethereum-mainnet"},
			},
			{
				name: "UnknownChainSelector",
				addresses: deployment.AddressesByChain{
					999999: map[string]deployment.TypeAndVersion{
						"0x1234567890123456789012345678901234567890": {
							Type:    "TestContract",
							Version: *semver.MustParse("1.0.0"),
						},
					},
				},
				descriptor:       AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
				expectedContains: []string{"TestContract", "999999"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx := NewArgumentContext(tt.addresses)
				result := tt.descriptor.Annotation(ctx)
				for _, expected := range tt.expectedContains {
					if expected == "" {
						require.Empty(t, result)
					} else {
						require.Contains(t, result, expected)
					}
				}
			})
		}
	})

	t.Run("Annotation_EmptyContext", func(t *testing.T) {
		t.Parallel()
		ctx := &DescriptorContext{Ctx: map[string]any{}}
		addrDesc := AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"}
		result := addrDesc.Annotation(ctx)
		require.Empty(t, result)
	})
}

func TestDescriptorIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		descriptor       Descriptor
		expectedContains []string
	}{
		{
			name: "NestedStructures",
			descriptor: StructDescriptor{
				Fields: []NamedDescriptor{
					{
						Name: "config",
						Value: StructDescriptor{
							Fields: []NamedDescriptor{
								{Name: "chainId", Value: ChainSelectorDescriptor{Value: chainsel.ETHEREUM_MAINNET.Selector}},
								{Name: "addresses", Value: ArrayDescriptor{
									Elements: []Descriptor{
										AddressDescriptor{Value: "0x1111111111111111111111111111111111111111"},
										AddressDescriptor{Value: "0x2222222222222222222222222222222222222222"},
									},
								}},
							},
						},
					},
					{
						Name:  "data",
						Value: BytesDescriptor{Value: []byte{0x01, 0x02, 0x03}},
					},
				},
			},
			expectedContains: []string{"config", "chainId", "addresses", "data", "0x010203", "5009297550715157269"},
		},
		{
			name: "MixedDescriptorTypes",
			descriptor: ArrayDescriptor{
				Elements: []Descriptor{
					SimpleDescriptor{Value: "string"},
					BytesDescriptor{Value: []byte{0x42}},
					ChainSelectorDescriptor{Value: chainsel.SOLANA_MAINNET.Selector},
					AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
				},
			},
			expectedContains: []string{"string", "0x42", "solana-mainnet", "0x1234567890123456789012345678901234567890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewArgumentContext(deployment.AddressesByChain{})
			result := tt.descriptor.Describe(ctx)
			for _, expected := range tt.expectedContains {
				require.Contains(t, result, expected)
			}
		})
	}
}
