package analyzer

import (
	"encoding/json"
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
					require.Equal(t, expectedInput.Value.Describe(nil), input.Value.Describe(nil), "Input value mismatch for call %d, input %d", i, j)
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
				require.Equal(t, expectedInput.Value.Describe(nil), input.Value.Describe(nil), "Input value mismatch for input %d", i)
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

// Tests for native token transfer functionality

func TestIsNativeTokenTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tx       types.Transaction
		expected bool
	}{
		{
			name: "Native transfer - empty data, non-zero value",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`), // 1 ETH in wei
			},
			expected: true,
		},
		{
			name: "Contract call - non-empty data, zero value",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Some method call
				AdditionalFields: json.RawMessage(`{"value": "0"}`),
			},
			expected: false,
		},
		{
			name: "Contract call with value - non-empty data, non-zero value",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{0xa9, 0x05, 0x9c, 0xbb}, // Some method call
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expected: false,
		},
		{
			name: "Empty transaction - empty data, zero value",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "0"}`),
			},
			expected: false,
		},
		{
			name: "Invalid AdditionalFields - empty data, non-zero value",
			tx: types.Transaction{
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
		tx            types.Transaction
		expectedValue string
	}{
		{
			name: "Valid value - 1 ETH",
			tx: types.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedValue: "1000000000000000000",
		},
		{
			name: "Valid value - 0.5 ETH",
			tx: types.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "500000000000000000"}`),
			},
			expectedValue: "500000000000000000",
		},
		{
			name: "Zero value",
			tx: types.Transaction{
				AdditionalFields: json.RawMessage(`{"value": "0"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Invalid JSON - should return 0",
			tx: types.Transaction{
				AdditionalFields: json.RawMessage(`{"invalid": "json"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Missing value field - should return 0",
			tx: types.Transaction{
				AdditionalFields: json.RawMessage(`{"other": "field"}`),
			},
			expectedValue: "0",
		},
		{
			name: "Large value - exceeds int64",
			tx: types.Transaction{
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
		tx           types.Transaction
		expectedCall *DecodedCall
	}{
		{
			name: "1 ETH transfer",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Method:  "native_transfer",
				Inputs: []NamedDescriptor{
					{
						Name:  "recipient",
						Value: AddressDescriptor{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleDescriptor{Value: "1000000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleDescriptor{Value: "1.000000000000000000"},
					},
				},
				Outputs: []NamedDescriptor{},
			},
		},
		{
			name: "0.5 ETH transfer",
			tx: types.Transaction{
				To:               "0x1234567890123456789012345678901234567890",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "500000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0x1234567890123456789012345678901234567890",
				Method:  "native_transfer",
				Inputs: []NamedDescriptor{
					{
						Name:  "recipient",
						Value: AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleDescriptor{Value: "500000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleDescriptor{Value: "0.500000000000000000"},
					},
				},
				Outputs: []NamedDescriptor{},
			},
		},
		{
			name: "Small amount transfer - 1 wei",
			tx: types.Transaction{
				To:               "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Method:  "native_transfer",
				Inputs: []NamedDescriptor{
					{
						Name:  "recipient",
						Value: AddressDescriptor{Value: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleDescriptor{Value: "1"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleDescriptor{Value: "0.000000000000000001"},
					},
				},
				Outputs: []NamedDescriptor{},
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
	ctx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		evmRegistry:      nil, // No registry - this would normally cause an error
	}

	decoder := NewTxCallDecoder(nil)

	tests := []struct {
		name          string
		tx            types.Transaction
		expectedCall  *DecodedCall
		expectedError bool
	}{
		{
			name: "Native transfer - should succeed without registry",
			tx: types.Transaction{
				To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Data:             []byte{},
				AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`),
			},
			expectedCall: &DecodedCall{
				Address: "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
				Method:  "native_transfer",
				Inputs: []NamedDescriptor{
					{
						Name:  "recipient",
						Value: AddressDescriptor{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
					},
					{
						Name:  "amount_wei",
						Value: SimpleDescriptor{Value: "1000000000000000000"},
					},
					{
						Name:  "amount_eth",
						Value: SimpleDescriptor{Value: "1.000000000000000000"},
					},
				},
				Outputs: []NamedDescriptor{},
			},
			expectedError: false,
		},
		{
			name: "Contract call - should fail without registry",
			tx: types.Transaction{
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
			result, abi, abiStr, err := AnalyzeEVMTransaction(ctx, decoder, chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, tt.tx)

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
	ctx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		evmRegistry:      nil,
	}

	tests := []struct {
		name          string
		txs           []types.Transaction
		expectedCalls []*DecodedCall
		expectedError bool
	}{
		{
			name: "Single native transfer",
			txs: []types.Transaction{
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
					Inputs: []NamedDescriptor{
						{
							Name:  "recipient",
							Value: AddressDescriptor{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleDescriptor{Value: "1000000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleDescriptor{Value: "1.000000000000000000"},
						},
					},
					Outputs: []NamedDescriptor{},
				},
			},
			expectedError: false,
		},
		{
			name: "Multiple native transfers",
			txs: []types.Transaction{
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
					Inputs: []NamedDescriptor{
						{
							Name:  "recipient",
							Value: AddressDescriptor{Value: "0xeE5E8f8Be22101d26084e90053695E2088a01a24"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleDescriptor{Value: "1000000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleDescriptor{Value: "1.000000000000000000"},
						},
					},
					Outputs: []NamedDescriptor{},
				},
				{
					Address: "0x1234567890123456789012345678901234567890",
					Method:  "native_transfer",
					Inputs: []NamedDescriptor{
						{
							Name:  "recipient",
							Value: AddressDescriptor{Value: "0x1234567890123456789012345678901234567890"},
						},
						{
							Name:  "amount_wei",
							Value: SimpleDescriptor{Value: "500000000000000000"},
						},
						{
							Name:  "amount_eth",
							Value: SimpleDescriptor{Value: "0.500000000000000000"},
						},
					},
					Outputs: []NamedDescriptor{},
				},
			},
			expectedError: false,
		},
		{
			name: "Mixed native transfer and contract call - should fail on contract call",
			txs: []types.Transaction{
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
			result, err := AnalyzeEVMTransactions(ctx, chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, tt.txs)

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
