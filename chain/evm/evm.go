package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zksync-sdk/zksync2-go/accounts"
	"github.com/zksync-sdk/zksync2-go/clients"

	chain_common "github.com/smartcontractkit/chainlink-deployments-framework/chain/common"
)

// OnchainClient is an EVM chain client.
// For EVM specifically we can use existing geth interface
// to abstract chain clients.
type OnchainClient interface {
	bind.ContractBackend
	bind.DeployBackend
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

// Chain represents an EVM chain.
type Chain struct {
	chain_common.ChainInfoProvider

	Client OnchainClient
	// Note the Sign function can be abstract supporting a variety of key storage mechanisms (e.g. KMS etc).
	DeployerKey *bind.TransactOpts
	Confirm     func(tx *types.Transaction) (uint64, error)
	// Users are a set of keys that can be used to interact with the chain.
	// These are distinct from the deployer key.
	Users []*bind.TransactOpts

	// ZK deployment specifics
	IsZkSyncVM          bool
	ClientZkSyncVM      *clients.Client
	DeployerKeyZkSyncVM *accounts.Wallet
}
