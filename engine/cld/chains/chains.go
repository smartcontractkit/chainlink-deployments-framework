package chains

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cldf_chain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_aptos_provider "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/provider"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldf_evm_provider "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	cldf_solana_provider "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider"
	cldf_tron_provider "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cldf_environment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// LoadChains concurrently loads all chains for the given environment. Each chain is loaded in parallel, and the results
// (including any errors) are collected for each chain. If any chains fail to load, the function aggregates the errors
// and returns a detailed error message specifying which chains failed and the reasons for failure.
func LoadChains(
	ctx context.Context,
	lggr logger.Logger,
	config *cldf_environment.Config,
	chainSelectorsToLoad []uint64,
) (cldf_chain.BlockChains, error) {
	chainLoaders := newChainLoaders(lggr, config.Networks, config.Env.Onchain)

	// Define a result struct to hold chain loading results
	type chainResult struct {
		chain    cldf_chain.BlockChain
		selector uint64
		family   string
		err      error
	}

	// Filter selectors that can actually be loaded
	validSelectors := make([]struct {
		selector uint64
		family   string
		loader   ChainLoader
	}, 0)

	for _, selector := range chainSelectorsToLoad {
		// Get the chain family for this selector
		chainFamily, err := chainsel.GetSelectorFamily(selector)
		if err != nil {
			lggr.Warnw("Unable to get chain family for selector",
				"selector", selector, "error", err,
			)

			return cldf_chain.BlockChains{}, fmt.Errorf("unable to get chain family for selector %d", selector)
		}

		// Check if we have a loader for this chain family
		loader, exists := chainLoaders[chainFamily]
		if !exists {
			lggr.Warnw("No chain loader available for chain family, skipping",
				"selector", selector, "family", chainFamily,
			)

			continue
		}

		validSelectors = append(validSelectors, struct {
			selector uint64
			family   string
			loader   ChainLoader
		}{selector, chainFamily, loader})
	}

	// Use indexed assignment to collect results (no mutex needed)
	results := make([]chainResult, len(validSelectors))

	// Use sync.WaitGroup for graceful collection
	var wg sync.WaitGroup

	// Load chains concurrently
	for i, vs := range validSelectors {
		wg.Add(1)

		go func(index int, selector uint64, family string, loader ChainLoader) {
			defer wg.Done()

			lggr.Infow("Loading chain", "selector", selector, "family", family)

			result := chainResult{
				selector: selector,
				family:   family,
			}

			// Handle context cancellation
			select {
			case <-ctx.Done():
				lggr.Warnw("Chain loading cancelled due to context cancellation",
					"selector", selector, "family", family)
				result.err = ctx.Err()
			default:
				// Load the chain
				chain, err := loader.Load(ctx, selector)
				result.chain = chain
				result.err = err
			}

			// Write result directly to assigned index (no mutex needed)
			results[index] = result
		}(i, vs.selector, vs.family, vs.loader)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Process all collected results
	loadedChains := make([]cldf_chain.BlockChain, 0)
	failedChains := make([]string, 0)

	for _, result := range results {
		if result.err != nil {
			lggr.Errorw("Failed to load chain",
				"selector", result.selector,
				"family", result.family,
				"error", result.err,
			)

			failedChains = append(failedChains, fmt.Sprintf("chain %d (%s): %v", result.selector, result.family, result.err))

			continue
		}

		loadedChains = append(loadedChains, result.chain)
	}

	// If any chains failed to load, return an error
	if len(failedChains) > 0 {
		return cldf_chain.BlockChains{}, fmt.Errorf("failed to load %d out of %d chains: %v",
			len(failedChains), len(validSelectors), failedChains)
	}

	lggr.Infow("Successfully loaded all chains",
		"total", len(chainSelectorsToLoad),
		"valid", len(validSelectors),
		"successful", len(loadedChains),
	)

	return cldf_chain.NewBlockChainsFromSlice(loadedChains), nil
}

// newChainLoaders returns a map of chain loaders for each supported chain family, based on the provided
// network config and secrets. Only chain loaders for which all required secrets are present will be created;
// if any required secret is missing for a chain family, its loader is omitted and a warning is logged.
// This ensures that only properly configured chains are attempted to be loaded, preventing runtime errors
// due to missing credentials or configuration.
func newChainLoaders(
	lggr logger.Logger, networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig,
) map[string]ChainLoader {
	// EVM chains are always loaded.
	loaders := map[string]ChainLoader{
		chainsel.FamilyEVM:  newChainLoaderEVM(networks, cfg, lggr),
		chainsel.FamilyTron: newChainLoaderTron(networks, cfg),
	}

	if cfg.Solana.ProgramsDirPath != "" && cfg.Solana.WalletKey != "" {
		loaders[chainsel.FamilySolana] = newChainLoaderSolana(networks, cfg)
	} else {
		lggr.Warn("Skipping Solana chains, no private key or program path found in secrets")
	}

	if cfg.Aptos.DeployerKey != "" {
		loaders[chainsel.FamilyAptos] = newChainLoaderAptos(networks, cfg)
	} else {
		lggr.Warn("Skipping Aptos chains, no private key found in secrets")
	}

	return loaders
}

var (
	_ ChainLoader = &chainLoaderAptos{}
	_ ChainLoader = &chainLoaderSolana{}
	_ ChainLoader = &chainLoaderEVM{}
	_ ChainLoader = &chainLoaderTron{}
)

// ChainLoader is an interface that defines the methods for loading a chain.
type ChainLoader interface {
	Load(ctx context.Context, selector uint64) (cldf_chain.BlockChain, error)
}

// baseChainLoader is a base implementation of the ChainLoader interface. It contains the common
// fields for all chain loaders.
type baseChainLoader struct {
	networks *cldf_config_network.Config
	cfg      cldf_config_env.OnchainConfig
}

// newBaseChainLoader creates a new base chain loader.
func newBaseChainLoader(
	networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig,
) *baseChainLoader {
	return &baseChainLoader{
		networks: networks,
		cfg:      cfg,
	}
}

// getNetwork gets the network for a given selector.
func (l *baseChainLoader) getNetwork(selector uint64) (cldf_config_network.Network, error) {
	network, err := l.networks.NetworkBySelector(selector)
	if err != nil {
		return cldf_config_network.Network{}, err
	}
	if len(network.RPCs) == 0 {
		return cldf_config_network.Network{}, fmt.Errorf("no RPCs found for chain selector: %d", selector)
	}

	return network, nil
}

// chainLoaderAptos implements the ChainLoader interface for Aptos.
type chainLoaderAptos struct {
	*baseChainLoader
}

// newChainLoaderAptos creates a new chain loader for Aptos.
func newChainLoaderAptos(
	networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig,
) *chainLoaderAptos {
	return &chainLoaderAptos{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads an Aptos Chain for a selector.
func (l *chainLoaderAptos) Load(ctx context.Context, selector uint64) (cldf_chain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	rpcURL := network.RPCs[0].HTTPURL
	c, err := cldf_aptos_provider.NewRPCChainProvider(selector,
		cldf_aptos_provider.RPCChainProviderConfig{
			RPCURL:            rpcURL,
			DeployerSignerGen: cldf_aptos_provider.AccountGenPrivateKey(l.cfg.Aptos.DeployerKey),
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Aptos chain %d: %w", selector, err)
	}

	return c, nil
}

// chainLoaderSolana implements the ChainLoader interface for Solana.
type chainLoaderSolana struct {
	*baseChainLoader
}

// newChainLoaderSolana a new chain loader for Solana.
func newChainLoaderSolana(
	networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig,
) *chainLoaderSolana {
	return &chainLoaderSolana{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Solana Chain for a selector.
func (l *chainLoaderSolana) Load(ctx context.Context, selector uint64) (cldf_chain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	// Load the path to the Solana programs from secrets
	programsPath, err := filepath.Abs(l.cfg.Solana.ProgramsDirPath)
	if err != nil {
		return nil, err
	}

	httpURL := network.RPCs[0].HTTPURL
	wsURL := network.RPCs[0].WSURL

	c, err := cldf_solana_provider.NewRPCChainProvider(selector,
		cldf_solana_provider.RPCChainProviderConfig{
			HTTPURL:        httpURL,
			WSURL:          wsURL,
			DeployerKeyGen: cldf_solana_provider.PrivateKeyFromRaw(l.cfg.Solana.WalletKey),
			ProgramsPath:   programsPath,
			KeypairDirPath: programsPath, // Use the same path for keypair storage
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Solana chain %d: %w", selector, err)
	}

	return c, nil
}

// chainLoaderEVM implements the ChainLoader interface for EVM.
type chainLoaderEVM struct {
	*baseChainLoader

	lggr logger.Logger
}

// newChainLoaderEVM creates a new chain loader for EVM.
func newChainLoaderEVM(
	networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig, lggr logger.Logger,
) *chainLoaderEVM {
	return &chainLoaderEVM{
		baseChainLoader: newBaseChainLoader(networks, cfg),
		lggr:            lggr,
	}
}

// Load loads an EVM Chain for a selector. It supports both regular EVM chains and zkSync flavored EVM chains.
func (l *chainLoaderEVM) Load(ctx context.Context, selector uint64) (cldf_chain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	// Load the RPCs for the chain
	rpcs, err := l.toRPCs(network.RPCs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RPCs for chain selector %d: %w", selector, err)
	}

	// Create a transactor generator based on whether we are using KMS or not.
	transactorGen, err := l.evmSignerGenerator(l.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create EVM signer generator: %w", err)
	}

	// Define the confirm function to use for transaction confirmation.
	confirmFunctor := l.confirmFunctor(network, l.cfg.EVM.Seth)

	// Define the client options to use for the MultiClient.
	clientOpts := []func(client *cldf.MultiClient){
		func(client *cldf.MultiClient) {
			client.RetryConfig = cldf.RetryConfig{
				Attempts:     5,                     // assuming failure rate is 20%, this will take 5 attempts to succeed
				Delay:        10 * time.Millisecond, // this is a very short delay, we want to be fast in this case
				Timeout:      5 * time.Second,
				DialAttempts: 5,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  2 * time.Second,
			}
		},
	}

	var c cldf_chain.BlockChain

	// Use the zkSync RPC if the chain is a zkSync chain.
	//
	// This is a temporary solution until we have a more generic way to identify zkSync chains in the
	// network config.
	if l.isZkSyncVM(selector) {
		var signerGen cldf_evm_provider.ZkSyncSignerGenerator
		signerGen, err = l.zkSyncSignerGenerator(l.cfg)
		if err != nil {
			return cldf_evm.Chain{}, fmt.Errorf("failed to create ZkSync signer generator: %w", err)
		}

		c, err = cldf_evm_provider.NewZkSyncRPCChainProvider(selector,
			cldf_evm_provider.ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: transactorGen,
				ZkSyncSignerGen:       signerGen,
				RPCs:                  rpcs,
				ConfirmFunctor:        confirmFunctor,
				ClientOpts:            clientOpts,
				Logger:                l.lggr,
			},
		).Initialize(ctx)
	} else {
		c, err = cldf_evm_provider.NewRPCChainProvider(selector,
			cldf_evm_provider.RPCChainProviderConfig{
				DeployerTransactorGen: transactorGen,
				RPCs:                  rpcs,
				ConfirmFunctor:        confirmFunctor,
				ClientOpts:            clientOpts,
				Logger:                l.lggr,
			},
		).Initialize(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chain %d: %w", network.ChainSelector, err)
	}

	return c, nil
}

// isZkSyncVM checks if the given chain selector corresponds to a zkSyncchain.
func (l *chainLoaderEVM) isZkSyncVM(selector uint64) bool {
	var zkSyncChainSelectors = []uint64{
		chainsel.ETHEREUM_TESTNET_SEPOLIA_ZKSYNC_1.Selector,
		chainsel.ETHEREUM_MAINNET_ZKSYNC_1.Selector,
		chainsel.LENS_MAINNET.Selector,
		chainsel.ETHEREUM_TESTNET_SEPOLIA_LENS_1.Selector,
		chainsel.CRONOS_ZKEVM_MAINNET.Selector,
		chainsel.CRONOS_ZKEVM_TESTNET_SEPOLIA.Selector,
	}

	return slices.Contains(zkSyncChainSelectors, selector)
}

// toRPCs converts a network to a slice of RPCs for a specific chain ID.
func (l *chainLoaderEVM) toRPCs(rpcCfgs []cldf_config_network.RPC) ([]cldf.RPC, error) {
	rpcs := make([]cldf.RPC, 0, len(rpcCfgs))

	for _, rpcCfg := range rpcCfgs {
		preferedUrlScheme, err := cldf.URLSchemePreferenceFromString(rpcCfg.PreferredURLScheme)
		if err != nil {
			return nil, fmt.Errorf("invalid URL scheme preference %s: %w",
				rpcCfg.PreferredURLScheme, err,
			)
		}

		rpcs = append(rpcs, cldf.RPC{
			Name:               rpcCfg.RPCName,
			WSURL:              rpcCfg.WSURL,
			HTTPURL:            rpcCfg.HTTPURL,
			PreferredURLScheme: preferedUrlScheme,
		})
	}

	return rpcs, nil
}

// evmSignerGenerator creates a transactor generator for an EVM chain.
func (l *chainLoaderEVM) evmSignerGenerator(
	cfg cldf_config_env.OnchainConfig,
) (cldf_evm_provider.SignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return cldf_evm_provider.TransactorFromKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return cldf_evm_provider.TransactorFromRaw(cfg.EVM.DeployerKey), nil
}

// confirmFunctor generates a confirm function for the EVM chain. It prefers to use Seth's confirm
// function, but falls back to Geth's confirm function if Seth config is not provided, or there
// are no wrappers provided.
func (l *chainLoaderEVM) confirmFunctor(
	network cldf_config_network.Network, sethCfg *cldf_config_env.SethConfig,
) cldf_evm_provider.ConfirmFunctor {
	if sethCfg == nil || len(sethCfg.GethWrapperDirs) == 0 {
		l.lggr.Infow("No Seth config provided, using Geth's confirm function",
			"chain_selector", network.ChainSelector,
		)

		return cldf_evm_provider.ConfirmFuncGeth(10 * time.Minute)
	}

	// Define the confirm function to use for transaction confirmation.
	return cldf_evm_provider.ConfirmFuncSeth(
		network.RPCs[0].PreferredEndpoint(),
		10*time.Minute,
		l.cfg.EVM.Seth.GethWrapperDirs,
		l.cfg.EVM.Seth.ConfigFilePath,
	)
}

// zkSyncSignerGenerator creates a ZkSync signer generator for a zkSync chain.
func (l *chainLoaderEVM) zkSyncSignerGenerator(
	cfg cldf_config_env.OnchainConfig,
) (cldf_evm_provider.ZkSyncSignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return cldf_evm_provider.ZkSyncSignerFromKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return cldf_evm_provider.ZkSyncSignerFromRaw(cfg.EVM.DeployerKey), nil
}

// chainLoaderTron implements the ChainLoader interface for Tron.
type chainLoaderTron struct {
	*baseChainLoader
}

// newChainLoaderTron a new chain loader for Tron.
func newChainLoaderTron(
	networks *cldf_config_network.Config, cfg cldf_config_env.OnchainConfig,
) *chainLoaderTron {
	return &chainLoaderTron{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Tron Chain for a selector.
func (l *chainLoaderTron) Load(ctx context.Context, selector uint64) (cldf_chain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	fullNodeURL := network.RPCs[0].HTTPURL + "/wallet"
	solidityNodeURL := network.RPCs[0].HTTPURL + "/walletsolidity"

	generator, err := l.tronSignerGenerator(l.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TRON account generator: %w", err)
	}

	c, err := cldf_tron_provider.NewRPCChainProvider(selector,
		cldf_tron_provider.RPCChainProviderConfig{
			FullNodeURL:       fullNodeURL,
			SolidityNodeURL:   solidityNodeURL,
			DeployerSignerGen: generator,
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Tron chain %d: %w", selector, err)
	}

	return c, nil
}

// tronSignerGenerator creates a transactor generator for an TRON chain.
func (l *chainLoaderTron) tronSignerGenerator(
	cfg cldf_config_env.OnchainConfig,
) (cldf_tron_provider.SignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return cldf_tron_provider.SignerGenKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return cldf_tron_provider.SignerGenPrivateKey(cfg.Tron.DeployerKey)
}

// useKMS returns true if both KeyID and KeyRegion are set in the provided KMS config.
func useKMS(kmsCfg cldf_config_env.KMSConfig) bool {
	return kmsCfg.KeyID != "" && kmsCfg.KeyRegion != ""
}
