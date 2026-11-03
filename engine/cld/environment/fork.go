package environment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/testcontainers/testcontainers-go"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

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
	Containers   map[uint64]testcontainers.Container
}

// LoadFork loads a deployment environment in which the chains are forks of real networks.
// Provides access to a forking client per chain that allows users to send transactions without signatures.
//
// Limitations:
// - EVM only
func LoadFork(
	ctx context.Context,
	domain fdomain.Domain,
	env fdeployment.Environment,
	networksCfg *cfgnet.Config,
	onchainCfg cfgenv.OnchainConfig,
	blockNumbers map[uint64]*big.Int,
	providerType ProviderType,
	opts ...LoadEnvironmentOption,
) (ForkedEnvironment, error) {
	loadcfg, err := newLoadConfig()
	if err != nil {
		return ForkedEnvironment{}, err
	}
	loadcfg.Configure(opts)

	networksCfg = networksCfg.FilterWith(
		cfgnet.ChainFamilyFilter(chainsel.FamilyEVM),
		func(n cfgnet.Network) bool {
			return loadcfg.chainSelectorsToLoad == nil || slices.Contains(loadcfg.chainSelectorsToLoad, n.ChainSelector)
		})

	anvilOutput, err := newAnvilChains(
		ctx,
		env.Logger,
		networksCfg,
		blockNumbers,
		onchainCfg,
		onchainCfg.KMS,
		networksCfg.ChainSelectors(),
		loadcfg.anvilKeyAsDeployer,
		providerType,
	)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			err = errors.Join(errors.New("check your VPN connection"), err)
		}

		return ForkedEnvironment{}, fmt.Errorf("failed to create anvil chains: %w", err)
	}

	// envdir := domain.EnvDir(env.Name)
	// nodes, err := envdir.LoadNodes()
	// if err != nil {
	// 	return ForkedEnvironment{}, fmt.Errorf("failed to load nodes: %w", err)
	// }

	// config, err := config.LoadEnvConfig(domain, env.Name)
	// if err != nil {
	// 	return ForkedEnvironment{}, fmt.Errorf("failed to load environment-specific config: %w", err)
	// }

	blockChains := map[uint64]fchain.BlockChain{}
	for selector, ch := range anvilOutput.Chains {
		blockChains[selector] = ch
	}

	// TODO: newSolChains, newAptosChains, etc.

	// addressBook, err := domain.AddressBookByEnv(env)
	// if err != nil {
	// 	return ForkedEnvironment{}, fmt.Errorf("failed to load addressbook for domain %s and environment %s: %w", domain.Key(), env, err)
	// }
	// dataStore, err := LoadDataStore(ctx, cfg, loadcfg, domain, env)
	// if err != nil {
	// 	return ForkedEnvironment{}, fmt.Errorf("failed to load datastore for domain %s and environment %s: %w", domain.Key(), env, err)
	// }
	environment := fdeployment.NewEnvironment(
		"fork",
		env.Logger,
		env.ExistingAddresses,
		env.DataStore,
		[]string{},
		foffchain.Client(nil),
		func() context.Context { return ctx },
		focr.XXXGenerateTestOCRSecrets(),
		fchain.NewBlockChains(blockChains),
		fdeployment.WithCRERunner(env.CRERunner),
	)

	return ForkedEnvironment{
		Environment:  *environment,
		ChainConfigs: anvilOutput.ChainConfigs,
		ForkClients:  anvilOutput.ForkClients, // TODO: map should eventually include clients from other families
		Containers:   anvilOutput.Containers,
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
