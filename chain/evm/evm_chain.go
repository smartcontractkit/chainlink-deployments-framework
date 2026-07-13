package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/zksync-sdk/zksync2-go/accounts"
	"github.com/zksync-sdk/zksync2-go/clients"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
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
	// StorageAt reads a storage slot from the given account at the specified block number.
	// This is needed for operations like EIP-1967 proxy detection.
	StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error)
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

	// GasConfig holds per-chain default gas settings and optional retry boost configuration.
	GasConfig *gas.Config

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

// NetworkType returns the type of network the chain is on (e.g. mainnet, testnet)
func (c Chain) NetworkType() (chainsel.NetworkType, error) {
	return chaincommon.ChainMetadata{Selector: c.Selector}.NetworkType()
}

// IsNetworkType checks if the chain is on the given network type
func (c Chain) IsNetworkType(networkType chainsel.NetworkType) bool {
	return chaincommon.ChainMetadata{Selector: c.Selector}.IsNetworkType(networkType)
}

func (c Chain) ReadOnly() (chaincommon.BlockChain, error) {
	if c.DeployerKey == nil {
		return c, nil
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key for read-only chain: %w", err)
	}
	c.DeployerKey = pointer.To(*c.DeployerKey)
	c.DeployerKey.From = crypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))

	return c, nil
}
