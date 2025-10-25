package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestDecodedCall_String(t *testing.T) {
	t.Parallel()
	addressesByChain := map[uint64]map[string]deployment.TypeAndVersion{}
	tests := []struct {
		name    string
		call    *DecodedCall
		context *FieldContext
		want    string
	}{
		{
			name: "Complete call with inputs and outputs",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "transfer",
				Inputs: []NamedField{
					{Name: "to", Value: SimpleField{Value: "0x0000000000000000000000000000000000000001"}},
					{Name: "amount", Value: SimpleField{Value: "100"}},
				},
				Outputs: []NamedField{
					{Name: "success", Value: SimpleField{Value: "true"}},
				},
			},
			context: NewFieldContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: transfer

Inputs:
  to: 0x0000000000000000000000000000000000000001
  amount: 100

Outputs:
  success: true
`,
		},
		{
			name: "Call with only inputs",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "setValue",
				Inputs: []NamedField{
					{Name: "value", Value: SimpleField{Value: "42"}},
				},
				Outputs: []NamedField{},
			},
			context: NewFieldContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: setValue

Inputs:
  value: 42
`,
		},
		{
			name: "Call with only outputs",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "getValue",
				Inputs:  []NamedField{},
				Outputs: []NamedField{
					{Name: "value", Value: SimpleField{Value: "42"}},
				},
			},
			context: NewFieldContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: getValue
Outputs:
  value: 42
`,
		},
		{
			name: "Empty call",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "fallback",
				Inputs:  []NamedField{},
				Outputs: []NamedField{},
			},
			context: NewFieldContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: fallback
`,
		},
		{
			name: "Call with complex descriptors",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "complexCall",
				Inputs: []NamedField{
					{Name: "address", Value: AddressField{Value: "0x0000000000000000000000000000000000000001"}},
					{Name: "chain", Value: ChainSelectorField{Value: 1}},
					{Name: "data", Value: BytesField{Value: []byte{0x01, 0x02, 0x03}}},
				},
				Outputs: []NamedField{
					{Name: "result", Value: ArrayField{Elements: []FieldValue{
						SimpleField{Value: "item1"},
						SimpleField{Value: "item2"},
					}}},
				},
			},
			context: NewFieldContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: complexCall

Inputs:
  address: 0x0000000000000000000000000000000000000001
  chain: ` + "`" + `1 (<chain unknown>)\n` + "`" + `
  data: 0x010203

Outputs:
  result: item1, item2
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.call.String(tt.context)
			require.Equal(t, tt.want, result)
		})
	}
}
