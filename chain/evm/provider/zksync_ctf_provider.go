package provider

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	zkAccounts "github.com/zksync-sdk/zksync2-go/accounts"
	zkClients "github.com/zksync-sdk/zksync2-go/clients"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

// ZkSyncCTFChainProviderConfig holds the configuration to initialize the ZkSyncCTFChainProvider.
type ZkSyncCTFChainProviderConfig struct {
	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once
}

// validate checks if the config fields are valid.
func (c ZkSyncCTFChainProviderConfig) validate() error {
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	return nil
}

var _ chain.Provider = (*ZkSyncCTFChainProvider)(nil)

// ZkSyncCTFChainProvider manages an ZkSync EVM chain instance running inside a (CTF) Docker
// container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container
// can be slow, so it is recommended to initialize the provider only once per test suite or parent
// test to optimize performance.
type ZkSyncCTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   ZkSyncCTFChainProviderConfig

	chain *evm.Chain
}

// NewZkCTFChainProvider creates a new ZkSyncCTFChainProvider with the given selector and
// configuration.
func NewZkSyncCTFChainProvider(
	t *testing.T, selector uint64, config ZkSyncCTFChainProviderConfig,
) *ZkSyncCTFChainProvider {
	t.Helper()

	p := &ZkSyncCTFChainProvider{
		t:        t,
		selector: selector,
		config:   config,
	}

	return p
}

// Initialize sets up the ZkSync EVM chain instance managed by this provider. It starts a CTF
// container, initializes the Ethereum client, and sets up the chain instance with the necessary
// transactors and deployer key gathered from the CTF's default zkSync accounts. The first
// account is used as the deployer key, and the rest are used as users for the chain.
func (p *ZkSyncCTFChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	err := p.config.validate()
	require.NoError(p.t, err, "failed to validate provider config")

	// Get the Chain ID
	chainID, err := chain_selectors.GetChainIDFromSelector(p.selector)
	require.NoError(p.t, err, "failed to get chain ID from selector")

	// Start the Zksync CTF container
	httpURL := p.startContainer(chainID)

	// Setup the Ethereum client
	client, err := ethclient.Dial(httpURL)
	require.NoError(p.t, err)

	// Fetch the suggested gas price for the chain to set on the transactors.
	// Anvil zkSync does not support eth_maxPriorityFeePerGas so we set gasPrice to force using
	// createLegacyTx
	gasPrice, err := client.SuggestGasPrice(ctx)
	require.NoError(p.t, err)

	// Build transactors from the default accounts provided by the CTF
	transactors := p.getTransactors(chainID, gasPrice)

	// Create SignHash function from the deployer's private key
	// assume the first account is the deployer
	deployerPrivateKey, err := crypto.HexToECDSA(blockchain.AnvilZKSyncRichAccountPks[0])
	require.NoError(p.t, err, "failed to parse deployer private key")

	// Initialize the zksync client and wallet
	clientZk := zkClients.NewClient(client.Client())
	deployerZk, err := zkAccounts.NewWallet(
		common.Hex2Bytes(blockchain.AnvilZKSyncRichAccountPks[0]), clientZk, nil,
	)
	require.NoError(p.t, err, "failed to create deployer wallet for ZkSync")

	// Construct the chain
	p.chain = &evm.Chain{
		Selector:    p.selector,
		Client:      client,
		DeployerKey: transactors[0],  // The first transactor is the deployer
		Users:       transactors[1:], // The rest are users
		Confirm: func(tx *types.Transaction) (uint64, error) {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			receipt, err := bind.WaitMined(ctxWithTimeout, client, tx)
			if err != nil {
				return 0, err
			}

			return receipt.Status, nil
		},
		SignHash: func(hash []byte) ([]byte, error) {
			return crypto.Sign(hash, deployerPrivateKey)
		},
		IsZkSyncVM:          true,
		ClientZkSyncVM:      clientZk,
		DeployerKeyZkSyncVM: deployerZk,
	}

	return *p.chain, nil
}

// Name returns the name of the ZkSyncCTFChainProvider.
func (*ZkSyncCTFChainProvider) Name() string {
	return "ZkSync EVM CTF Chain Provider"
}

// ChainSelector returns the chain selector of the ZkSync EVM chain managed by this provider.
func (p *ZkSyncCTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the ZkSync EVM chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *ZkSyncCTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// startContainer starts a CTF container for the ZkSync EVM returning the HTTP URL of the node.
//
// Due to the docker container setup making a flakey curl command, we use a retry mechanism
// to ensure the container is fully up and running before proceeding.
func (p *ZkSyncCTFChainProvider) startContainer(
	chainID string,
) string {
	var (
		attempts = uint(10)
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	httpURL, err := retry.DoWithData(func() (string, error) {
		// Initialize a port for the container
		port := freeport.GetOne(p.t)

		// Create the CTF container for ZkSync
		output, rerr := blockchain.NewBlockchainNetwork(&blockchain.Input{
			Type:    "anvil-zksync",
			ChainID: chainID,
			Port:    strconv.Itoa(port),
		})
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			freeport.Return([]int{port})

			return "", rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		return output.Nodes[0].ExternalHTTPUrl, nil
	},
		retry.Context(p.t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	require.NoError(p.t, err, "Failed to start CTF ZkSync container after %d attempts", attempts)

	return httpURL
}

// getTransactors generates transactors from the default list of accounts provided by the CTF.
func (p *ZkSyncCTFChainProvider) getTransactors(
	chainID string, gasPrice *big.Int,
) []*bind.TransactOpts {
	require.Greater(p.t, len(blockchain.AnvilZKSyncRichAccountPks), 1)

	cid, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		require.FailNowf(p.t, "failed to parse chain ID into big.Int: %s", chainID)
	}

	transactors := make([]*bind.TransactOpts, 0)
	for _, pk := range blockchain.AnvilZKSyncRichAccountPks {
		privateKey, err := crypto.HexToECDSA(pk)
		require.NoError(p.t, err)

		transactor, err := bind.NewKeyedTransactorWithChainID(privateKey, cid)
		transactor.GasPrice = gasPrice
		require.NoError(p.t, err)

		transactors = append(transactors, transactor)
	}

	return transactors
}
