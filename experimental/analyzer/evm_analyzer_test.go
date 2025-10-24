package analyzer

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestAnalyzeEVMTransactions(t *testing.T) {
	t.Parallel()

	// Test addresses
	evmTestAddress := "0x1234567890123456789012345678901234567890"
	unknownAddress := "0x9999999999999999999999999999999999999999"

	testABI := `[{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[]}]`

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
				evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
			},
		},
		evmRegistry: &mockEVMRegistry{
			abis: map[string]string{
				"TestContract 1.0.0": testABI,
			},
		},
	}

	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		txs     []types.Transaction
		want    []*DecodedCall
		wantErr bool
	}{
		{
			name: "Unknown address - should fail",
			txs: []types.Transaction{
				{
					To:   unknownAddress,
					Data: []byte{0x29, 0x99, 0x89, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid transaction data - too short",
			txs: []types.Transaction{
				{
					To:   evmTestAddress,
					Data: []byte{0x29, 0x99}, // Too short, should fail
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeEVMTransactions(defaultProposalCtx, chainSelector, tt.txs)

			if tt.wantErr {
				require.Error(t, err, "AnalyzeEVMTransactions() should have failed")
				return
			}

			require.NoError(t, err, "AnalyzeEVMTransactions() should not have failed")
			require.Len(t, result, len(tt.want), "Number of decoded calls should match")

			// Compare each DecodedCall
			for i, decodedCall := range result {
				expected := tt.want[i]
				require.Equal(t, expected.Address, decodedCall.Address, "Address mismatch for call %d", i)
				require.Equal(t, expected.Method, decodedCall.Method, "Method mismatch for call %d", i)
				require.Len(t, decodedCall.Inputs, len(expected.Inputs), "Number of inputs should match for call %d", i)

				// Compare each input
				for j, input := range decodedCall.Inputs {
					expectedInput := expected.Inputs[j]
					require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for call %d, input %d", i, j)
					require.Equal(t, expectedInput.Value.GetType(), input.Value.GetType(), "Input value type mismatch for call %d, input %d", i, j)

					// Compare field values based on type
					switch expectedField := expectedInput.Value.(type) {
					case SimpleField:
						if actualField, ok := input.Value.(SimpleField); ok {
							require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "SimpleField value mismatch for call %d, input %d", i, j)
						} else {
							t.Errorf("Expected SimpleField but got %T for call %d, input %d", input.Value, i, j)
						}
					case AddressField:
						if actualField, ok := input.Value.(AddressField); ok {
							require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "AddressField value mismatch for call %d, input %d", i, j)
						} else {
							t.Errorf("Expected AddressField but got %T for call %d, input %d", input.Value, i, j)
						}
					default:
						// For other field types, we can add more specific comparisons as needed
						require.Equal(t, expectedInput.Value, input.Value, "Field value mismatch for call %d, input %d", i, j)
					}
				}
			}
		})
	}
}

// TestAnalyzeEVMTransaction tests the individual transaction analysis function
func TestAnalyzeEVMTransaction(t *testing.T) {
	t.Parallel()

	evmTestAddress := "0x1234567890123456789012345678901234567890"
	testABI := `[{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[]}]`

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
				evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
			},
		},
		evmRegistry: &mockEVMRegistry{
			abis: map[string]string{
				"TestContract 1.0.0": testABI,
			},
		},
	}

	decoder := NewTxCallDecoder(nil)
	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		mcmsTx  types.Transaction
		want    *DecodedCall
		wantErr bool
	}{
		{
			name: "No EVM registry",
			mcmsTx: types.Transaction{
				To:   evmTestAddress,
				Data: []byte{0x29, 0x99, 0x89, 0x89, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := defaultProposalCtx
			if tt.name == "No EVM registry" {
				ctx = &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
							evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
						},
					},
					evmRegistry: nil, // No registry
				}
			}

			result, abi, abiStr, err := AnalyzeEVMTransaction(ctx, decoder, chainSelector, tt.mcmsTx)

			if tt.wantErr {
				require.Error(t, err, "AnalyzeEVMTransaction() should have failed")
				return
			}

			require.NoError(t, err, "AnalyzeEVMTransaction() should not have failed")
			require.NotNil(t, result, "Result should not be nil")
			require.NotNil(t, abi, "ABI should not be nil")
			require.NotEmpty(t, abiStr, "ABI string should not be empty")

			require.Equal(t, tt.want.Address, result.Address, "Address mismatch")
			require.Equal(t, tt.want.Method, result.Method, "Method mismatch")
			require.Len(t, result.Inputs, len(tt.want.Inputs), "Number of inputs should match")

			// Compare each input
			for i, input := range result.Inputs {
				expectedInput := tt.want.Inputs[i]
				require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for input %d", i)
				require.Equal(t, expectedInput.Value.GetType(), input.Value.GetType(), "Input value type mismatch for input %d", i)

				// Compare field values based on type
				switch expectedField := expectedInput.Value.(type) {
				case SimpleField:
					if actualField, ok := input.Value.(SimpleField); ok {
						require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "SimpleField value mismatch for input %d", i)
					} else {
						t.Errorf("Expected SimpleField but got %T for input %d", input.Value, i)
					}
				case AddressField:
					if actualField, ok := input.Value.(AddressField); ok {
						require.Equal(t, expectedField.GetValue(), actualField.GetValue(), "AddressField value mismatch for input %d", i)
					} else {
						t.Errorf("Expected AddressField but got %T for input %d", input.Value, i)
					}
				default:
					// For other field types, we can add more specific comparisons as needed
					require.Equal(t, expectedInput.Value, input.Value, "Field value mismatch for input %d", i)
				}
			}
		})
	}
}

// mockEVMRegistry is a simple mock implementation of EVMABIRegistry for testing
type mockEVMRegistry struct {
	abis map[string]string
}

func (m *mockEVMRegistry) GetABIByAddress(chainSelector uint64, address string) (*abi.ABI, string, error) {
	// Find the contract type for this address
	// This is a simplified mock - in a real test you'd need to look up the address in the datastore
	// For now, we'll assume the address maps to "TestContract 1.0.0"
	contractType := "TestContract 1.0.0"

	abiStr, exists := m.abis[contractType]
	if !exists {
		return nil, "", errors.New("ABI not found for contract type")
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, "", err
	}

	return &parsedABI, abiStr, nil
}

func (m *mockEVMRegistry) GetAllABIs() map[string]string {
	return m.abis
}

func (m *mockEVMRegistry) GetABIByType(typeAndVersion deployment.TypeAndVersion) (*abi.ABI, string, error) {
	abiStr, exists := m.abis[typeAndVersion.String()]
	if !exists {
		return nil, "", errors.New("ABI not found for contract type")
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, "", err
	}

	return &parsedABI, abiStr, nil
}

func (m *mockEVMRegistry) AddABI(contractType deployment.TypeAndVersion, abi string) error {
	if m.abis == nil {
		m.abis = make(map[string]string)
	}
	m.abis[contractType.String()] = abi

	return nil
}
