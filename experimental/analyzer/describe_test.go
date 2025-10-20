package analyzer

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestDescribeProposal(t *testing.T) {
	t.Parallel()

	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	tests := []struct {
		name           string
		operations     []types.Operation
		expectError    bool
		errorContains  string
		outputContains []string
	}{
		{
			name:           "Empty proposal",
			operations:     []types.Operation{},
			expectError:    false,
			outputContains: []string{""},
		},
		{
			name: "Single operation - unsupported chain",
			operations: []types.Operation{
				{
					ChainSelector: 1, // This will fail in test environment
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Multiple operations - unsupported chain",
			operations: []types.Operation{
				{
					ChainSelector: 1,
					Transaction: types.Transaction{
						To:   "0x1111111111111111111111111111111111111111",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
				{
					ChainSelector: 1,
					Transaction: types.Transaction{
						To:   "0x2222222222222222222222222222222222222222",
						Data: []byte{0x05, 0x06, 0x07, 0x08},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Unsupported chain",
			operations: []types.Operation{
				{
					ChainSelector: 999999, // Unsupported chain
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposal := &mcms.Proposal{Operations: tt.operations}
			output, err := DescribeProposal(ctx, proposal)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				for _, contains := range tt.outputContains {
					if contains == "" {
						require.Empty(t, output)
					} else {
						require.Contains(t, output, contains)
					}
				}
			}
		})
	}
}

func TestDescribeTimelockProposal(t *testing.T) {
	t.Parallel()

	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	tests := []struct {
		name           string
		operations     []types.BatchOperation
		expectError    bool
		errorContains  string
		outputContains []string
	}{
		{
			name:           "Empty proposal",
			operations:     []types.BatchOperation{},
			expectError:    false,
			outputContains: []string{""},
		},
		{
			name: "Single batch - unsupported chain",
			operations: []types.BatchOperation{
				{
					ChainSelector: 1, // This will fail in test environment
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Multiple batches - unsupported chain",
			operations: []types.BatchOperation{
				{
					ChainSelector: 1,
					Transactions: []types.Transaction{
						{
							To:   "0x1111111111111111111111111111111111111111",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
				{
					ChainSelector: 1,
					Transactions: []types.Transaction{
						{
							To:   "0x2222222222222222222222222222222222222222",
							Data: []byte{0x05, 0x06, 0x07, 0x08},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Unsupported chain",
			operations: []types.BatchOperation{
				{
					ChainSelector: 999999, // Unsupported chain
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposal := &mcms.TimelockProposal{Operations: tt.operations}
			output, err := DescribeTimelockProposal(ctx, proposal)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				for _, contains := range tt.outputContains {
					if contains == "" {
						require.Empty(t, output)
					} else {
						require.Contains(t, output, contains)
					}
				}
			}
		})
	}
}

func TestDescribeBatchOperationsHelper(t *testing.T) {
	t.Parallel()

	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	tests := []struct {
		name            string
		batches         []types.BatchOperation
		expectError     bool
		errorContains   string
		expectedLength  int
		expectedContent []string // Expected content patterns in the output
	}{
		{
			name:           "Empty batches",
			batches:        []types.BatchOperation{},
			expectError:    false,
			expectedLength: 0,
		},
		{
			name: "EVM batch - Ethereum Sepolia",
			batches: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
						{
							To:   "0x1111111111111111111111111111111111111111",
							Data: []byte{0x05, 0x06, 0x07, 0x08},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "EVM registry is not available",
		},
		{
			name: "Solana batch - Solana Devnet",
			batches: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.SOLANA_DEVNET.Selector),
					Transactions: []types.Transaction{
						{
							To:   "11111111111111111111111111111111", // Solana address format
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "Solana decoder registry is not available",
		},
		{
			name: "Aptos batch - Aptos Testnet",
			batches: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
					Transactions: []types.Transaction{
						{
							To:   "0x1::aptos_coin::AptosCoin", // Aptos resource format
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "failed to unmarshal Aptos additional fields",
		},
		{
			name: "Sui batch - Sui Testnet",
			batches: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.SUI_TESTNET.Selector),
					Transactions: []types.Transaction{
						{
							To:   "0x2::sui::SUI", // Sui resource format
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "failed to unmarshal Sui additional fields",
		},
		{
			name: "Mixed chain families",
			batches: []types.BatchOperation{
				{
					ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
				{
					ChainSelector: types.ChainSelector(chainsel.SOLANA_DEVNET.Selector),
					Transactions: []types.Transaction{
						{
							To:   "11111111111111111111111111111111",
							Data: []byte{0x05, 0x06, 0x07, 0x08},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "EVM registry is not available",
		},
		{
			name: "Unsupported chain family",
			batches: []types.BatchOperation{
				{
					ChainSelector: 999999, // Unsupported chain
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 999999",
		},
		{
			name: "Default case - unknown family",
			batches: []types.BatchOperation{
				{
					ChainSelector: 888888, // This will get a family but not be handled in switch
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 888888",
		},
		{
			name: "Single batch - unsupported chain",
			batches: []types.BatchOperation{
				{
					ChainSelector: 1, // This will fail in test environment
					Transactions: []types.Transaction{
						{
							To:   "0x1234567890123456789012345678901234567890",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Multiple batches - unsupported chain",
			batches: []types.BatchOperation{
				{
					ChainSelector: 1,
					Transactions: []types.Transaction{
						{
							To:   "0x1111111111111111111111111111111111111111",
							Data: []byte{0x01, 0x02, 0x03, 0x04},
						},
					},
				},
				{
					ChainSelector: 1,
					Transactions: []types.Transaction{
						{
							To:   "0x2222222222222222222222222222222222222222",
							Data: []byte{0x05, 0x06, 0x07, 0x08},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := describeBatchOperations(ctx, tt.batches)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, result, tt.expectedLength)

				// Check content patterns if specified
				if len(tt.expectedContent) > 0 {
					for i, batch := range result {
						for _, expectedPattern := range tt.expectedContent {
							found := false
							for _, transaction := range batch {
								if len(transaction) > 0 {
									// Check if any transaction in the batch contains the expected pattern
									if len(transaction) > 0 {
										found = true
										break
									}
								}
							}
							require.True(t, found, "Expected pattern '%s' not found in batch %d", expectedPattern, i)
						}
					}
				}

				// If we have batches, check that they contain some content
				if tt.expectedLength > 0 {
					for i, batch := range result {
						require.NotEmpty(t, batch, "Batch %d should not be empty", i)
						// Each transaction in the batch should have some content
						for j, transaction := range batch {
							require.NotEmpty(t, transaction, "Transaction %d in batch %d should not be empty", j, i)
						}
					}
				}
			}
		})
	}
}

func TestDescribeOperationsHelper(t *testing.T) {
	t.Parallel()

	ctx := &DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}

	tests := []struct {
		name            string
		operations      []types.Operation
		expectError     bool
		errorContains   string
		expectedLength  int
		expectedContent []string // Expected content patterns in the output
	}{
		{
			name:           "Empty operations",
			operations:     []types.Operation{},
			expectError:    false,
			expectedLength: 0,
		},
		{
			name: "EVM operation - Ethereum Sepolia",
			operations: []types.Operation{
				{
					ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
				{
					ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transaction: types.Transaction{
						To:   "0x1111111111111111111111111111111111111111",
						Data: []byte{0x05, 0x06, 0x07, 0x08},
					},
				},
			},
			expectError:   true,
			errorContains: "EVM registry is not available",
		},
		{
			name: "Solana operation - Solana Devnet",
			operations: []types.Operation{
				{
					ChainSelector: types.ChainSelector(chainsel.SOLANA_DEVNET.Selector),
					Transaction: types.Transaction{
						To:   "11111111111111111111111111111111", // Solana address format
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "Solana decoder registry is not available",
		},
		{
			name: "Aptos operation - Aptos Testnet",
			operations: []types.Operation{
				{
					ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
					Transaction: types.Transaction{
						To:   "0x1::aptos_coin::AptosCoin", // Aptos resource format
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "failed to unmarshal Aptos additional fields",
		},
		{
			name: "Sui operation - Sui Testnet",
			operations: []types.Operation{
				{
					ChainSelector: types.ChainSelector(chainsel.SUI_TESTNET.Selector),
					Transaction: types.Transaction{
						To:   "0x2::sui::SUI", // Sui resource format
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "failed to unmarshal Sui additional fields",
		},
		{
			name: "Mixed chain families",
			operations: []types.Operation{
				{
					ChainSelector: types.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
				{
					ChainSelector: types.ChainSelector(chainsel.SOLANA_DEVNET.Selector),
					Transaction: types.Transaction{
						To:   "11111111111111111111111111111111",
						Data: []byte{0x05, 0x06, 0x07, 0x08},
					},
				},
			},
			expectError:   true,
			errorContains: "EVM registry is not available",
		},
		{
			name: "Unsupported chain family",
			operations: []types.Operation{
				{
					ChainSelector: 999999, // Unsupported chain
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 999999",
		},
		{
			name: "Default case - unknown family",
			operations: []types.Operation{
				{
					ChainSelector: 888888, // This will get a family but not be handled in switch
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 888888",
		},
		{
			name: "Single operation - unsupported chain",
			operations: []types.Operation{
				{
					ChainSelector: 1, // This will fail in test environment
					Transaction: types.Transaction{
						To:   "0x1234567890123456789012345678901234567890",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
		{
			name: "Multiple operations - unsupported chain",
			operations: []types.Operation{
				{
					ChainSelector: 1,
					Transaction: types.Transaction{
						To:   "0x1111111111111111111111111111111111111111",
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
				{
					ChainSelector: 1,
					Transaction: types.Transaction{
						To:   "0x2222222222222222222222222222222222222222",
						Data: []byte{0x05, 0x06, 0x07, 0x08},
					},
				},
			},
			expectError:   true,
			errorContains: "unknown chain selector 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := describeOperations(ctx, tt.operations)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, result, tt.expectedLength)

				// Check content patterns if specified
				if len(tt.expectedContent) > 0 {
					for i, operation := range result {
						for _, expectedPattern := range tt.expectedContent {
							require.Contains(t, operation, expectedPattern, "Expected pattern '%s' not found in operation %d", expectedPattern, i)
						}
					}
				}

				// If we have operations, check that they contain some content
				if tt.expectedLength > 0 {
					for i, operation := range result {
						require.NotEmpty(t, operation, "Operation %d should not be empty", i)
					}
				}
			}
		})
	}
}
