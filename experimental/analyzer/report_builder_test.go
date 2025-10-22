package analyzer

import (
	"encoding/json"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestBuildProposalReport_EmptyProposal(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.Proposal{
		Operations: []types.Operation{},
	}

	report, err := BuildProposalReport(ctx, proposal)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Empty(t, report.Operations)
	require.Empty(t, report.Batches)
}

func TestBuildProposalReport_SingleOperation(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.Proposal{
		Operations: []types.Operation{
			{
				ChainSelector: 1, // Ethereum Mainnet
				Transaction: types.Transaction{
					To:   "0x1234567890123456789012345678901234567890",
					Data: []byte{0x01, 0x02, 0x03, 0x04}, // Minimal data for method ID
				},
			},
		},
	}

	// This should return an error because chain selector 1 is not recognized in test environment
	_, err := BuildProposalReport(ctx, proposal)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown chain selector 1")
}

func TestBuildProposalReport_MultipleOperations(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.Proposal{
		Operations: []types.Operation{
			{
				ChainSelector: 1, // Ethereum Mainnet
				Transaction: types.Transaction{
					To:   "0x1111111111111111111111111111111111111111",
					Data: []byte{0x01, 0x02, 0x03, 0x04},
				},
			},
			{
				ChainSelector: 1, // Ethereum Mainnet (same chain for simplicity)
				Transaction: types.Transaction{
					To:   "0x2222222222222222222222222222222222222222",
					Data: []byte{0x05, 0x06, 0x07, 0x08},
				},
			},
		},
	}

	// This should return an error because chain selector 1 is not recognized in test environment
	_, err := BuildProposalReport(ctx, proposal)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown chain selector 1")
}

func TestBuildTimelockReport_EmptyProposal(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.TimelockProposal{
		Operations: []types.BatchOperation{},
	}

	report, err := BuildTimelockReport(ctx, proposal)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Empty(t, report.Operations)
	require.Empty(t, report.Batches)
}

func TestBuildTimelockReport_SingleBatch(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.TimelockProposal{
		Operations: []types.BatchOperation{
			{
				ChainSelector: 1, // Ethereum Mainnet
				Transactions: []types.Transaction{
					{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
	}

	// This should return an error because chain selector 1 is not recognized in test environment
	_, err := BuildTimelockReport(ctx, proposal)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown chain selector 1")
}

func TestBuildTimelockReport_MultipleBatches(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.TimelockProposal{
		Operations: []types.BatchOperation{
			{
				ChainSelector: 1, // Ethereum Mainnet
				Transactions: []types.Transaction{
					{
						To:   "0x1111111111111111111111111111111111111111",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			{
				ChainSelector: 1, // Ethereum Mainnet (same chain for simplicity)
				Transactions: []types.Transaction{
					{
						To:   "0x2222222222222222222222222222222222222222",
						Data: []byte{0x05, 0x06, 0x07, 0x08},
					},
				},
			},
		},
	}

	// This should return an error because chain selector 1 is not recognized in test environment
	_, err := BuildTimelockReport(ctx, proposal)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown chain selector 1")
}

func TestBuildProposalReport_UnsupportedChainFamily(t *testing.T) {
	t.Parallel()
	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
	proposal := &mcms.Proposal{
		Operations: []types.Operation{
			{
				ChainSelector: 999999, // Unsupported chain
				Transaction: types.Transaction{
					To:   "0x1234567890123456789012345678901234567890",
					Data: []byte{0x01, 0x02, 0x03, 0x04},
				},
			},
		},
	}

	// This should return an error because the chain selector is unknown
	_, err := BuildProposalReport(ctx, proposal)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown chain selector 999999")
}

func TestChainNameOrUnknown(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Ethereum Mainnet", chainNameOrUnknown("Ethereum Mainnet"))
	require.Equal(t, "<chain unknown>", chainNameOrUnknown(""))
	require.Equal(t, "<chain unknown>", chainNameOrUnknown(" "))
}

func TestBuildProposalReport_FamilyBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		expectedError string
	}{
		{
			name:          "EVM_missing_registry",
			selector:      chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
			expectedError: "EVM registry is not available",
		},
		{
			name:          "Solana_missing_registry",
			selector:      chainsel.SOLANA_DEVNET.Selector,
			expectedError: "failed to analyze solana transaction 0: solana decoder registry is not available",
		},
		{
			name:          "Aptos_unmarshal_additional_fields",
			selector:      chainsel.APTOS_TESTNET.Selector,
			expectedError: "failed to unmarshal Aptos additional fields: unexpected end of JSON input",
		},
		{
			name:          "Sui_unmarshal_additional_fields",
			selector:      chainsel.SUI_TESTNET.Selector,
			expectedError: "failed to unmarshal Sui additional fields: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
			proposal := &mcms.Proposal{
				Operations: []types.Operation{
					{
						ChainSelector: types.ChainSelector(tt.selector),
						Transaction: types.Transaction{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			}

			_, err := BuildProposalReport(ctx, proposal)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestBuildTimelockReport_FamilyBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		expectedError string
	}{
		{
			name:          "EVM_missing_registry",
			selector:      chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
			expectedError: "EVM registry is not available",
		},
		{
			name:          "Solana_missing_registry",
			selector:      chainsel.SOLANA_DEVNET.Selector,
			expectedError: "failed to analyze solana transaction 0: solana decoder registry is not available",
		},
		{
			name:          "Aptos_unmarshal_additional_fields",
			selector:      chainsel.APTOS_TESTNET.Selector,
			expectedError: "failed to unmarshal Aptos additional fields: unexpected end of JSON input",
		},
		{
			name:          "Sui_unmarshal_additional_fields",
			selector:      chainsel.SUI_TESTNET.Selector,
			expectedError: "failed to unmarshal Sui additional fields: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
			proposal := &mcms.TimelockProposal{
				Operations: []types.BatchOperation{
					{
						ChainSelector: types.ChainSelector(tt.selector),
						Transactions: []types.Transaction{
							{To: "0x1111111111111111111111111111111111111111", Data: []byte{0x01, 0x02, 0x03, 0x04}},
							{To: "0x2222222222222222222222222222222222222222", Data: []byte{0x05, 0x06, 0x07, 0x08}},
						},
					},
				},
			}

			_, err := BuildTimelockReport(ctx, proposal)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// Test native token transfer integration with report builder
func TestBuildProposalReport_NativeTransfer(t *testing.T) {
	t.Parallel()

	// Create a context without EVM registry to test native transfers
	ctx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		evmRegistry:      nil, // No registry - native transfers should work
	}

	proposal := &mcms.Proposal{
		Operations: []types.Operation{
			{
				ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Transaction: types.Transaction{
					To:               "0xeE5E8f8Be22101d26084e90053695E2088a01a24",
					Data:             []byte{},                                            // Empty data for native transfer
					AdditionalFields: json.RawMessage(`{"value": "1000000000000000000"}`), // 1 ETH
				},
			},
		},
	}

	report, err := BuildProposalReport(ctx, proposal)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Operations, 1)

	operation := report.Operations[0]
	require.Equal(t, chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, operation.ChainSelector)
	require.Equal(t, chainsel.FamilyEVM, operation.Family)
	require.Len(t, operation.Calls, 1)

	call := operation.Calls[0]
	require.Equal(t, "0xeE5E8f8Be22101d26084e90053695E2088a01a24", call.Address)
	require.Equal(t, "native_transfer", call.Method)
	require.Len(t, call.Inputs, 3)

	// Check recipient
	require.Equal(t, "recipient", call.Inputs[0].Name)
	require.IsType(t, AddressDescriptor{}, call.Inputs[0].Value)
	require.Equal(t, "0xeE5E8f8Be22101d26084e90053695E2088a01a24", call.Inputs[0].Value.(AddressDescriptor).Value)

	// Check amount in wei
	require.Equal(t, "amount_wei", call.Inputs[1].Name)
	require.IsType(t, SimpleDescriptor{}, call.Inputs[1].Value)
	require.Equal(t, "1000000000000000000", call.Inputs[1].Value.(SimpleDescriptor).Value)

	// Check amount in ETH
	require.Equal(t, "amount_eth", call.Inputs[2].Name)
	require.IsType(t, SimpleDescriptor{}, call.Inputs[2].Value)
	require.Equal(t, "1.000000000000000000", call.Inputs[2].Value.(SimpleDescriptor).Value)
}
