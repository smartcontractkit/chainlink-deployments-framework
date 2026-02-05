package environment

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-resty/resty/v2"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

var oneEth = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)

// anvilClient operates the methods exposed by the Anvil node related to forking.
// For more information, see https://book.getfoundry.sh/reference/anvil/#custom-methods.
// TODO: We can also move this to the existing Anvil client here: https://github.com/smartcontractkit/chainlink-testing-framework/blob/main/framework/rpc/rpc.go.
type anvilClient struct {
	url    string
	client *resty.Client
}

// newAnvilForkClient creates a new client that can utilize Anvil's forking capabilities
func newAnvilForkClient(url string, headers map[string]string, tls *tls.Config, debug bool) ForkedOnchainClient {
	return &anvilClient{
		url: url,
		client: resty.New().
			SetDebug(debug).
			SetHeaders(headers).
			SetTLSClientConfig(tls),
	}
}

// SendTransaction sends a transaction, ensuring the sender address is properly funded.
// The sender of the transaction, whether EOA or contract, will be impersonated assuming --auto-impersonate has been supplied on startup.
// We do not call anvil_autoImpersonateAccount in this function to reduce the number of RPC calls made.
// If we are forking a chain, it should be assumed that we will always enable auto impersonation.
func (c *anvilClient) SendTransaction(ctx context.Context, from string, to string, data []byte) error {
	err := c.setBalance(ctx, from, oneEth)
	if err != nil {
		return fmt.Errorf("failed to update balance of %s to 1 ETH: %w", from, err)
	}

	err = c.post(ctx, "eth_sendTransaction", map[string]string{
		"to":   to,
		"from": from,
		"data": hexutil.Encode(data),
	})
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Mine the transaction to properly update state.
	// Inputted block parameters are arbitrary.
	err = c.mine(ctx, 1, 1*time.Second)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	return nil
}

// setBalance updates the balance of an account.
func (c *anvilClient) setBalance(ctx context.Context, account string, balance *big.Int) error {
	return c.post(ctx, "anvil_setBalance", account, balance.String())
}

// mine mines a series of blocks.
// Note: evm_setAutomine could be an alternative, but did not seem to be triggering state updates.
func (c *anvilClient) mine(ctx context.Context, numBlocks uint64, timeBetweenBlocks time.Duration) error {
	return c.post(ctx, "anvil_mine", numBlocks, timeBetweenBlocks.Seconds())
}

// post submits data to the anvil node.
func (c *anvilClient) post(ctx context.Context, method string, params ...any) error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      rand.Int(), //nolint:gosec // G404: Only used for fork testing, so not a security risk.
	}
	if _, err := c.client.R().SetContext(ctx).SetBody(payload).Post(c.url); err != nil {
		return fmt.Errorf("failed to call %s: %w", method, err)
	}

	return nil
}

// RPCs represents the internal and external RPCs for a chain.
type RPCs struct {
	External string
}

// ChainConfig represents the configuration for a chain.
type ChainConfig struct {
	ChainID  string // chain id as per EIP-155
	HTTPRPCs []RPCs // http rpcs to connect to the chain
}

// AnvilChainsOutput represents the output of the newAnvilChains function.
type AnvilChainsOutput struct {
	Chains       map[uint64]fevm.Chain
	ForkClients  map[uint64]ForkedOnchainClient
	ChainConfigs map[uint64]ChainConfig
}

// newAnvilChains creates chain abstractions using local anvil nodes.
func newAnvilChains(
	ctx context.Context,
	lggr logger.Logger,
	addressBook fdeployment.AddressBook,
	dataStore datastore.DataStore,
	evmNetworks *cfgnet.Config,
	blockNumbers map[uint64]*big.Int,
	onchainConfig cfgenv.OnchainConfig,
	kmsConfig cfgenv.KMSConfig,
	chainSelectorsToLoad []uint64,
	anvilKeyAsDeployer bool,
) (*AnvilChainsOutput, error) {
	// filter out not in blockNumbers, if any, to ensure we only use the chains we care about for forking
	filteredEvmNetworks := make([]cfgnet.Network, 0, len(blockNumbers))
	if blockNumbers != nil {
		for _, network := range evmNetworks.Networks() {
			if _, ok := blockNumbers[network.ChainSelector]; !ok {
				lggr.Warnf("skipping chain selector %d for forking as it is not in the provided blockNumbers map", network.ChainSelector)
				// remove this rpc from the list of evm networks for forking
				continue
			}

			filteredEvmNetworks = append(filteredEvmNetworks, network)
		}

		if len(filteredEvmNetworks) == 0 {
			return nil, errors.New("no evm networks found for forking in the provided blockNumbers map")
		}
	} else {
		// if no fork block numbers were specified fork all the chains
		filteredEvmNetworks = evmNetworks.Networks()
	}

	chainConfigsBySelector := make(map[uint64]ChainConfig)
	anvilClients := make(map[uint64]ForkedOnchainClient)
	addressesByChain, err1 := addressBook.Addresses()
	if err1 != nil {
		return nil, fmt.Errorf("failed to get addresses by chain selector: %w", err1)
	}
	dataStoreAddresses, err1 := dataStore.Addresses().Fetch()
	if err1 != nil {
		return nil, fmt.Errorf("failed to get addresses from data store: %w", err1)
	}
	for _, address := range dataStoreAddresses {
		addressesByChain[address.ChainSelector] = map[string]fdeployment.TypeAndVersion{}
	}

	var once sync.Once
	blockChains := make([]fchain.BlockChain, 0, len(filteredEvmNetworks))
	for _, network := range filteredEvmNetworks {
		chainSelector := network.ChainSelector
		if chainSelectorsToLoad != nil && !slices.Contains(chainSelectorsToLoad, chainSelector) {
			lggr.Debugw("Excluding chain with selector from environment, not in the provided chain selectors to load", "chainSelector", chainSelector)
			continue
		}
		chainIDStr, errChainID := network.ChainID()
		if errChainID != nil {
			return nil, fmt.Errorf("no chain ID exists for chain selector %d: %w", chainSelector, errChainID)
		}
		chainID, errParse := strconv.ParseUint(chainIDStr, 10, 64)
		if errParse != nil {
			return nil, fmt.Errorf("failed to convert chain ID %s to uint64: %w", chainIDStr, errParse)
		}

		// Extract the anvil metadata from the network
		if network.Metadata == nil {
			ports, errPort := freeport.Take(1)
			if errPort != nil {
				// Fallback (very unlikely to be hit)
				ports = []int{8546}
			}
			if len(ports) == 0 {
				return nil, fmt.Errorf("no free ports available for chain selector %d", chainSelector)
			}
			network.Metadata = cfgnet.EVMMetadata{
				AnvilConfig: &cfgnet.AnvilConfig{
					Image: "f4hrenh9it/foundry:latest",
					Port:  uint64(ports[0]), //nolint:gosec // G115: int to uint64 conversion is safe here (port numbers are always in valid range)
				},
			}
		}

		metadata, errMeta := cfgnet.DecodeMetadata[cfgnet.EVMMetadata](network.Metadata)
		if errMeta != nil {
			return nil, fmt.Errorf(
				"failed to decode network metadata for chain selector %d: %w", chainSelector, errMeta,
			)
		}
		forkURLs, err := selectPublicRPC(ctx, lggr, &metadata, network.ChainSelector, network.RPCs)
		if err != nil {
			lggr.Infof("Excluding chain with ID %d from environment: %s", chainID, err.Error())
			continue
		}
		if err = metadata.AnvilConfig.Validate(); err != nil {
			lggr.Infof("Excluding chain with ID %d from environment due to failed anvil config validation: %s", chainID, err.Error())
			continue
		}

		// Skip chains that are not included in the address book
		if _, ok := addressesByChain[chainSelector]; !ok {
			lggr.Infof("Excluding chain with selector %d from environment, does not have addresses defined in the address book", chainSelector)

			continue
		}

		var signerGenerator evmprov.SignerGenerator
		if kmsConfig.KeyID != "" {
			var terr error
			signerGenerator, terr = evmprov.TransactorFromKMS(kmsConfig.KeyID, kmsConfig.KeyRegion, "")
			if terr != nil {
				return nil, fmt.Errorf("failed to create transactor from KMS: %w", terr)
			}
		} else {
			signerGenerator = evmprov.TransactorFromRaw(onchainConfig.EVM.DeployerKey)
		}

		if anvilKeyAsDeployer {
			// Set high gas limit to avoid using gas estimator
			// In fork tests the gas estimator can cause timeouts if txs have errors
			// occluding the real issue.
			signerGenerator = evmprov.TransactorFromRaw(
				blockchain.DefaultAnvilPrivateKey,
				evmprov.WithGasLimit(10_000_000),
			)
		}

		config := evmprov.CTFAnvilChainProviderConfig{
			Once:                     &once,
			ConfirmFunctor:           evmprov.ConfirmFuncGeth(3 * time.Minute),
			DockerCmdParamsOverrides: []string{"--auto-impersonate"},
			Image:                    metadata.AnvilConfig.Image,
			ForkURLs:                 forkURLs,
			DeployerTransactorGen:    signerGenerator,
			T:                        testing.TB(&testing.T{}),
			Port:                     "", // let the provider choose a free port; this ensures retries are handled properly
		}

		if blockNumber, ok := blockNumbers[chainSelector]; ok {
			config.DockerCmdParamsOverrides = append(config.DockerCmdParamsOverrides, "--fork-block-number", blockNumber.String())
		}

		provider := evmprov.NewCTFAnvilChainProvider(chainSelector, config)
		b, err := provider.Initialize(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize anvil chain provider for chain selector %d: %w", chainSelector, err)
		}

		blockChains = append(blockChains, b)
		anvilClients[chainSelector] = newAnvilForkClient(
			provider.GetNodeHTTPURL(),
			map[string]string{
				"Content-Type": "application/json",
			},
			&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // G402: TODO: Verify certificate? Though this will only be for testing so not sure if needed.
			},
			os.Getenv("RESTY_DEBUG") == "true",
		)

		chainConfigsBySelector[chainSelector] = ChainConfig{
			ChainID: chainIDStr,
			HTTPRPCs: []RPCs{
				{
					External: provider.GetNodeHTTPURL(),
				},
			},
		}
	}

	return &AnvilChainsOutput{
		Chains:       fchain.NewBlockChainsFromSlice(blockChains).EVMChains(),
		ForkClients:  anvilClients,
		ChainConfigs: chainConfigsBySelector,
	}, nil
}

func selectPublicRPC(
	ctx context.Context, lggr logger.Logger, metadata *cfgnet.EVMMetadata, chainSelector uint64, rpcs []cfgnet.RPC,
) ([]string, error) {
	if metadata.AnvilConfig.ArchiveHTTPURL != "" && isPublicRPC(metadata.AnvilConfig.ArchiveHTTPURL) {
		return []string{metadata.AnvilConfig.ArchiveHTTPURL}, nil
	}

	urls := []string{}
	for _, rpc := range rpcs {
		if isPublicRPC(rpc.HTTPURL) {
			err := runHealthCheck(ctx, rpc.HTTPURL)
			if err != nil {
				lggr.Infow("rpc failed health check", "url", rpc.HTTPURL, "chainSelector", chainSelector)
			} else {
				lggr.Infow("selected rpc for fork environment", "url", rpc.HTTPURL, "chainSelector", chainSelector)
				urls = append(urls, rpc.HTTPURL)
			}
		}
	}

	if len(urls) == 0 {
		return []string{}, fmt.Errorf("no public RPCs found for chain %d", chainSelector)
	}

	return urls, nil
}

var privateRpcRegexp = regexp.MustCompile(`^https?://(rpcs\.cldev\.sh|gap\-.*\.(prod|stage)\.cldev\.sh|.*\.tail[a-z0-9]+\.ts\.net)(?::\d+)?/`)

func isPublicRPC(url string) bool {
	return !privateRpcRegexp.MatchString(url)
}

func runHealthCheck(ctx context.Context, rpcURL string) error {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to rpc %v: %w", rpcURL, err)
	}

	_, err = client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve block number: %w", err)
	}

	return nil
}
