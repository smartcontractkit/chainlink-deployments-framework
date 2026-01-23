package analyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	testenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
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
			addressesByChain: deployment.AddressesByChain{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
				},
			},
		},
	}

	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		txs     []mcmstypes.Transaction
		want    []*DecodedCall
		wantErr bool
	}{
		{
			name: "Unknown address - should fail",
			txs: []mcmstypes.Transaction{
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
			txs: []mcmstypes.Transaction{
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

			result, err := AnalyzeEVMTransactions(t.Context(), defaultProposalCtx, deployment.Environment{}, chainSelector, tt.txs)

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
			addressesByChain: deployment.AddressesByChain{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
				},
			},
		},
	}

	decoder := NewTxCallDecoder(nil)
	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name    string
		mcmsTx  mcmstypes.Transaction
		want    *DecodedCall
		wantErr bool
	}{
		{
			name: "No EVM registry",
			mcmsTx: mcmstypes.Transaction{
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

			proposalCtx := defaultProposalCtx
			if tt.name == "No EVM registry" {
				proposalCtx = &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
							evmTestAddress: deployment.MustTypeAndVersionFromString("TestContract 1.0.0"),
						},
					},
					evmRegistry: nil, // No registry
				}
			}

			result, abi, abiStr, err := AnalyzeEVMTransaction(t.Context(), proposalCtx, deployment.Environment{}, decoder, chainSelector, tt.mcmsTx)

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
	abis             map[string]string
	addressesByChain deployment.AddressesByChain
}

func (m *mockEVMRegistry) GetABIByAddress(chainSelector uint64, address string) (*abi.ABI, string, error) {
	addressesForChain, ok := m.addressesByChain[chainSelector]
	if !ok {
		return nil, "", fmt.Errorf("no addresses found for chain selector %d", chainSelector)
	}
	addressTypeAndVersion, ok := addressesForChain[address]
	if !ok {
		return nil, "", fmt.Errorf("address %s not found for chain selector %d", address, chainSelector)
	}

	return m.GetABIByType(addressTypeAndVersion)
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

// Tests for native token transfer functionality

func TestIsNativeTokenTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tx       mcmstypes.Transaction
		expected bool
	}{
		{
			name: "Native transfer - empty data, non-zero value",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": 1000000000000000000}`), // 1 ETH in wei
			},
			expected: true,
		},
		{
			name: "Contract call - non-empty data, zero value",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Some method call
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
			expected: false,
		},
		{
			name: "Contract call with value - non-empty data, non-zero value",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Some method call
				AdditionalFields: json.RawMessage(`{"value": 1000000000000000000}`),
			},
			expected: false,
		},
		{
			name: "Empty transaction - empty data, zero value",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
			expected: false,
		},
		{
			name: "Invalid AdditionalFields - empty data, non-zero value",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"invalid": "json"}`),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isNativeTokenTransfer(tt.tx)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTransactionValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tx            mcmstypes.Transaction
		expectedValue string
	}{
		{
			name: "Valid value - 1 ETH (string)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedValue: "1000000000000000000",
		},
		{
			name: "Valid value - 1 ETH (number)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": 1000000000000000000}`),
			},
			expectedValue: "1000000000000000000",
		},
		{
			name: "Valid value - 0.5 ETH (string)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "500000000000000000"}`),
			},
			expectedValue: "500000000000000000",
		},
		{
			name: "Valid value - 0.5 ETH (number)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": 500000000000000000}`),
			},
			expectedValue: "500000000000000000",
		},
		{
			name: "Zero value (string)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "0"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Zero value (number)",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
			expectedValue: "0",
		},
		{
			name: "Invalid JSON - should return 0",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"invalid": "json"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Missing value field - should return 0",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"other": "field"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Large value - exceeds int64",
			tx: mcmstypes.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "115792089237316195423570985008687907853269984665640564039457584007913129639935"}`), // 2^256-1
			},
			expectedValue: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getTransactionValue(tt.tx)
			require.Equal(t, tt.expectedValue, result.String())
		})
	}
}

func TestCreateNativeTransferCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tx           mcmstypes.Transaction
		expectedCall *DecodedCall
	}{
		{
			name: "1 ETH transfer",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Method:  "native_transfer",
				Inputs: []NamedField{
					{
						Name:  "recipient",
						Value: AddressField{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleField{Value: "1000000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleField{Value: "1.000000000000000000"},
					},
				},
				Outputs: []NamedField{},
			},
		},
		{
			name: "0.5 ETH transfer",
			tx: mcmstypes.Transaction{
				To:               "0x1234567890123456789012345678901234567890",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "500000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "native_transfer",
				Inputs: []NamedField{
					{
						Name:  "recipient",
						Value: AddressField{Value: "0x1234567890123456789012345678901234567890"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleField{Value: "500000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleField{Value: "0.500000000000000000"},
					},
				},
				Outputs: []NamedField{},
			},
		},
		{
			name: "Small amount transfer - 1 wei",
			tx: mcmstypes.Transaction{
				To:               "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Method:  "native_transfer",
				Inputs: []NamedField{
					{
						Name:  "recipient",
						Value: AddressField{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleField{Value: "1"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleField{Value: "0.000000000000000001"},
					},
				},
				Outputs: []NamedField{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := createNativeTransferCall(tt.tx)
			require.Equal(t, tt.expectedCall, result)
		})
	}
}

func TestAnalyzeEVMTransaction_NativeTransfer(t *testing.T) {
	t.Parallel()

	// Create a context without EVM registry to test that native transfers work without it
	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		evmRegistry:      nil, // No registry - this would normally cause an error
	}

	decoder := NewTxCallDecoder(nil)

	tests := []struct {
		name          string
		tx            mcmstypes.Transaction
		expectedCall  *DecodedCall
		expectedError bool
	}{
		{
			name: "Native transfer - should succeed without registry",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Method:  "native_transfer",
				Inputs: []NamedField{
					{
						Name:  "recipient",
						Value: AddressField{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleField{Value: "1000000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleField{Value: "1.000000000000000000"},
					},
				},
				Outputs: []NamedField{},
			},
			expectedError: false,
		},
		{
			name: "Contract call - should fail without registry",
			tx: mcmstypes.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Some method call
				AdditionalFields: json.RawMessage(`{"value": "0"}`),
			},
			expectedCall:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, abi, abiStr, err := AnalyzeEVMTransaction(t.Context(), proposalCtx, deployment.Environment{}, decoder, chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, tt.tx)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, result)
				require.Nil(t, abi)
				require.Empty(t, abiStr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCall, result)
				require.Nil(t, abi) // Native transfers don't have ABI
				require.Empty(t, abiStr)
			}
		})
	}
}

func TestAnalyzeEVMTransactions_NativeTransfer(t *testing.T) {
	t.Parallel()

	// Create a context without EVM registry
	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		evmRegistry:      nil,
	}

	tests := []struct {
		name          string
		txs           []mcmstypes.Transaction
		expectedCalls []*DecodedCall
		expectedError bool
	}{
		{
			name: "Single native transfer",
			txs: []mcmstypes.Transaction{
				{
					To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Data:             []byte{},
					AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
				},
			},
			expectedCalls: []*DecodedCall{
				{
					Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Method:  "native_transfer",
					Inputs: []NamedField{
						{
							Name:  "recipient",
							Value: AddressField{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleField{Value: "1000000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleField{Value: "1.000000000000000000"},
						},
					},
					Outputs: []NamedField{},
				},
			},
			expectedError: false,
		},
		{
			name: "Multiple native transfers",
			txs: []mcmstypes.Transaction{
				{
					To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Data:             []byte{},
					AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
				},
				{
					To:               "0x1234567890123456789012345678901234567890",
					Data:             []byte{},
					AdditionalFields: json.RawMessage(`{"value": "500000000000000000"}`),
				},
			},
			expectedCalls: []*DecodedCall{
				{
					Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Method:  "native_transfer",
					Inputs: []NamedField{
						{
							Name:  "recipient",
							Value: AddressField{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleField{Value: "1000000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleField{Value: "1.000000000000000000"},
						},
					},
					Outputs: []NamedField{},
				},
				{
					Address: "0x1234567890123456789012345678901234567890",
					Method:  "native_transfer",
					Inputs: []NamedField{
						{
							Name:  "recipient",
							Value: AddressField{Value: "0x1234567890123456789012345678901234567890"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleField{Value: "500000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleField{Value: "0.500000000000000000"},
						},
					},
					Outputs: []NamedField{},
				},
			},
			expectedError: false,
		},
		{
			name: "Mixed native transfer and contract call - should fail on contract call",
			txs: []mcmstypes.Transaction{
				{
					To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Data:             []byte{},
					AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
				},
				{
					To:               "0x1234567890123456789012345678901234567890",
					Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Contract call
					AdditionalFields: json.RawMessage(`{"value": "0"}`),
				},
			},
			expectedCalls: nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := AnalyzeEVMTransactions(t.Context(), proposalCtx, deployment.Environment{}, chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, tt.txs)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCalls, result)
			}
		})
	}
}

func TestIsMethodNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "no method with id error",
			err:      errors.New("no method with id: 0x8861cc77"),
			expected: true,
		},
		{
			name:     "method not found error",
			err:      errors.New("method not found"),
			expected: true,
		},
		{
			name:     "invalid method id error",
			err:      errors.New("invalid method id"),
			expected: true,
		},
		{
			name:     "mixed case - no method with id",
			err:      errors.New("No Method With Id: 0x1234"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "unpacking error",
			err:      errors.New("abi: cannot unpack"),
			expected: false,
		},
		{
			name:     "decode error",
			err:      errors.New("failed to decode"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isMethodNotFoundError(tt.err)
			require.Equal(t, tt.expected, result, "isMethodNotFoundError(%v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}

func TestQueryEIP1967ImplementationSlot(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	proxyAddress := "0x1234567890123456789012345678901234567890"

	tests := []struct {
		name          string
		setupChain    func(t *testing.T) evm.Chain
		proxyAddress  string
		expectedAddr  common.Address
		expectedError bool
		errorContains string
	}{
		{
			name: "zero address - not a proxy (empty storage)",
			setupChain: func(t *testing.T) evm.Chain {
				t.Helper()
				env, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)
				evmChains := env.Chains().EVMChains()

				return evmChains[chainSelector]
			},
			proxyAddress:  proxyAddress,
			expectedAddr:  common.Address{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evmChain := tt.setupChain(t)
			result, err := queryEIP1967ImplementationSlot(context.Background(), evmChain, tt.proxyAddress)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedAddr, result)
		})
	}
}

func TestTryEIP1967ProxyFallback(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	proxyAddress := "0x1234567890123456789012345678901234567890"
	implAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")

	proxyABI := `[{"type":"function","name":"upgradeTo","stateMutability":"nonpayable","inputs":[{"type":"address","name":"newImplementation"}],"outputs":[]}]`

	// Method ID for transfer(address,uint256) = 0xa9059cbb
	transferMethodID := []byte{0xa9, 0x05, 0x9c, 0xbb}
	txData := append(transferMethodID, make([]byte, 64)...) // Method ID + 64 bytes of zeros for params

	tests := []struct {
		name          string
		setupCtx      func(t *testing.T) (ProposalContext, deployment.Environment)
		chainSelector uint64
		proxyAddress  string
		txData        []byte
		expectedError bool
		errorContains string
		verifyResult  func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string)
	}{
		{
			name: "chain not available",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				return &DefaultProposalContext{
						AddressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							},
						},
						evmRegistry: &mockEVMRegistry{
							abis: map[string]string{
								"TransparentUpgradeableProxy 1.0.0": proxyABI,
							},
							addressesByChain: deployment.AddressesByChain{
								chainSelector: {
									proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								},
							},
						},
					}, deployment.Environment{
						BlockChains: chain.NewBlockChainsFromSlice([]chain.BlockChain{}), // empty blockchains
					}
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: true,
			errorContains: "EVM chain not available",
		},
		{
			name: "not an EIP-1967 proxy - zero implementation address",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Don't set storage - it will be zero, simulating a non-proxy contract
				// The proxyAddress contract doesn't have EIP-1967 storage set

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: true,
			errorContains: "EIP-1967 slot contains zero address",
		},
		{
			name: "implementation not found in address book",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress (via mock wrapper), but don't add it to address book
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				// Add proxy address to ExistingAddresses so getAllAddressesByChain can find it
				err = testEnv.ExistingAddresses.Save(chainSelector, proxyAddress, deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0")) // nolint
				require.NoError(t, err)
				// Note: Implementation address NOT added to address book (this is the test case)

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							// Implementation address NOT in address book
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: true,
			errorContains: "not found in address book",
		},
		{
			name: "implementation ABI not found",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				// Add addresses to ExistingAddresses so getAllAddressesByChain can find them
				err = testEnv.ExistingAddresses.Save(chainSelector, proxyAddress, deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0")) // nolint
				require.NoError(t, err)
				err = testEnv.ExistingAddresses.Save(chainSelector, implAddress.Hex(), deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0")) // nolint
				require.NoError(t, err)

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress:      deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							implAddress.Hex(): deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
							// ImplementationContract ABI not registered
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress:      deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								implAddress.Hex(): deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: true,
			errorContains: "failed to get ABI for implementation",
		},
		{
			name: "empty environment - no EVM chains",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				return mockProposalContext(t), deployment.Environment{
					BlockChains: chain.NewBlockChainsFromSlice([]chain.BlockChain{}), // empty blockchains
				}
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: true,
			errorContains: "EVM chain not available",
		},
		{
			name: "success - EIP-1967 proxy fallback succeeds (from ExistingAddresses)",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				// Implementation ABI with transfer function that matches the txData
				implABI := `[{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[{"type":"bool"}]}]`

				// Add implementation address to ExistingAddresses
				err = testEnv.ExistingAddresses.Save(chainSelector, implAddress.Hex(), deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0")) // nolint
				require.NoError(t, err)

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
							"ImplementationContract 1.0.0":      implABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress:      deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								implAddress.Hex(): deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: false,
			verifyResult: func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string) {
				t.Helper()

				require.NotNil(t, result)
				require.NotNil(t, abiObj)
				require.NotEmpty(t, abiStr)
				require.Equal(t, proxyAddress, result.Address)
				require.Contains(t, result.Method, "transfer")
				require.Contains(t, abiStr, "transfer")
			},
		},
		{
			name: "success - EIP-1967 proxy fallback succeeds (implementation from DataStore)",
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				// Implementation ABI with transfer function that matches the txData
				implABI := `[{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[{"type":"bool"}]}]`

				// Add implementation address to DataStore (not ExistingAddresses)
				ds := datastore.NewMemoryDataStore()
				version := semver.MustParse("1.0.0")
				err = ds.Addresses().Add(datastore.AddressRef{
					ChainSelector: chainSelector,
					Address:       implAddress.Hex(),
					Type:          datastore.ContractType("ImplementationContract"),
					Version:       version,
					Qualifier:     "",
				})
				require.NoError(t, err)

				// Update testEnv to use the DataStore with implementation address
				testEnv.DataStore = ds.Seal()

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							// Implementation address NOT in ExistingAddresses, only in DataStore
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
							"ImplementationContract 1.0.0":      implABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								// Implementation address only in DataStore, not in registry's addressesByChain
							},
						},
					},
				}, *testEnv
			},
			chainSelector: chainSelector,
			proxyAddress:  proxyAddress,
			txData:        txData,
			expectedError: false,
			verifyResult: func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string) {
				t.Helper()

				require.NotNil(t, result)
				require.NotNil(t, abiObj)
				require.NotEmpty(t, abiStr)
				require.Equal(t, proxyAddress, result.Address)
				require.Contains(t, result.Method, "transfer")
				require.Contains(t, abiStr, "transfer")
				// Verify it successfully found the implementation from DataStore
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposalContext, env := tt.setupCtx(t)
			decoder := NewTxCallDecoder(nil)

			result, abiObj, abiStr, err := tryEIP1967ProxyFallback(
				t.Context(),
				proposalContext,
				env,
				tt.chainSelector,
				tt.proxyAddress,
				tt.txData,
				decoder,
			)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				require.Nil(t, result)
				require.Nil(t, abiObj)
				require.Empty(t, abiStr)

				return
			}

			require.NoError(t, err)
			if tt.verifyResult != nil {
				tt.verifyResult(t, result, abiObj, abiStr)
			}
		})
	}
}

// mockStorageClient wraps an OnchainClient and intercepts StorageAt calls
// to return a mock value for the EIP-1967 storage slot.
// This is a special wrapper that delegates all calls to the real client except StorageAt.
type mockStorageClient struct {
	evm.OnchainClient
	proxyAddr   common.Address
	implAddr    common.Address
	storageSlot common.Hash
}

// StorageAt intercepts calls to StorageAt and returns the mock implementation address
// if the call is for the EIP-1967 storage slot of the proxy address.
func (m *mockStorageClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	// If this is the EIP-1967 slot for our proxy, return the mock value
	if account == m.proxyAddr && key == m.storageSlot {
		// Return the address as 32 bytes (right-padded)
		result := make([]byte, 32)
		copy(result[12:], m.implAddr.Bytes())

		return result, nil
	}
	// Otherwise, delegate to the underlying client
	return m.OnchainClient.StorageAt(ctx, account, key, blockNumber)
}

// setEIP1967StorageOnSimulatedChain wraps the chain client to mock StorageAt calls
// for the EIP-1967 storage slot. This is a test helper that allows us to simulate
// EIP-1967 proxy behavior without actually setting storage on the simulated chain.
// It returns the modified chain that should be used in tests.
func setEIP1967StorageOnSimulatedChain(t *testing.T, evmChain evm.Chain, proxyAddr, implAddr common.Address) (evm.Chain, error) {
	t.Helper()

	// Wrap the client to intercept StorageAt calls
	mockClient := &mockStorageClient{
		OnchainClient: evmChain.Client,
		proxyAddr:     proxyAddr,
		implAddr:      implAddr,
		storageSlot:   EIP1967TargetContractStorageSlot,
	}

	// Create a new chain with the mocked client
	mockChain := evmChain
	mockChain.Client = mockClient

	// Verify the mock works
	ctx := context.Background()
	storageValueRead, err := mockClient.StorageAt(ctx, proxyAddr, EIP1967TargetContractStorageSlot, nil)
	if err != nil {
		return evmChain, fmt.Errorf("failed to read storage slot via mock: %w", err)
	}

	readAddr := common.BytesToAddress(storageValueRead)
	if readAddr != implAddr {
		return evmChain, fmt.Errorf("mock storage verification failed: expected %s, got %s", implAddr.Hex(), readAddr.Hex())
	}

	return mockChain, nil
}

// mockProposalContext creates a MockProposalContext with all methods returning nil/empty values.
func mockProposalContext(t *testing.T) *MockProposalContext {
	t.Helper()
	mock := NewMockProposalContext(t)
	mock.On("GetEVMRegistry").Return(nil).Maybe()
	mock.On("GetSolanaDecoderRegistry").Return(nil).Maybe()
	mock.On("FieldsContext", testifymock.Anything).Return(nil).Maybe()
	mock.On("GetRenderer").Return(nil).Maybe()
	mock.On("SetRenderer", testifymock.Anything).Return().Maybe()

	return mock
}

// TestAnalyzeEVMTransaction_EIP1967ProxyFallback tests the EIP-1967 proxy fallback mechanism
// through the full AnalyzeEVMTransaction flow.
func TestAnalyzeEVMTransaction_EIP1967ProxyFallback(t *testing.T) {
	t.Parallel()

	chainSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	proxyAddress := "0x1234567890123456789012345678901234567890"
	implAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	implAddressStr := implAddress.Hex()

	// Proxy ABI doesn't have the method, implementation ABI does
	proxyABI := `[{"type":"function","name":"upgradeTo","stateMutability":"nonpayable","inputs":[{"type":"address","name":"newImplementation"}],"outputs":[]}]`
	implABI := `[{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[]}]`

	// Method ID for transfer(address,uint256) = 0xa9059cbb
	transferMethodID := []byte{0xa9, 0x05, 0x9c, 0xbb}
	txData := append(transferMethodID, make([]byte, 64)...) // Method ID + 64 bytes of zeros for params

	tests := []struct {
		name          string
		mcmsTx        mcmstypes.Transaction
		setupCtx      func(t *testing.T) (ProposalContext, deployment.Environment)
		expectedError bool
		errorContains string
		verifyResult  func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string)
	}{
		{
			name: "success path: proxy ABI fails, EIP-1967 fallback to implementation ABI succeeds",
			mcmsTx: mcmstypes.Transaction{
				To:   proxyAddress,
				Data: txData,
			},
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				// Add addresses to ExistingAddresses so getAllAddressesByChain can find them
				err = testEnv.ExistingAddresses.Save(chainSelector, proxyAddress, deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0")) // nolint
				require.NoError(t, err)
				err = testEnv.ExistingAddresses.Save(chainSelector, implAddressStr, deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0")) // nolint
				require.NoError(t, err)

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress:   deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							implAddressStr: deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
							"ImplementationContract 1.0.0":      implABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress:   deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								implAddressStr: deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			expectedError: false,
			verifyResult: func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string) {
				t.Helper()

				require.NotNil(t, result)
				require.NotNil(t, abiObj)
				require.Contains(t, abiStr, "transfer")
				require.Equal(t, proxyAddress, result.Address)
			},
		},
		{
			name: "non-method-not-found error: no fallback triggered",
			mcmsTx: mcmstypes.Transaction{
				To:   proxyAddress,
				Data: []byte{0x12, 0x34}, // Too short, will fail before method lookup
			},
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							},
						},
					},
				}, deployment.Environment{}
			},
			expectedError: true,
			errorContains: "error analyzing operation",
			verifyResult: func(t *testing.T, result *DecodedCall, abiObj *abi.ABI, abiStr string) {
				t.Helper()

				require.Nil(t, result)
				require.Nil(t, abiObj)
				require.Empty(t, abiStr)
			},
		},
		{
			name: "both proxy and implementation ABIs fail after fallback",
			mcmsTx: mcmstypes.Transaction{
				To:   proxyAddress,
				Data: []byte{0x88, 0x61, 0xcc, 0x77, 0x00, 0x00}, // Method ID that doesn't exist in either ABI
			},
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				testEnv, err := testenv.New(t.Context(), testenv.WithEVMSimulated(t, []uint64{chainSelector}))
				require.NoError(t, err)

				// Set storage to return implAddress
				evmChains := testEnv.Chains().EVMChains()
				evmChain := evmChains[chainSelector]
				mockChain, err := setEIP1967StorageOnSimulatedChain(t, evmChain, common.HexToAddress(proxyAddress), implAddress)
				require.NoError(t, err)
				evmChains[chainSelector] = mockChain
				// Rebuild BlockChains with all chains (including the mocked one)
				allChains := make([]chain.BlockChain, 0)
				for _, c := range testEnv.Chains().All() {
					if c.ChainSelector() == chainSelector {
						allChains = append(allChains, mockChain)
					} else {
						allChains = append(allChains, c)
					}
				}
				testEnv.BlockChains = chain.NewBlockChainsFromSlice(allChains)

				return &DefaultProposalContext{
					AddressesByChain: deployment.AddressesByChain{
						chainSelector: {
							proxyAddress:   deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							implAddressStr: deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
						},
					},
					evmRegistry: &mockEVMRegistry{
						abis: map[string]string{
							"TransparentUpgradeableProxy 1.0.0": proxyABI,
							"ImplementationContract 1.0.0":      implABI,
						},
						addressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress:   deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								implAddressStr: deployment.MustTypeAndVersionFromString("ImplementationContract 1.0.0"),
							},
						},
					},
				}, *testEnv
			},
			expectedError: true,
			errorContains: "error analyzing operation",
		},
		{
			name: "no chain available: returns original method-not-found error (not chain error)",
			mcmsTx: mcmstypes.Transaction{
				To:   proxyAddress,
				Data: txData,
			},
			setupCtx: func(t *testing.T) (ProposalContext, deployment.Environment) {
				t.Helper()

				return &DefaultProposalContext{
						AddressesByChain: deployment.AddressesByChain{
							chainSelector: {
								proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
							},
						},
						evmRegistry: &mockEVMRegistry{
							abis: map[string]string{
								"TransparentUpgradeableProxy 1.0.0": proxyABI,
							},
							addressesByChain: deployment.AddressesByChain{
								chainSelector: {
									proxyAddress: deployment.MustTypeAndVersionFromString("TransparentUpgradeableProxy 1.0.0"),
								},
							},
						},
					}, deployment.Environment{
						BlockChains: chain.NewBlockChainsFromSlice([]chain.BlockChain{}), // empty blockchains
					}
			},
			expectedError: true,
			errorContains: "error analyzing operation", // Original error, not chain error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposalContext, env := tt.setupCtx(t)
			decoder := NewTxCallDecoder(nil)
			// For tests that use testEnv, we'll need to extract it from the test setup context
			// For now, tests that need chains should pass env explicitly

			result, abiObj, abiStr, err := AnalyzeEVMTransaction(t.Context(), proposalContext, env, decoder, chainSelector, tt.mcmsTx)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				if tt.verifyResult != nil {
					tt.verifyResult(t, result, abiObj, abiStr)
				}

				return
			}

			require.NoError(t, err)
			if tt.verifyResult != nil {
				tt.verifyResult(t, result, abiObj, abiStr)
			}
		})
	}
}
