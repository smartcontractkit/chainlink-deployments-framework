package environment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	chainsel "github.com/smartcontractkit/chain-selectors"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
)

// ForkedOnchainClient is a client for a fork of a blockchain node.
// It should be able to send transactions from any address without the need for a private key.
type ForkedOnchainClient interface {
	// SendTransaction sends transaction data from one address to another.
	// Implementations should ensure that the transaction doesn't need a valid signature to be accepted.
	SendTransaction(ctx context.Context, from string, to string, data []byte) error
}

// ForkedEnvironment represents a forked deployment environment.
// It embeds a standard environment with the addition of a client for forking per fchain.
type ForkedEnvironment struct {
	fdeployment.Environment
	ChainConfigs map[uint64]ChainConfig
	ForkClients  map[uint64]ForkedOnchainClient
}

// LoadFork loads a deployment environment in which the chains are forks of real networks.
// Provides access to a forking client per chain that allows users to send transactions without signatures.
//
// Limitations:
// - EVM only
func LoadFork(
	ctx context.Context,
	domain fdomain.Domain,
	env string,
	blockNumbers map[uint64]*big.Int,
	opts ...LoadEnvironmentOption,
) (ForkedEnvironment, error) {
	loadcfg, err := newLoadConfig()
	if err != nil {
		return ForkedEnvironment{}, err
	}
	loadcfg.Configure(opts)

	lggr := loadcfg.lggr

	cfg, err := config.Load(domain, env, lggr)
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Limit to EVM networks only
	networks := cfg.Networks.FilterWith(
		cfgnet.ChainFamilyFilter(chainsel.FamilyEVM),
	)

	addressBook, err := domain.AddressBookByEnv(env)
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load address book for domain %s and environment %s: %w", domain.Key(), env, err)
	}

	// TODO: once all products are on the new datastore, we can remove this default
	dataStore := fdatastore.NewMemoryDataStore().Seal()
	envDataStore, err := domain.DataStoreByEnv(env)
	if err == nil {
		dataStore = envDataStore
	} else {
		lggr.Warnf("failed to load data store for domain %s and environment %s: %w", domain.Key(), env, err)
	}

	anvilOutput, err := newAnvilChains(
		ctx,
		lggr,
		addressBook,
		dataStore,
		networks,
		blockNumbers,
		cfg.Env.Onchain,
		cfg.Env.Onchain.KMS,
		loadcfg.chainSelectorsToLoad,
		loadcfg.anvilKeyAsDeployer,
	)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			err = errors.Join(errors.New("check your VPN connection"), err)
		}

		return ForkedEnvironment{}, fmt.Errorf("failed to create anvil chains: %w", err)
	}

	envdir := domain.EnvDir(env)
	nodes, err := envdir.LoadNodes()
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load nodes: %w", err)
	}

	config, err := config.LoadEnvConfig(domain, env)
	if err != nil {
		return ForkedEnvironment{}, err
	}

	var oc foffchain.Client

	if !loadcfg.withoutJD {
		oc, err = offchain.LoadOffchainClient(ctx, domain, config.Offchain.JobDistributor,
			offchain.WithLogger(lggr),
			offchain.WithDryRun(true),
			offchain.WithCredentials(credentials.GetCredsForEnv(env)),
		)
		if err != nil {
			if errors.Is(err, offchain.ErrEndpointsRequired) {
				lggr.Warn("Skipping JD initialization: gRPC and wsRPC endpoints are not set in config")
			} else {
				return ForkedEnvironment{}, fmt.Errorf("failed to load offchain client: %w", err)
			}

			return ForkedEnvironment{}, fmt.Errorf("failed to load offchain client: %w", err)
		}
	} else {
		lggr.Info("Override: skipping JD initialization")
	}

	blockChains := map[uint64]fchain.BlockChain{}
	for selector, ch := range anvilOutput.Chains {
		blockChains[selector] = ch
	}

	// TODO: newSolChains, newAptosChains, etc.
	environment := fdeployment.NewEnvironment(
		"fork",
		lggr,
		addressBook,
		dataStore,
		nodes.Keys(),
		oc,
		func() context.Context { return ctx },
		focr.XXXGenerateTestOCRSecrets(),
		fchain.NewBlockChains(blockChains),
	)

	return ForkedEnvironment{
		Environment:  *environment,
		ChainConfigs: anvilOutput.ChainConfigs,
		ForkClients:  anvilOutput.ForkClients, // TODO: map should eventually include clients from other families
	}, nil
}

// ApplyChangesetOutput executes MCMS proposals and merges addresses into the address book.
func (e ForkedEnvironment) ApplyChangesetOutput(ctx context.Context, output fdeployment.ChangesetOutput) (ForkedEnvironment, error) {
	// TODO: Applying jobs? How would this work in a forked environment?

	// Apply mcms proposals that forego timelock usage
	for _, proposal := range output.MCMSProposals {
		for _, operation := range proposal.Operations {
			chainMetadata, ok := proposal.ChainMetadata[operation.ChainSelector]
			if !ok {
				return ForkedEnvironment{}, fmt.Errorf("no chain metadata defined for chain selector %d", operation.ChainSelector)
			}
			forkClient, ok := e.ForkClients[uint64(operation.ChainSelector)]
			if !ok {
				return ForkedEnvironment{}, fmt.Errorf("no fork client defined for chain selector %d", operation.ChainSelector)
			}
			err := forkClient.SendTransaction(ctx, chainMetadata.MCMAddress, operation.Transaction.To, operation.Transaction.Data)
			if err != nil {
				return ForkedEnvironment{}, fmt.Errorf("failed to send transaction on chain with selector %d: %w", operation.ChainSelector, err)
			}
		}
	}

	// Apply timelock proposals
	for _, proposal := range output.MCMSTimelockProposals {
		for _, operation := range proposal.Operations {
			timelockAddress, ok := proposal.TimelockAddresses[operation.ChainSelector]
			if !ok {
				return ForkedEnvironment{}, fmt.Errorf("no timelock address defined for chain selector %d", operation.ChainSelector)
			}
			forkClient, ok := e.ForkClients[uint64(operation.ChainSelector)]
			if !ok {
				return ForkedEnvironment{}, fmt.Errorf("no fork client defined for chain selector %d", operation.ChainSelector)
			}
			for _, op := range operation.Transactions {
				err := forkClient.SendTransaction(ctx, timelockAddress, op.To, op.Data)
				if err != nil {
					return ForkedEnvironment{}, fmt.Errorf("failed to send transaction on chain with selector %d: %w", operation.ChainSelector, err)
				}
			}
		}
	}

	// Merge new addresses into address book
	if output.AddressBook != nil { // nolint
		err := e.ExistingAddresses.Merge(output.AddressBook) // nolint
		if err != nil {
			return ForkedEnvironment{}, fmt.Errorf("failed to merge new addresses into address book: %w", err)
		}
	}

	return e, nil
}
