package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestDescribeProposal(t *testing.T) {
	t.Parallel()

	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		renderer:         NewMarkdownRenderer(),
	}

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
			output, err := DescribeProposal(t.Context(), proposalCtx, deployment.Environment{}, proposal)

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

	proposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{},
		renderer:         NewMarkdownRenderer(),
	}

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
			output, err := DescribeTimelockProposal(t.Context(), proposalCtx, deployment.Environment{}, proposal)

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
