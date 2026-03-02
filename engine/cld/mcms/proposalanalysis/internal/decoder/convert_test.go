package decoder

import (
	"encoding/json"
	"testing"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

func TestAdaptTimelockProposal(t *testing.T) {
	t.Parallel()

	t.Run("empty proposal produces empty batches", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{}
		proposal := &mcms.TimelockProposal{}

		result := adaptTimelockProposal(report, proposal)

		require.NotNil(t, result)
		assert.Empty(t, result.BatchOperations())
	})

	t.Run("single batch with one decoded call", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{
					ChainSelector: 1111,
					ChainName:     "ethereum-mainnet",
					Operations: []experimentalanalyzer.OperationReport{
						{
							Calls: []*experimentalanalyzer.DecodedCall{
								{
									Address:         "0xABC",
									Method:          "function setConfig(uint256)",
									ContractType:    "Router",
									ContractVersion: "1.0.0",
									Inputs: []experimentalanalyzer.NamedField{
										{Name: "value", Value: experimentalanalyzer.SimpleField{Value: "42"}, RawValue: "42"},
									},
								},
							},
						},
					},
				},
			},
		}
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{
					ChainSelector: 1111,
					Transactions: []mcmstypes.Transaction{
						{
							To:               "0xABC",
							Data:             []byte{0x01, 0x02},
							AdditionalFields: json.RawMessage(`{"gas": 100}`),
							OperationMetadata: mcmstypes.OperationMetadata{
								ContractType: "Router",
							},
						},
					},
				},
			},
		}

		result := adaptTimelockProposal(report, proposal)

		require.Len(t, result.BatchOperations(), 1)

		batch := result.BatchOperations()[0]
		assert.Equal(t, uint64(1111), batch.ChainSelector())

		require.Len(t, batch.Calls(), 1)

		call := batch.Calls()[0]
		assert.Equal(t, "0xABC", call.To())
		assert.Equal(t, "setConfig", call.Name())
		assert.Equal(t, "Router", call.ContractType())
		assert.Equal(t, "1.0.0", call.ContractVersion())
		assert.Equal(t, []byte{0x01, 0x02}, call.Data())
		assert.JSONEq(t, `{"gas": 100}`, string(call.AdditionalFields()))

		require.Len(t, call.Inputs(), 1)
		assert.Equal(t, "value", call.Inputs()[0].Name())
		assert.Equal(t, "SimpleField", call.Inputs()[0].Type())
		assert.Equal(t, experimentalanalyzer.SimpleField{Value: "42"}, call.Inputs()[0].Value())
	})

	t.Run("undecoded transaction creates call with raw data only", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{
					ChainSelector: 2222,
					ChainName:     "polygon",
					Operations: []experimentalanalyzer.OperationReport{
						{Calls: nil},
					},
				},
			},
		}
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{
					ChainSelector: 2222,
					Transactions: []mcmstypes.Transaction{
						{
							To:               "0xDEF",
							Data:             []byte{0xAA, 0xBB},
							AdditionalFields: json.RawMessage(`{}`),
							OperationMetadata: mcmstypes.OperationMetadata{
								ContractType: "Unknown",
							},
						},
					},
				},
			},
		}

		result := adaptTimelockProposal(report, proposal)

		require.Len(t, result.BatchOperations(), 1)

		call := result.BatchOperations()[0].Calls()[0]
		assert.Equal(t, "0xDEF", call.To())
		assert.Equal(t, undecodedCallName, call.Name())
		assert.Nil(t, call.Inputs())
		assert.Nil(t, call.Outputs())
		assert.Equal(t, []byte{0xAA, 0xBB}, call.Data())
		assert.Equal(t, "Unknown", call.ContractType())
		assert.Empty(t, call.ContractVersion())
	})

	t.Run("multiple batches across chains", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{ChainSelector: 1, ChainName: "chain-a", Operations: []experimentalanalyzer.OperationReport{{Calls: nil}}},
				{ChainSelector: 2, ChainName: "chain-b", Operations: []experimentalanalyzer.OperationReport{{Calls: nil}}},
				{ChainSelector: 3, ChainName: "chain-c", Operations: []experimentalanalyzer.OperationReport{{Calls: nil}}},
			},
		}
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{ChainSelector: 1, Transactions: []mcmstypes.Transaction{{To: "0x1"}}},
				{ChainSelector: 2, Transactions: []mcmstypes.Transaction{{To: "0x2"}}},
				{ChainSelector: 3, Transactions: []mcmstypes.Transaction{{To: "0x3"}}},
			},
		}

		result := adaptTimelockProposal(report, proposal)

		require.Len(t, result.BatchOperations(), 3)
		assert.Equal(t, uint64(1), result.BatchOperations()[0].ChainSelector())
		assert.Equal(t, uint64(2), result.BatchOperations()[1].ChainSelector())
		assert.Equal(t, uint64(3), result.BatchOperations()[2].ChainSelector())
	})

	t.Run("call with inputs and outputs", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{
					ChainSelector: 1,
					Operations: []experimentalanalyzer.OperationReport{
						{
							Calls: []*experimentalanalyzer.DecodedCall{
								{
									Address: "0xABC",
									Method:  "getBalance",
									Inputs: []experimentalanalyzer.NamedField{
										{Name: "account", Value: experimentalanalyzer.AddressField{Value: "0x123"}, RawValue: "0x123"},
									},
									Outputs: []experimentalanalyzer.NamedField{
										{Name: "balance", Value: experimentalanalyzer.SimpleField{Value: "1000"}, RawValue: "1000"},
									},
								},
							},
						},
					},
				},
			},
		}
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{ChainSelector: 1, Transactions: []mcmstypes.Transaction{{To: "0xABC"}}},
			},
		}

		result := adaptTimelockProposal(report, proposal)

		call := result.BatchOperations()[0].Calls()[0]

		require.Len(t, call.Inputs(), 1)
		assert.Equal(t, "account", call.Inputs()[0].Name())
		assert.Equal(t, "AddressField", call.Inputs()[0].Type())
		assert.Equal(t, experimentalanalyzer.AddressField{Value: "0x123"}, call.Inputs()[0].Value())

		require.Len(t, call.Outputs(), 1)
		assert.Equal(t, "balance", call.Outputs()[0].Name())
		assert.Equal(t, "SimpleField", call.Outputs()[0].Type())
		assert.Equal(t, experimentalanalyzer.SimpleField{Value: "1000"}, call.Outputs()[0].Value())
	})
}

func TestCleanMethodName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full Solidity signature",
			input:    "function applyRampUpdates(uint256[])",
			expected: "applyRampUpdates",
		},
		{
			name:     "plain method name",
			input:    "setConfig",
			expected: "setConfig",
		},
		{
			name:     "function keyword with no parens",
			input:    "function transfer",
			expected: "transfer",
		},
		{
			name:     "method with complex params",
			input:    "function setOCR2Config(address[],bytes32[],uint8,bytes,uint64[],bytes)",
			expected: "setOCR2Config",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "with leading and trailing spaces",
			input:    "  function  doStuff(uint256)  ",
			expected: "doStuff",
		},
		{
			name:     "parens only",
			input:    "foo()",
			expected: "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, cleanMethodName(tt.input))
		})
	}
}

func TestAdaptNamedFields(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, adaptNamedFields(nil))
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, adaptNamedFields([]experimentalanalyzer.NamedField{}))
	})

	t.Run("field with nil value", func(t *testing.T) {
		t.Parallel()

		fields := []experimentalanalyzer.NamedField{
			{Name: "noValue", Value: nil},
		}

		result := adaptNamedFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "noValue", result[0].Name())
		assert.Empty(t, result[0].Type())
		assert.Nil(t, result[0].Value())
	})

	t.Run("field with Value stores the FieldValue", func(t *testing.T) {
		t.Parallel()

		fields := []experimentalanalyzer.NamedField{
			{Name: "selector", Value: experimentalanalyzer.ChainSelectorField{Value: 42}},
		}

		result := adaptNamedFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "selector", result[0].Name())
		assert.Equal(t, "ChainSelectorField", result[0].Type())
		assert.Equal(t, experimentalanalyzer.ChainSelectorField{Value: 42}, result[0].Value())
	})

	t.Run("multiple fields of different types", func(t *testing.T) {
		t.Parallel()

		fields := []experimentalanalyzer.NamedField{
			{Name: "addr", Value: experimentalanalyzer.AddressField{Value: "0xABC"}, RawValue: "0xABC"},
			{Name: "amount", Value: experimentalanalyzer.SimpleField{Value: "1000"}, RawValue: "1000"},
			{Name: "data", Value: experimentalanalyzer.BytesField{Value: []byte{0xFF}}, RawValue: []byte{0xFF}},
		}

		result := adaptNamedFields(fields)

		require.Len(t, result, 3)

		assert.Equal(t, "addr", result[0].Name())
		assert.Equal(t, "AddressField", result[0].Type())
		assert.Equal(t, experimentalanalyzer.AddressField{Value: "0xABC"}, result[0].Value())

		assert.Equal(t, "amount", result[1].Name())
		assert.Equal(t, "SimpleField", result[1].Type())
		assert.Equal(t, experimentalanalyzer.SimpleField{Value: "1000"}, result[1].Value())

		assert.Equal(t, "data", result[2].Name())
		assert.Equal(t, "BytesField", result[2].Type())
		assert.Equal(t, experimentalanalyzer.BytesField{Value: []byte{0xFF}}, result[2].Value())
	})
}
