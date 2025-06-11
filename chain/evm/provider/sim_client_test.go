package provider

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"
)

func Test_SimClient_Delegates(t *testing.T) {
	t.Parallel()

	var (
		contractAddr = common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	)

	tests := []struct {
		name    string
		runFunc func(*testing.T, *SimClient)
	}{
		{
			name: "Commit",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				blockNumber, err := c.BlockNumber(t.Context())
				require.NoError(t, err)
				require.Equal(t, uint64(0), blockNumber) // No blocks have been mined yet

				// Commit the changes to the simulated backend
				hash := c.Commit()
				require.NotEmpty(t, hash)

				blockNumber, err = c.BlockNumber(t.Context())
				require.NoError(t, err)
				require.Equal(t, uint64(1), blockNumber) // After commit, the block number should be 1
			},
		},
		{
			name: "BlockNumber",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				blockNumber, err := c.BlockNumber(t.Context())
				require.NoError(t, err)
				require.Equal(t, uint64(0), blockNumber) // No blocks have been mined yet
			},
		},
		{
			name: "CodeAt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				blah, err := c.CodeAt(t.Context(), contractAddr, nil)
				require.NoError(t, err)
				require.Empty(t, blah) // Empty because no code is deployed at this address in the simulated backend
			},
		},
		{
			name: "CallContract",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				// CallContract should return an error because no contract is deployed at this address
				b, err := c.CallContract(t.Context(), ethereum.CallMsg{
					To:   &contractAddr,
					Data: []byte{},
				}, big.NewInt(0))
				require.NoError(t, err)
				require.Empty(t, b) // Should return empty bytes since no contract is deployed
			},
		},
		{
			name: "EstimateGas",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				gas, err := c.EstimateGas(t.Context(), ethereum.CallMsg{
					To:   &common.Address{},
					Data: []byte{},
				})
				require.NoError(t, err)
				require.Equal(t, uint64(21000), gas) // Expect the base gas cost
			},
		},
		{
			name: "SuggestGasPrice",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				suggestedGasPrice, err := c.SuggestGasPrice(t.Context())
				require.NoError(t, err)
				require.NotEmpty(t, suggestedGasPrice) // Should return some valid gas price
			},
		},
		{
			name: "SuggestGasTipCap",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				suggestedGasTipCap, err := c.SuggestGasTipCap(t.Context())
				require.NoError(t, err)
				require.NotEmpty(t, suggestedGasTipCap) // Should return some valid gas tip cap
			},
		},
		{
			name: "SendTransaction",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				// Sending a transaction should not work since no account is funded in the simulated backend
				tx := types.NewTransaction(0, contractAddr, big.NewInt(0), 21000, big.NewInt(1), nil)
				err := c.SendTransaction(t.Context(), tx)
				require.Error(t, err) // Expect an error since no account is funded
			},
		},
		{
			name: "HeaderByNumber",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				header, err := c.HeaderByNumber(t.Context(), nil)
				require.NoError(t, err)
				require.NotNil(t, header) // Should return the latest block header
			},
		},
		{
			name: "PendingCodeAt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				blah, err := c.PendingCodeAt(t.Context(), contractAddr)
				require.NoError(t, err)
				require.Empty(t, blah) // Empty because no code is deployed at this address in the simulated backend
			},
		},
		{
			name: "PendingNonceAt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				nonce, err := c.PendingNonceAt(t.Context(), contractAddr)
				require.NoError(t, err)
				require.Equal(t, uint64(0), nonce) // Expect the nonce to be 0 in the simulated backend
			},
		},
		{
			name: "FilterLogs",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				logs, err := c.FilterLogs(t.Context(), ethereum.FilterQuery{
					FromBlock: nil,
					ToBlock:   nil,
					Addresses: []common.Address{contractAddr},
				})
				require.NoError(t, err)
				require.Empty(t, logs) // No logs should be present in the simulated backend
			},
		},
		{
			name: "SubscribeFilterLogs",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				ch := make(chan types.Log)
				sub, err := c.SubscribeFilterLogs(t.Context(), ethereum.FilterQuery{
					FromBlock: nil,
					ToBlock:   nil,
					Addresses: []common.Address{contractAddr},
				}, ch)
				require.NoError(t, err)

				// Ensure the subscription is valid
				require.NotNil(t, sub)

				// Close the channel to avoid leaks
				close(ch)
			},
		},
		{
			name: "TransactionReceipt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				// There are no transactions for the empty hash so this will return an error
				receipt, err := c.TransactionReceipt(t.Context(), common.Hash{})
				require.Error(t, err)
				require.Nil(t, receipt)
			},
		},
		{
			name: "BalanceAt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				balance, err := c.BalanceAt(t.Context(), contractAddr, nil)
				require.NoError(t, err)
				require.Equal(t, uint64(0), balance.Uint64())
			},
		},
		{
			name: "NonceAt",
			runFunc: func(t *testing.T, c *SimClient) {
				t.Helper()

				nonce, err := c.NonceAt(t.Context(), contractAddr, nil)
				require.NoError(t, err)
				require.Equal(t, uint64(0), nonce) // Expect the nonce to be 0 since no transactions have been sent from this address
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a simulated backend
			sim := simulated.NewBackend(types.GenesisAlloc{})

			// Create a new SimClient instance
			client := NewSimClient(t, sim)

			// Run the test function
			tt.runFunc(t, client)
		})
	}
}
