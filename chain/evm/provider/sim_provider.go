package provider

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

var (
	// simChainID is the chain ID for the simulated EVM chain. This is always set to 1337 across
	// all instances of EVM Simulated Chains.
	simChainID = params.AllDevChainProtocolChanges.ChainID
	// prefundAmount is the amount of Ether to pre-fund the deployer account with.
	prefundAmountEth = big.NewInt(1_000_000)
	// prefundAmountWei is the prefund amount in wei.
	prefundAmountWei = prefundAmountEth.Mul(prefundAmountEth, big.NewInt(params.Ether))
)

// SimChainProviderConfig holds the configuration to initialize the SimChainProvider.
type SimChainProviderConfig struct {
	// Optional: NumAdditionalAccounts is the number of additional accounts to generate for the
	// simulated chain.
	NumAdditionalAccounts uint
	// Optional: BlockTime configures the time between blocks being committed. By default, this is
	// set to 0s, meaning that blocks are not mined automatically and you must call the Commit
	// method on the Simulated Backend to produce a new block.
	BlockTime time.Duration
}

var _ chain.Provider = (*SimChainProvider)(nil)

// SimChainProvider manages an Simulated EVM chain that is backed by go-ethereum's in memory
// simulated backend.
type SimChainProvider struct {
	t        *testing.T
	selector uint64
	config   SimChainProviderConfig

	chain *evm.Chain
}

// NewSimChainProvider creates a new SimChainProvider with the given selector and configuration.
func NewSimChainProvider(
	t *testing.T, selector uint64, config SimChainProviderConfig,
) *SimChainProvider {
	t.Helper()

	return &SimChainProvider{
		t:        t,
		selector: selector,
		config:   config,
	}
}

// Initialize sets up the simulated chain with a deployer account and additional accounts as
// specified in the configuration. It returns an initialized evm.Chain instance that can be used
// to interact with the simulated chain.
//
// Each account is prefunded with 1,000,000 Ether (1 million wei).
func (p *SimChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	// Generate a deployer account
	key, err := crypto.GenerateKey()
	require.NoError(p.t, err, "failed to generate deployer key")

	adminTransactor, err := bind.NewKeyedTransactorWithChainID(key, simChainID)
	require.NoError(p.t, err)

	// Prefund the admin account
	genesis := types.GenesisAlloc{
		adminTransactor.From: {Balance: prefundAmountWei},
	}

	// Generate keys for additional accounts
	additionalTransactors := make([]*bind.TransactOpts, 0, p.config.NumAdditionalAccounts)
	for range p.config.NumAdditionalAccounts {
		key, err := crypto.GenerateKey()
		require.NoError(p.t, err)

		transactor, err := bind.NewKeyedTransactorWithChainID(key, simChainID)
		require.NoError(p.t, err)

		additionalTransactors = append(additionalTransactors, transactor)

		// Prefund each additional account
		genesis[transactor.From] = types.Account{Balance: prefundAmountWei}
	}

	// Initialize the simulated backend with the genesis state
	backend := simulated.NewBackend(genesis, simulated.WithBlockGasLimit(50000000))
	backend.Commit() // Commit the genesis block

	// Start mining blocks if a block time is configured
	if p.config.BlockTime > 0 {
		startAutoMine(p.t, backend, p.config.BlockTime)
	}

	// Wrap the simulated client to implement the OnchainClient interface. This allows us to use
	// the simulated client as a client for the evm.Chain.
	client := NewSimClient(p.t, backend)

	p.chain = &evm.Chain{
		Selector:    p.selector,
		Client:      client,
		DeployerKey: adminTransactor,
		Users:       additionalTransactors,
		Confirm: func(tx *types.Transaction) (uint64, error) {
			if tx == nil {
				return 0, fmt.Errorf("tx was nil, nothing to confirm for selector: %d", p.selector)
			}

			// Ensure the transaction is mined by committing a new block
			client.Commit()

			receipt, err := func() (*types.Receipt, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()

				return bind.WaitMined(ctx, client, tx)
			}()
			if err != nil {
				return 0, fmt.Errorf("tx %s failed to confirm for selector %d: %w",
					tx.Hash().Hex(), p.selector, err,
				)
			}

			if receipt.Status == 0 {
				reason, err := getErrorReasonFromTx(
					p.t.Context(), client, adminTransactor.From, tx, receipt,
				)
				if err == nil && reason != "" {
					return 0, fmt.Errorf("tx %s reverted for selector %d: %s",
						tx.Hash().Hex(), p.selector, reason,
					)
				}

				return 0, fmt.Errorf("tx %s reverted, could not decode error reason for selector %d",
					tx.Hash().Hex(), p.selector,
				)
			}

			return receipt.BlockNumber.Uint64(), nil
		},
	}

	return *p.chain, nil
}

// Name returns the name of the SimChainProvider.
func (*SimChainProvider) Name() string {
	return "Simulated EVM Chain Provider"
}

// ChainSelector returns the chain selector of the simulated chain managed by this provider.
func (p *SimChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the simulated chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *SimChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// startAutoMine triggers the simulated backend to create a new block at intervals defined by
// `blockTime`. After the test is done, it stops the mining goroutine.
func startAutoMine(t *testing.T, backend *simulated.Backend, blockTime time.Duration) {
	t.Helper()

	ctx := t.Context() // Available since Go 1.20
	ticker := time.NewTicker(blockTime)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				backend.Commit()
			case <-ctx.Done():
				return
			}
		}
	}()
}
