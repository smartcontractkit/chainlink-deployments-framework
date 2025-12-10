package provider

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"
)

// SimClient is a wrapper struct around a simulated backend which implements OnchainClient but
// also exposes backend methods.
type SimClient struct {
	mu sync.Mutex

	// Embed the simulated.Client to provide access to its methods and adhere to the OnchainClient interface.
	simulated.Client
	// backend is the underlying simulated backend that this client wraps.
	backend *simulated.Backend
}

// NewSimClient creates a new wrappedSimBackend instance from a simulated backend.
func NewSimClient(t *testing.T, backend *simulated.Backend) *SimClient {
	t.Helper()

	require.NotNil(t, backend, "simulated backend must not be nil")

	return &SimClient{
		backend: backend,
		Client:  backend.Client(),
	}
}

func (b *SimClient) Commit() common.Hash {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.backend.Commit()
}

func (b *SimClient) Backend() *simulated.Backend {
	return b.backend
}
