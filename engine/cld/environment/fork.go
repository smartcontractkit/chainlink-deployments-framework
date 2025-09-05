package environment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_config "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldf_engine_offchain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// ForkedOnchainClient is a client for a fork of a blockchain node.
// It should be able to send transactions from any address without the need for a private key.
type ForkedOnchainClient interface {
	// SendTransaction sends transaction data from one address to another.
	// Implementations should ensure that the transaction doesn't need a valid signature to be accepted.
	SendTransaction(ctx context.Context, from string, to string, data []byte) error
}

// ForkedEnvironment represents a forked deployment environment.
// It embeds a standard environment with the addition of a client for forking per chain.
type ForkedEnvironment struct {
	cldf.Environment
	ChainConfigs map[uint64]ChainConfig
	ForkClients  map[uint64]ForkedOnchainClient
}

// LoadForkedEnvironment loads a deployment environment in which the chains are forks of real networks.
// Provides access to a forking client per chain that allows users to send transactions without signatures.
//
// Limitations:
// - EVM only
func LoadForkedEnvironment(ctx context.Context, lggr logger.Logger, env string, domain domain.Domain, blockNumbers map[uint64]*big.Int, opts ...LoadEnvironmentOption) (ForkedEnvironment, error) {
	// Default options
	options := &LoadEnvironmentOptions{
		reporter:          operations.NewMemoryReporter(),
		operationRegistry: operations.NewOperationRegistry(),
	}
	for _, opt := range opts {
		opt(options)
	}
	config, err := cldf_config.Load(domain, env, lggr)
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Limit to EVM networks only
	networks := config.Networks.FilterWith(
		cldf_config_network.ChainFamilyFilter(chainselectors.FamilyEVM),
	)

	envdir := domain.EnvDir(env)
	ab, err := envdir.AddressBook()
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load address book for domain %s and environment %s: %w", domain.Key(), env, err)
	}
	anvilOutput, err := newAnvilChains(
		ctx,
		lggr,
		ab,
		networks,
		blockNumbers,
		config.Env.Onchain,
		config.Env.Onchain.KMS,
		options.chainSelectorsToLoad,
		options.anvilKeyAsDeployer,
	)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			err = errors.Join(errors.New("check your VPN connection"), err)
		}

		return ForkedEnvironment{}, fmt.Errorf("failed to create anvil chains: %w", err)
	}
	nodes, err := envdir.LoadNodes()
	if err != nil {
		return ForkedEnvironment{}, fmt.Errorf("failed to load nodes: %w", err)
	}

	cfg, err := cldf_config.LoadEnvConfig(domain, env)
	if err != nil {
		return ForkedEnvironment{}, err
	}

	var oc cldf_offchain.Client

	if !options.withoutJD {
		oc, err = cldf_engine_offchain.LoadOffchainClient(ctx, domain, env, cfg, lggr, false)
		if err != nil {
			return ForkedEnvironment{}, fmt.Errorf("failed to load offchain client: %w", err)
		}
	} else {
		lggr.Info("Override: skipping JD initialization")
	}

	// TODO: once all products are on the new datastore, we can remove this default
	ds := datastore.NewMemoryDataStore().Seal()
	if s, err := envdir.DataStore(); err == nil {
		ds = s
	} else {
		lggr.Warnf("failed to load datastore: %v", err)
	}

	blockChains := map[uint64]chain.BlockChain{}
	for selector, ch := range anvilOutput.Chains {
		blockChains[selector] = ch
	}

	// TODO: newSolChains, newAptosChains, etc.
	environment := cldf.NewEnvironment(
		"fork",
		lggr,
		ab,
		ds,
		nodes.Keys(),
		oc,
		func() context.Context { return ctx },
		focr.XXXGenerateTestOCRSecrets(),
		chain.NewBlockChains(blockChains),
	)

	return ForkedEnvironment{
		Environment:  *environment,
		ChainConfigs: anvilOutput.ChainConfigs,
		ForkClients:  anvilOutput.ForkClients, // TODO: map should eventually include clients from other families
	}, nil
}

// ApplyChangesetOutput executes MCMS proposals and merges addresses into the address book.
func (e ForkedEnvironment) ApplyChangesetOutput(ctx context.Context, output cldf.ChangesetOutput) (ForkedEnvironment, error) {
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
	if output.AddressBook != nil { //nolint
		err := e.ExistingAddresses.Merge(output.AddressBook) //nolint
		if err != nil {
			return ForkedEnvironment{}, fmt.Errorf("failed to merge new addresses into address book: %w", err)
		}
	}

	return e, nil
}
