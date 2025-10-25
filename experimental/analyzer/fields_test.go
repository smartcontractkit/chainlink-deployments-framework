package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestFieldContext(t *testing.T) {
	t.Parallel()

	t.Run("NewFieldContext", func(t *testing.T) {
		t.Parallel()
		addresses := deployment.AddressesByChain{
			chainsel.ETHEREUM_MAINNET.Selector: {
				"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
			},
		}
		ctx := NewFieldContext(addresses)
		assert.NotNil(t, ctx)
		assert.NotNil(t, ctx.Ctx)
	})

	t.Run("FieldContextGet", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			ctx      *FieldContext
			key      string
			expected any
			wantErr  bool
		}{
			{
				name: "Success_String",
				ctx: &FieldContext{
					Ctx: map[string]any{
						"test": "value",
					},
				},
				key:      "test",
				expected: "value",
				wantErr:  false,
			},
			{
				name: "Success_Int",
				ctx: &FieldContext{
					Ctx: map[string]any{
						"number": 42,
					},
				},
				key:      "number",
				expected: 42,
				wantErr:  false,
			},
			{
				name: "NotFound",
				ctx: &FieldContext{
					Ctx: map[string]any{},
				},
				key:      "missing",
				expected: nil,
				wantErr:  true,
			},
			{
				name: "TypeMismatch",
				ctx: &FieldContext{
					Ctx: map[string]any{
						"test": "string",
					},
				},
				key:      "test",
				expected: 0,
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var result any
				var err error

				switch tt.name {
				case "Success_String":
					result, err = FieldContextGet[string](tt.ctx, tt.key)
				case "Success_Int":
					result, err = FieldContextGet[int](tt.ctx, tt.key)
				case "NotFound":
					result, err = FieldContextGet[string](tt.ctx, tt.key)
				case "TypeMismatch":
					result, err = FieldContextGet[int](tt.ctx, tt.key)
				}

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result)
				}
			})
		}
	})
}

func TestNamedField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    NamedField
		expected string
	}{
		{
			name: "SimpleValue",
			field: NamedField{
				Name:  "test",
				Value: SimpleField{Value: "testValue"},
			},
			expected: "test",
		},
		{
			name: "ComplexValue_Array",
			field: NamedField{
				Name: "array",
				Value: ArrayField{
					Elements: []FieldValue{
						SimpleField{Value: "item1"},
						SimpleField{Value: "item2"},
					},
				},
			},
			expected: "array",
		},
		{
			name: "EmptyName",
			field: NamedField{
				Name:  "",
				Value: SimpleField{Value: "value"},
			},
			expected: "",
		},
		{
			name: "EmptyValue",
			field: NamedField{
				Name:  "test",
				Value: SimpleField{Value: ""},
			},
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.Name)
			assert.NotNil(t, tt.field.Value)
		})
	}
}

func TestArrayField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    ArrayField
		expected int
	}{
		{
			name:     "EmptyArray",
			field:    ArrayField{Elements: []FieldValue{}},
			expected: 0,
		},
		{
			name: "SingleElement",
			field: ArrayField{
				Elements: []FieldValue{
					SimpleField{Value: "single"},
				},
			},
			expected: 1,
		},
		{
			name: "MultipleElements",
			field: ArrayField{
				Elements: []FieldValue{
					SimpleField{Value: "item1"},
					SimpleField{Value: "item2"},
					SimpleField{Value: "item3"},
				},
			},
			expected: 3,
		},
		{
			name: "NestedStructures",
			field: ArrayField{
				Elements: []FieldValue{
					StructField{
						Fields: []NamedField{
							{Name: "field1", Value: SimpleField{Value: "value1"}},
							{Name: "field2", Value: SimpleField{Value: "value2"}},
						},
					},
					SimpleField{Value: "simple"},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetLength())
			assert.Equal(t, "ArrayField", tt.field.GetType())
		})
	}
}

func TestStructField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    StructField
		expected int
	}{
		{
			name:     "EmptyStruct",
			field:    StructField{Fields: []NamedField{}},
			expected: 0,
		},
		{
			name: "SingleField",
			field: StructField{
				Fields: []NamedField{
					{Name: "field1", Value: SimpleField{Value: "value1"}},
				},
			},
			expected: 1,
		},
		{
			name: "MultipleFields",
			field: StructField{
				Fields: []NamedField{
					{Name: "field1", Value: SimpleField{Value: "value1"}},
					{Name: "field2", Value: SimpleField{Value: "value2"}},
				},
			},
			expected: 2,
		},
		{
			name: "ManyFields",
			field: StructField{
				Fields: []NamedField{
					{Name: "field1", Value: SimpleField{Value: "value1"}},
					{Name: "field2", Value: SimpleField{Value: "value2"}},
					{Name: "field3", Value: SimpleField{Value: "value3"}},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Len(t, tt.field.Fields, tt.expected)
			assert.Equal(t, "StructField", tt.field.GetType())
		})
	}
}

func TestSimpleField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    SimpleField
		expected string
	}{
		{
			name:     "BasicValue",
			field:    SimpleField{Value: "test value"},
			expected: "test value",
		},
		{
			name:     "EmptyString",
			field:    SimpleField{Value: ""},
			expected: "",
		},
		{
			name:     "SpecialCharacters",
			field:    SimpleField{Value: "test\nwith\ttabs and spaces"},
			expected: "test\nwith\ttabs and spaces",
		},
		{
			name:     "UnicodeCharacters",
			field:    SimpleField{Value: "测试 🚀"},
			expected: "测试 🚀",
		},
		{
			name:     "NumbersAsString",
			field:    SimpleField{Value: "12345"},
			expected: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetValue())
			assert.Equal(t, "SimpleField", tt.field.GetType())
		})
	}
}

func TestChainSelectorField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    ChainSelectorField
		expected uint64
	}{
		{
			name:     "KnownChain_Ethereum",
			field:    ChainSelectorField{Value: chainsel.ETHEREUM_MAINNET.Selector},
			expected: chainsel.ETHEREUM_MAINNET.Selector,
		},
		{
			name:     "KnownChain_Polygon",
			field:    ChainSelectorField{Value: chainsel.POLYGON_MAINNET.Selector},
			expected: chainsel.POLYGON_MAINNET.Selector,
		},
		{
			name:     "KnownChain_Solana",
			field:    ChainSelectorField{Value: chainsel.SOLANA_MAINNET.Selector},
			expected: chainsel.SOLANA_MAINNET.Selector,
		},
		{
			name:     "UnknownChain",
			field:    ChainSelectorField{Value: 999999999},
			expected: 999999999,
		},
		{
			name:     "ZeroSelector",
			field:    ChainSelectorField{Value: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetValue())
			assert.Equal(t, "ChainSelectorField", tt.field.GetType())
		})
	}
}

func TestBytesField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    BytesField
		expected []byte
	}{
		{
			name:     "EmptyBytes",
			field:    BytesField{Value: []byte{}},
			expected: []byte{},
		},
		{
			name:     "SingleByte",
			field:    BytesField{Value: []byte{0x42}},
			expected: []byte{0x42},
		},
		{
			name:     "MultipleBytes",
			field:    BytesField{Value: []byte{0x01, 0x02, 0x03, 0x04}},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "AllZeros",
			field:    BytesField{Value: []byte{0x00, 0x00, 0x00}},
			expected: []byte{0x00, 0x00, 0x00},
		},
		{
			name:     "AllOnes",
			field:    BytesField{Value: []byte{0xFF, 0xFF, 0xFF}},
			expected: []byte{0xFF, 0xFF, 0xFF},
		},
		{
			name:     "MixedBytes",
			field:    BytesField{Value: []byte{0xAB, 0xCD, 0xEF}},
			expected: []byte{0xAB, 0xCD, 0xEF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetValue())
			assert.Equal(t, len(tt.expected), tt.field.GetLength())
			assert.Equal(t, "BytesField", tt.field.GetType())
		})
	}

	t.Run("LargeBytes", func(t *testing.T) {
		t.Parallel()
		largeBytes := make([]byte, 1000)
		for i := range largeBytes {
			largeBytes[i] = byte(i % 256)
		}
		bytesField := BytesField{Value: largeBytes}
		assert.Equal(t, 1000, bytesField.GetLength())
		assert.Equal(t, largeBytes, bytesField.GetValue())
	})
}

func TestAddressField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    AddressField
		expected string
	}{
		{
			name:     "ValidAddress",
			field:    AddressField{Value: "0x1234567890123456789012345678901234567890"},
			expected: "0x1234567890123456789012345678901234567890",
		},
		{
			name:     "EmptyAddress",
			field:    AddressField{Value: ""},
			expected: "",
		},
		{
			name:     "ShortAddress",
			field:    AddressField{Value: "0x1234"},
			expected: "0x1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetValue())
			assert.Equal(t, "AddressField", tt.field.GetType())
		})
	}

	t.Run("Annotation", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name      string
			field     AddressField
			addresses deployment.AddressesByChain
			expected  string
		}{
			{
				name:  "KnownAddress_SingleChain",
				field: AddressField{Value: "0x1234567890123456789012345678901234567890"},
				addresses: deployment.AddressesByChain{
					chainsel.ETHEREUM_MAINNET.Selector: {
						"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
					},
				},
				expected: "address of Token 1.0.0 from ethereum-mainnet",
			},
			{
				name:  "UnknownAddress",
				field: AddressField{Value: "0x1111111111111111111111111111111111111111"},
				addresses: deployment.AddressesByChain{
					chainsel.ETHEREUM_MAINNET.Selector: {
						"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
					},
				},
				expected: "",
			},
			{
				name:  "MultipleChains",
				field: AddressField{Value: "0x1234567890123456789012345678901234567890"},
				addresses: deployment.AddressesByChain{
					chainsel.ETHEREUM_MAINNET.Selector: {
						"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 1.0.0"),
					},
					chainsel.POLYGON_MAINNET.Selector: {
						"0x1234567890123456789012345678901234567890": deployment.MustTypeAndVersionFromString("Token 2.0.0"),
					},
				},
				expected: "", // Will be checked separately since map iteration order is not guaranteed
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx := NewFieldContext(tt.addresses)
				result := tt.field.Annotation(ctx)

				if tt.name == "MultipleChains" {
					// For multiple chains, we expect either result since map iteration order is not guaranteed
					expected1 := "address of Token 1.0.0 from ethereum-mainnet"
					expected2 := "address of Token 2.0.0 from polygon-mainnet"
					assert.True(t, result == expected1 || result == expected2,
						"Expected either %q or %q, got %q", expected1, expected2, result)
				} else {
					assert.Equal(t, tt.expected, result)
				}
			})
		}
	})

	t.Run("Annotation_EmptyContext", func(t *testing.T) {
		t.Parallel()
		ctx := &FieldContext{Ctx: map[string]any{}}
		addrField := AddressField{Value: "0x1234567890123456789012345678901234567890"}
		result := addrField.Annotation(ctx)
		assert.Empty(t, result)
	})
}

func TestFieldValueIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    FieldValue
		expected string
	}{
		{
			name: "NestedStructures",
			field: StructField{
				Fields: []NamedField{
					{
						Name: "config",
						Value: StructField{
							Fields: []NamedField{
								{Name: "chainId", Value: ChainSelectorField{Value: chainsel.ETHEREUM_MAINNET.Selector}},
								{Name: "addresses", Value: ArrayField{
									Elements: []FieldValue{
										AddressField{Value: "0x1111111111111111111111111111111111111111"},
										AddressField{Value: "0x2222222222222222222222222222222222222222"},
									},
								}},
							},
						},
					},
					{
						Name:  "data",
						Value: BytesField{Value: []byte{0x01, 0x02, 0x03}},
					},
				},
			},
			expected: "StructField",
		},
		{
			name: "MixedFieldTypes",
			field: ArrayField{
				Elements: []FieldValue{
					SimpleField{Value: "string"},
					BytesField{Value: []byte{0x42}},
					ChainSelectorField{Value: chainsel.SOLANA_MAINNET.Selector},
					AddressField{Value: "0x1234567890123456789012345678901234567890"},
				},
			},
			expected: "ArrayField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.field.GetType())
		})
	}
}
