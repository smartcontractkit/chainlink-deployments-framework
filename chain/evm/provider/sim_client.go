package provider

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"
)

// SimClient is a wrapper struct around a simulated backend which implements OnchainClient but
// also exposes backend methods.
type SimClient struct {
	mu sync.Mutex

	sim *simulated.Backend
}

// NewSimClient creates a new wrappedSimBackend instance from a simulated backend.
func NewSimClient(t *testing.T, sim *simulated.Backend) *SimClient {
	t.Helper()

	require.NotNil(t, sim, "simulated backend must not be nil")

	return &SimClient{
		sim: sim,
	}
}

func (b *SimClient) Commit() common.Hash {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.sim.Commit()
}

func (b *SimClient) BlockNumber(ctx context.Context) (uint64, error) {
	return b.sim.Client().BlockNumber(ctx)
}

func (b *SimClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.sim.Client().CodeAt(ctx, contract, blockNumber)
}

func (b *SimClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return b.sim.Client().CallContract(ctx, call, blockNumber)
}

func (b *SimClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	return b.sim.Client().EstimateGas(ctx, call)
}

func (b *SimClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.sim.Client().SuggestGasPrice(ctx)
}

func (b *SimClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.sim.Client().SuggestGasTipCap(ctx)
}

func (b *SimClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return b.sim.Client().SendTransaction(ctx, tx)
}

func (b *SimClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return b.sim.Client().HeaderByNumber(ctx, number)
}

func (b *SimClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return b.sim.Client().PendingCodeAt(ctx, account)
}

func (b *SimClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return b.sim.Client().PendingNonceAt(ctx, account)
}

func (b *SimClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return b.sim.Client().FilterLogs(ctx, q)
}

func (b *SimClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return b.sim.Client().SubscribeFilterLogs(ctx, q, ch)
}

func (b *SimClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return b.sim.Client().TransactionReceipt(ctx, txHash)
}

func (b *SimClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return b.sim.Client().BalanceAt(ctx, account, blockNumber)
}

func (b *SimClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return b.sim.Client().NonceAt(ctx, account, blockNumber)
}
