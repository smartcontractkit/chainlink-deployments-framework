package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zksync-sdk/zksync2-go/accounts"
	"github.com/zksync-sdk/zksync2-go/clients"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// ConfirmFunc is a function that takes a transaction, waits for the transaction to be confirmed,
// and returns the block number and an error.
type ConfirmFunc func(tx *types.Transaction) (uint64, error)

// OnchainClient is an EVM chain client.
// For EVM specifically we can use existing geth interface to abstract chain clients.
type OnchainClient interface {
	bind.ContractBackend
	bind.DeployBackend

	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

// Chain represents an EVM chain.
type Chain struct {
	Selector uint64

	Client OnchainClient
	// Note the Sign function can be abstract supporting a variety of key storage mechanisms (e.g. KMS etc).
	DeployerKey *bind.TransactOpts
	Confirm     ConfirmFunc
	// Users are a set of keys that can be used to interact with the chain.
	// These are distinct from the deployer key.
	Users []*bind.TransactOpts

	// SignHash allows signing of arbitrary hashes using the deployer key's signing mechanism.
	// This function signature matches the expected format: func([]byte) ([]byte, error)
	SignHash func([]byte) ([]byte, error)

	// ZK deployment specifics
	IsZkSyncVM          bool
	ClientZkSyncVM      *clients.Client
	DeployerKeyZkSyncVM *accounts.Wallet
}

// Author note: Have to implement the blockhain interface methods explicitly below
// instead of composing the ChainMetadata struct to avoid breaking change since there are existing usage.

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Name()
}

// Family returns the family of the chain
func (c Chain) Family() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Family()
}
