package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestDecodedCall_Describe(t *testing.T) {
	t.Parallel()
	addressesByChain := map[uint64]map[string]deployment.TypeAndVersion{}
	tests := []struct {
		name    string
		call    *DecodedCall
		context *DescriptorContext
		want    string
	}{
		{
			name: "Complete call with inputs and outputs",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "transfer",
				Inputs: []NamedDescriptor{
					{Name: "to", Value: SimpleDescriptor{Value: "0x0000000000000000000000000000000000000001"}},
					{Name: "amount", Value: SimpleDescriptor{Value: "100"}},
				},
				Outputs: []NamedDescriptor{
					{Name: "success", Value: SimpleDescriptor{Value: "true"}},
				},
			},
			context: NewDescriptorContext(addressesByChain),
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
				Inputs: []NamedDescriptor{
					{Name: "value", Value: SimpleDescriptor{Value: "42"}},
				},
				Outputs: []NamedDescriptor{},
			},
			context: NewDescriptorContext(addressesByChain),
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
				Inputs:  []NamedDescriptor{},
				Outputs: []NamedDescriptor{
					{Name: "value", Value: SimpleDescriptor{Value: "42"}},
				},
			},
			context: NewDescriptorContext(addressesByChain),
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
				Inputs:  []NamedDescriptor{},
				Outputs: []NamedDescriptor{},
			},
			context: NewDescriptorContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: fallback
`,
		},
		{
			name: "Call with complex descriptors",
			call: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "complexCall",
				Inputs: []NamedDescriptor{
					{Name: "address", Value: AddressDescriptor{Value: "0x0000000000000000000000000000000000000001"}},
					{Name: "chain", Value: ChainSelectorDescriptor{Value: 1}},
					{Name: "data", Value: BytesDescriptor{Value: []byte{0x01, 0x02, 0x03}}},
				},
				Outputs: []NamedDescriptor{
					{Name: "result", Value: ArrayDescriptor{Elements: []Descriptor{
						SimpleDescriptor{Value: "item1"},
						SimpleDescriptor{Value: "item2"},
					}}},
				},
			},
			context: NewDescriptorContext(addressesByChain),
			want: `Address: 0x1234567890123456789012345678901234567890
Method: complexCall
Inputs:
  address: 0x0000000000000000000000000000000000000001
  chain: 1 (<chain unknown>)
  data: 0x010203
Outputs:
  result: [item1,item2]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.call.Describe(tt.context)
			require.Equal(t, tt.want, result)
		})
	}
}
