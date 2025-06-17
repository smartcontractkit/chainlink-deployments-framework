package provider

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"
)

func Test_SimClient_Commit(t *testing.T) {
	t.Parallel()

	// Create a simulated backend
	sim := simulated.NewBackend(types.GenesisAlloc{})

	// Create a new SimClient instance
	client := NewSimClient(t, sim)

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(0), blockNumber) // No blocks have been mined yet

	// Commit the changes to the simulated backend
	hash := client.Commit()
	require.NotEmpty(t, hash)

	blockNumber, err = client.BlockNumber(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(1), blockNumber) // After commit, the block number should be 1
}
