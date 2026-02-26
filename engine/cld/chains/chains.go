package chains

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	aptosprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/provider"
	cantonprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider"
	cantonauth "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider/authentication"
	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
	solanaprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider"
	stellarprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar/provider"
	suiprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/provider"
	tonprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/provider"
	tronprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// LoadChains concurrently loads all chains for the given environment. Each chain is loaded in parallel, and the results
// (including any errors) are collected for each chain. If any chains fail to load, the function aggregates the errors
// and returns a detailed error message specifying which chains failed and the reasons for failure.
func LoadChains(
	ctx context.Context,
	lggr logger.Logger,
	cfg *config.Config,
	chainselToLoad []uint64,
) (fchain.BlockChains, error) {
	if len(chainselToLoad) == 0 {
		lggr.Info("No chain selectors provided, skipping chain loading")
		return fchain.NewBlockChains(map[uint64]fchain.BlockChain{}), nil
	}
	chainLoaders := newChainLoaders(lggr, cfg.Networks, cfg.Env.Onchain)

	// Define a result struct to hold chain loading results
	type chainResult struct {
		chain    fchain.BlockChain
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

	for _, selector := range chainselToLoad {
		// Get the chain family for this selector
		chainFamily, err := chainsel.GetSelectorFamily(selector)
		if err != nil {
			lggr.Warnw("Unable to get chain family for selector",
				"selector", selector, "error", err,
			)

			return fchain.BlockChains{}, fmt.Errorf("unable to get chain family for selector %d", selector)
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
	loadedChains := make([]fchain.BlockChain, 0)
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
		return fchain.BlockChains{}, fmt.Errorf("failed to load %d out of %d chains: %v",
			len(failedChains), len(validSelectors), failedChains)
	}

	lggr.Infow("Successfully loaded all chains",
		"total", len(chainselToLoad),
		"valid", len(validSelectors),
		"successful", len(loadedChains),
	)

	return fchain.NewBlockChainsFromSlice(loadedChains), nil
}

// newChainLoaders returns a map of chain loaders for each supported chain family, based on the provided
// network config and secrets. Only chain loaders for which all required secrets are present will be created;
// if any required secret is missing for a chain family, its loader is omitted and a warning is logged.
// This ensures that only properly configured chains are attempted to be loaded, preventing runtime errors
// due to missing credentials or configuration.
func newChainLoaders(
	lggr logger.Logger, networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) map[string]ChainLoader {
	loaders := map[string]ChainLoader{}

	// EVM chains are loaded if either KMS or deployer key is configured.
	if useKMS(cfg.KMS) || cfg.EVM.DeployerKey != "" {
		loaders[chainsel.FamilyEVM] = newChainLoaderEVM(networks, cfg, lggr)
	} else {
		lggr.Info("Skipping EVM chains, no private key or KMS config found in secrets")
	}

	// Tron chains are loaded if either KMS or deployer key is configured.
	if useKMS(cfg.KMS) || cfg.Tron.DeployerKey != "" {
		loaders[chainsel.FamilyTron] = newChainLoaderTron(networks, cfg)
	} else {
		lggr.Info("Skipping Tron chains, no private key or KMS config found in secrets")
	}

	if cfg.Solana.ProgramsDirPath != "" && cfg.Solana.WalletKey != "" {
		loaders[chainsel.FamilySolana] = newChainLoaderSolana(networks, cfg)
	} else {
		lggr.Info("Skipping Solana chains, no private key or program path found in secrets")
	}

	if cfg.Aptos.DeployerKey != "" {
		loaders[chainsel.FamilyAptos] = newChainLoaderAptos(networks, cfg)
	} else {
		lggr.Info("Skipping Aptos chains, no private key found in secrets")
	}

	if cfg.Sui.DeployerKey != "" {
		loaders[chainsel.FamilySui] = newChainLoaderSui(networks, cfg)
	} else {
		lggr.Info("Skipping Sui chains, no private key found in secrets")
	}

	if cfg.Stellar.DeployerKey != "" {
		loaders[chainsel.FamilyStellar] = newChainLoaderStellar(networks, cfg)
	} else {
		lggr.Info("Skipping Stellar chains, no private key found in secrets")
	}

	if cfg.Ton.DeployerKey != "" {
		loaders[chainsel.FamilyTon] = newChainLoaderTon(networks, cfg)
	} else {
		lggr.Info("Skipping Ton chains, no private key found in secrets")
	}

	if cantonAuthConfigured(cfg.Canton) {
		loaders[chainsel.FamilyCanton] = newChainLoaderCanton(networks, cfg)
	} else {
		lggr.Info("Skipping Canton chains, no Canton auth configured (set auth_type and jwt_token, or auth_url+client_id for OAuth)")
	}

	return loaders
}

var (
	_ ChainLoader = &chainLoaderAptos{}
	_ ChainLoader = &chainLoaderSolana{}
	_ ChainLoader = &chainLoaderEVM{}
	_ ChainLoader = &chainLoaderTron{}
	_ ChainLoader = &chainLoaderSui{}
	_ ChainLoader = &chainLoaderStellar{}
	_ ChainLoader = &chainLoaderTon{}
	_ ChainLoader = &chainLoaderCanton{}
)

// ChainLoader is an interface that defines the methods for loading a chain.
type ChainLoader interface {
	Load(ctx context.Context, selector uint64) (fchain.BlockChain, error)
}

// baseChainLoader is a base implementation of the ChainLoader interface. It contains the common
// fields for all chain loaders.
type baseChainLoader struct {
	networks *cfgnet.Config
	cfg      cfgenv.OnchainConfig
}

// newBaseChainLoader creates a new base chain loader.
func newBaseChainLoader(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *baseChainLoader {
	return &baseChainLoader{
		networks: networks,
		cfg:      cfg,
	}
}

// getNetwork gets the network for a given selector.
func (l *baseChainLoader) getNetwork(selector uint64) (cfgnet.Network, error) {
	network, err := l.networks.NetworkBySelector(selector)
	if err != nil {
		return cfgnet.Network{}, err
	}
	if len(network.RPCs) == 0 {
		return cfgnet.Network{}, fmt.Errorf("no RPCs found for chain selector: %d", selector)
	}

	return network, nil
}

// chainLoaderAptos implements the ChainLoader interface for Aptos.
type chainLoaderAptos struct {
	*baseChainLoader
}

// newChainLoaderAptos creates a new chain loader for Aptos.
func newChainLoaderAptos(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderAptos {
	return &chainLoaderAptos{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads an Aptos Chain for a selector.
func (l *chainLoaderAptos) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	rpcURL := network.RPCs[0].HTTPURL
	c, err := aptosprov.NewRPCChainProvider(selector,
		aptosprov.RPCChainProviderConfig{
			RPCURL:            rpcURL,
			DeployerSignerGen: aptosprov.AccountGenPrivateKey(l.cfg.Aptos.DeployerKey),
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Aptos chain %d: %w", selector, err)
	}

	return c, nil
}

// chainLoaderSui implements the ChainLoader interface for Sui.
type chainLoaderSui struct {
	*baseChainLoader
}

// newChainLoaderSui creates a new chain loader for Sui.
func newChainLoaderSui(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderSui {
	return &chainLoaderSui{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Sui Chain for a selector.
func (l *chainLoaderSui) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	rpcURL := network.RPCs[0].HTTPURL
	c, err := suiprov.NewRPCChainProvider(selector,
		suiprov.RPCChainProviderConfig{
			RPCURL:            rpcURL,
			DeployerSignerGen: suiprov.AccountGenPrivateKey(l.cfg.Sui.DeployerKey),
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sui chain %d: %w", selector, err)
	}

	return c, nil
}

// chainLoaderStellar implements the ChainLoader interface for Stellar.
type chainLoaderStellar struct {
	*baseChainLoader
}

// newChainLoaderStellar creates a new chain loader for Stellar.
func newChainLoaderStellar(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderStellar {
	return &chainLoaderStellar{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Stellar Chain for a selector.
// RPC URL (Soroban) comes from network.RPCs like other chains; passphrase and Friendbot URL from metadata.
func (l *chainLoaderStellar) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	rpcURL := network.RPCs[0].HTTPURL
	if rpcURL == "" {
		return nil, fmt.Errorf("stellar network %d: RPC http_url is required", selector)
	}

	md, err := cfgnet.DecodeMetadata[cfgnet.StellarMetadata](network.Metadata)
	if err != nil {
		return nil, fmt.Errorf("stellar network %d: decode metadata: %w", selector, err)
	}

	c, err := stellarprov.NewRPCChainProvider(selector,
		stellarprov.RPCChainProviderConfig{
			NetworkPassphrase:  md.NetworkPassphrase,
			FriendbotURL:       md.FriendbotURL,
			SorobanRPCURL:      rpcURL,
			DeployerKeypairGen: stellarprov.KeypairFromHex(l.cfg.Stellar.DeployerKey),
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Stellar chain %d: %w", selector, err)
	}

	return c, nil
}

// chainLoaderSolana implements the ChainLoader interface for Solana.
type chainLoaderSolana struct {
	*baseChainLoader
}

// newChainLoaderSolana a new chain loader for Solana.
func newChainLoaderSolana(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderSolana {
	return &chainLoaderSolana{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Solana Chain for a selector.
func (l *chainLoaderSolana) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
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

	c, err := solanaprov.NewRPCChainProvider(selector,
		solanaprov.RPCChainProviderConfig{
			HTTPURL:        httpURL,
			WSURL:          wsURL,
			DeployerKeyGen: solanaprov.PrivateKeyFromRaw(l.cfg.Solana.WalletKey),
			ProgramsPath:   programsPath,
			KeypairDirPath: programsPath, // Use the same path for keypair storage
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Solana chain %d: %w", selector, err)
	}

	return c, nil
}

type chainLoaderTon struct {
	*baseChainLoader
}

// newChainLoaderTon a new chain loader for Ton.
func newChainLoaderTon(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderTon {
	return &chainLoaderTon{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Ton Chain for a selector.
func (l *chainLoaderTon) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	httpURL := network.RPCs[0].HTTPURL
	wsURL := network.RPCs[0].WSURL

	c, err := tonprov.NewRPCChainProvider(selector,
		tonprov.RPCChainProviderConfig{
			HTTPURL:           httpURL,
			WSURL:             wsURL,
			DeployerSignerGen: tonprov.PrivateKeyFromRaw(l.cfg.Ton.DeployerKey),
			WalletVersion:     tonprov.WalletVersion(l.cfg.Ton.WalletVersion),
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TON chain %d: %w", selector, err)
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
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig, lggr logger.Logger,
) *chainLoaderEVM {
	return &chainLoaderEVM{
		baseChainLoader: newBaseChainLoader(networks, cfg),
		lggr:            lggr,
	}
}

// Load loads an EVM Chain for a selector. It supports both regular EVM chains and zkSync flavored EVM chains.
func (l *chainLoaderEVM) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
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
	clientOpts := []func(client *evmclient.MultiClient){
		func(client *evmclient.MultiClient) {
			client.RetryConfig = evmclient.RetryConfig{
				Attempts:     5,                     // assuming failure rate is 20%, this will take 5 attempts to succeed
				Delay:        10 * time.Millisecond, // this is a very short delay, we want to be fast in this case
				Timeout:      5 * time.Second,
				DialAttempts: 5,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  2 * time.Second,
			}
		},
	}

	var c fchain.BlockChain

	// Use the zkSync RPC if the chain is a zkSync chain.
	//
	// This is a temporary solution until we have a more generic way to identify zkSync chains in the
	// network config.
	if l.isZkSyncVM(selector) {
		var signerGen evmprov.ZkSyncSignerGenerator
		signerGen, err = l.zkSyncSignerGenerator(l.cfg)
		if err != nil {
			return fevm.Chain{}, fmt.Errorf("failed to create ZkSync signer generator: %w", err)
		}

		c, err = evmprov.NewZkSyncRPCChainProvider(selector,
			evmprov.ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: transactorGen,
				ZkSyncSignerGen:       signerGen,
				RPCs:                  rpcs,
				ConfirmFunctor:        confirmFunctor,
				ClientOpts:            clientOpts,
				Logger:                l.lggr,
			},
		).Initialize(ctx)
	} else {
		c, err = evmprov.NewRPCChainProvider(selector,
			evmprov.RPCChainProviderConfig{
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
	var zkSyncchainsel = []uint64{
		chainsel.ETHEREUM_TESTNET_SEPOLIA_ZKSYNC_1.Selector,
		chainsel.ETHEREUM_MAINNET_ZKSYNC_1.Selector,
		chainsel.LENS_MAINNET.Selector,
		chainsel.ETHEREUM_TESTNET_SEPOLIA_LENS_1.Selector,
		chainsel.CRONOS_ZKEVM_MAINNET.Selector,
		chainsel.CRONOS_ZKEVM_TESTNET_SEPOLIA.Selector,
	}

	return slices.Contains(zkSyncchainsel, selector)
}

// toRPCs converts a network to a slice of RPCs for a specific chain ID.
func (l *chainLoaderEVM) toRPCs(rpcCfgs []cfgnet.RPC) ([]evmclient.RPC, error) {
	rpcs := make([]evmclient.RPC, 0, len(rpcCfgs))

	for _, rpcCfg := range rpcCfgs {
		preferedUrlScheme, err := evmclient.URLSchemePreferenceFromString(rpcCfg.PreferredURLScheme)
		if err != nil {
			return nil, fmt.Errorf("invalid URL scheme preference %s: %w",
				rpcCfg.PreferredURLScheme, err,
			)
		}

		rpcs = append(rpcs, evmclient.RPC{
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
	cfg cfgenv.OnchainConfig,
) (evmprov.SignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return evmprov.TransactorFromKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return evmprov.TransactorFromRaw(cfg.EVM.DeployerKey), nil
}

// confirmFunctor generates a confirm function for the EVM chain. It prefers to use Seth's confirm
// function, but falls back to Geth's confirm function if Seth config is not provided, or there
// are no wrappers provided.
func (l *chainLoaderEVM) confirmFunctor(
	network cfgnet.Network, sethCfg *cfgenv.SethConfig,
) evmprov.ConfirmFunctor {
	if sethCfg == nil || len(sethCfg.GethWrapperDirs) == 0 {
		l.lggr.Infow("No Seth config provided, using Geth's confirm function",
			"chain_selector", network.ChainSelector,
		)

		return evmprov.ConfirmFuncGeth(10 * time.Minute)
	}

	// Define the confirm function to use for transaction confirmation.
	return evmprov.ConfirmFuncSeth(
		network.RPCs[0].PreferredEndpoint(),
		10*time.Minute,
		l.cfg.EVM.Seth.GethWrapperDirs,
		l.cfg.EVM.Seth.ConfigFilePath,
	)
}

// zkSyncSignerGenerator creates a ZkSync signer generator for a zkSync chain.
func (l *chainLoaderEVM) zkSyncSignerGenerator(
	cfg cfgenv.OnchainConfig,
) (evmprov.ZkSyncSignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return evmprov.ZkSyncSignerFromKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return evmprov.ZkSyncSignerFromRaw(cfg.EVM.DeployerKey), nil
}

// chainLoaderTron implements the ChainLoader interface for Tron.
type chainLoaderTron struct {
	*baseChainLoader
}

// newChainLoaderTron a new chain loader for Tron.
func newChainLoaderTron(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderTron {
	return &chainLoaderTron{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Tron Chain for a selector.
func (l *chainLoaderTron) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
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

	c, err := tronprov.NewRPCChainProvider(selector,
		tronprov.RPCChainProviderConfig{
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
	cfg cfgenv.OnchainConfig,
) (tronprov.SignerGenerator, error) {
	if useKMS(cfg.KMS) {
		return tronprov.SignerGenKMS(
			cfg.KMS.KeyID,
			cfg.KMS.KeyRegion,
			"", // This is set to empty string as we don't have a profile name for the KMS config. This adheres to the existing behavior.
		)
	}

	return tronprov.SignerGenPrivateKey(cfg.Tron.DeployerKey)
}

// chainLoaderCanton implements the ChainLoader interface for Canton.
type chainLoaderCanton struct {
	*baseChainLoader
}

// newChainLoaderCanton creates a new chain loader for Canton.
func newChainLoaderCanton(
	networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
) *chainLoaderCanton {
	return &chainLoaderCanton{
		baseChainLoader: newBaseChainLoader(networks, cfg),
	}
}

// Load loads a Canton Chain for a selector.
// Participant configurations come from network metadata, and JWT token from env config.
func (l *chainLoaderCanton) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
	network, err := l.getNetwork(selector)
	if err != nil {
		return nil, err
	}

	// Decode Canton metadata to get participant configurations
	md, err := cfgnet.DecodeMetadata[cfgnet.CantonMetadata](network.Metadata)
	if err != nil {
		return nil, fmt.Errorf("canton network %d: decode metadata: %w", selector, err)
	}

	if len(md.Participants) == 0 {
		return nil, fmt.Errorf("canton network %d: no participants found in metadata", selector)
	}

	authProvider, err := l.cantonAuthProvider(ctx, selector)
	if err != nil {
		return nil, err
	}

	participants := make([]cantonprov.ParticipantConfig, len(md.Participants))
	for i, participantMD := range md.Participants {
		participants[i] = cantonprov.ParticipantConfig{
			JSONLedgerAPIURL: participantMD.JSONLedgerAPIURL,
			GRPCLedgerAPIURL: participantMD.GRPCLedgerAPIURL,
			AdminAPIURL:      participantMD.AdminAPIURL,
			ValidatorAPIURL:  participantMD.ValidatorAPIURL,
			UserID:           participantMD.UserID,
			PartyID:          participantMD.PartyID,
			AuthProvider:     authProvider,
		}
	}

	c, err := cantonprov.NewRPCChainProvider(selector,
		cantonprov.RPCChainProviderConfig{
			Participants: participants,
		},
	).Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Canton chain %d: %w", selector, err)
	}

	return c, nil
}

// cantonAuthConfigured returns true if Canton auth is configured for at least one scheme (static, client_credentials, or authorization_code).
func cantonAuthConfigured(c cfgenv.CantonConfig) bool {
	switch c.AuthType {
	case cfgenv.CantonAuthTypeClientCredentials:
		return c.AuthURL != "" && c.ClientID != "" && c.ClientSecret != ""
	case cfgenv.CantonAuthTypeAuthorizationCode:
		return c.AuthURL != "" && c.ClientID != ""
	default:
		// static or empty (backward compat: jwt_token alone enables Canton)
		return c.JWTToken != ""
	}
}

// cantonAuthProvider builds a Canton auth Provider from config. Caller must ensure cantonAuthConfigured(cfg.Canton) is true.
func (l *chainLoaderCanton) cantonAuthProvider(ctx context.Context, selector uint64) (cantonauth.Provider, error) {
	c := l.cfg.Canton
	switch c.AuthType {
	case cfgenv.CantonAuthTypeClientCredentials:
		if c.AuthURL == "" || c.ClientID == "" || c.ClientSecret == "" {
			return nil, fmt.Errorf("canton network %d: client_credentials requires auth_url, client_id, and client_secret", selector)
		}
		oidc, err := cantonauth.NewClientCredentialsProvider(ctx, c.AuthURL, c.ClientID, c.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("canton network %d: client_credentials auth: %w", selector, err)
		}

		return oidc, nil
	case cfgenv.CantonAuthTypeAuthorizationCode:
		if c.AuthURL == "" || c.ClientID == "" {
			return nil, fmt.Errorf("canton network %d: authorization_code requires auth_url and client_id", selector)
		}
		oidc, err := cantonauth.NewAuthorizationCodeProvider(ctx, c.AuthURL, c.ClientID)
		if err != nil {
			return nil, fmt.Errorf("canton network %d: authorization_code auth: %w", selector, err)
		}

		return oidc, nil
	default:
		// static or empty
		if c.JWTToken == "" {
			return nil, fmt.Errorf("canton network %d: JWT token is required for static auth", selector)
		}

		return cantonauth.NewStaticProvider(c.JWTToken), nil
	}
}

// useKMS returns true if both KeyID and KeyRegion are set in the provided KMS config.
func useKMS(kmsCfg cfgenv.KMSConfig) bool {
	return kmsCfg.KeyID != "" && kmsCfg.KeyRegion != ""
}
