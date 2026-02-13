package mcms

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
	"github.com/xssnick/tonutils-go/tlb"

	suibindings "github.com/smartcontractkit/chainlink-sui/bindings"
)

// getInspectorFromChainSelector returns an inspector for the given chain selector.
func getInspectorFromChainSelector(cfg *forkConfig) (sdk.Inspector, error) {
	fam, err := types.GetChainSelectorFamily(types.ChainSelector(cfg.chainSelector))
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		evmChain := cfg.blockchains.EVMChains()[cfg.chainSelector]
		inspector = evm.NewInspector(evmChain.Client)
	case chainsel.FamilySolana:
		solanaChain := cfg.blockchains.SolanaChains()[cfg.chainSelector]
		inspector = solana.NewInspector(solanaChain.Client)
	case chainsel.FamilyAptos:
		role, err := aptosRoleFromProposal(cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		aptosChain := cfg.blockchains.AptosChains()[cfg.chainSelector]
		inspector = aptos.NewInspector(aptosChain.Client, *role)
	case chainsel.FamilySui:
		metadata, err := suiMetadataFromProposal(types.ChainSelector(cfg.chainSelector), cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		suiChain := cfg.blockchains.SuiChains()[cfg.chainSelector]
		inspector, err = sui.NewInspector(suiChain.Client, suiChain.Signer, metadata.McmsPackageID, metadata.Role)
		if err != nil {
			return nil, fmt.Errorf("error creating sui inspector: %w", err)
		}
	case chainsel.FamilyTon:
		tonChain := cfg.blockchains.TonChains()[cfg.chainSelector]
		inspector = ton.NewInspector(tonChain.Client)
	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}

// createExecutable creates an MCMS executable for the proposal.
func createExecutable(cfg *forkConfig) (*mcms.Executable, error) {
	executors := make(map[types.ChainSelector]sdk.Executor, len(cfg.proposal.ChainMetadata))
	for chainSelector := range cfg.proposal.ChainMetadata {
		if cfg.chainSelector == 0 || cfg.chainSelector == uint64(chainSelector) {
			executor, err := getExecutorWithChainOverride(cfg, chainSelector)
			if err != nil {
				return &mcms.Executable{}, fmt.Errorf("unable to get executor with chain override: %w", err)
			}
			executors[chainSelector] = executor
		}
	}

	return mcms.NewExecutable(&cfg.proposal, executors)
}

// createTimelockExecutable creates a timelock executable for the proposal.
func createTimelockExecutable(ctx context.Context, cfg *forkConfig) (*mcms.TimelockExecutable, error) {
	executors := make(map[types.ChainSelector]sdk.TimelockExecutor, len(cfg.timelockProposal.ChainMetadata))
	for chainSelector := range cfg.timelockProposal.ChainMetadata {
		if cfg.chainSelector != 0 && cfg.chainSelector != uint64(chainSelector) {
			continue
		}
		executor, err := getTimelockExecutorWithChainOverride(cfg, chainSelector)
		if err != nil {
			return &mcms.TimelockExecutable{}, err
		}
		executors[chainSelector] = executor
	}

	return mcms.NewTimelockExecutable(ctx, cfg.timelockProposal, executors)
}

// getExecutorWithChainOverride returns an executor for the given chain selector.
func getExecutorWithChainOverride(cfg *forkConfig, chainSelector types.ChainSelector) (sdk.Executor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	encoders, err := cfg.proposal.GetEncoders()
	if err != nil {
		return nil, fmt.Errorf("error getting encoders: %w", err)
	}
	encoder, ok := encoders[chainSelector]
	if !ok {
		return nil, fmt.Errorf("unable to get encoder from proposal for chain selector %v", chainSelector)
	}

	switch family {
	case chainsel.FamilyEVM:
		evmEncoder, ok := encoder.(*evm.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type: %T", encoder)
		}
		c := cfg.blockchains.EVMChains()[uint64(chainSelector)]

		return evm.NewExecutor(evmEncoder, c.Client, c.DeployerKey), nil

	case chainsel.FamilySolana:
		solanaEncoder, ok := encoder.(*solana.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type: %T", encoder)
		}
		c := cfg.blockchains.SolanaChains()[uint64(chainSelector)]

		return solana.NewExecutor(solanaEncoder, c.Client, *c.DeployerKey), nil

	case chainsel.FamilyAptos:
		aptosEncoder, ok := encoder.(*aptos.Encoder)
		if !ok {
			return nil, fmt.Errorf("error getting encoder for chain %d", cfg.chainSelector)
		}
		role, err := aptosRoleFromProposal(cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		c := cfg.blockchains.AptosChains()[uint64(chainSelector)]

		return aptos.NewExecutor(c.Client, c.DeployerSigner, aptosEncoder, *role), nil

	case chainsel.FamilySui:
		suiEncoder, ok := encoder.(*sui.Encoder)
		if !ok {
			return nil, fmt.Errorf("error getting encoder for chain %d", cfg.chainSelector)
		}
		metadata, err := suiMetadataFromProposal(chainSelector, cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		c := cfg.blockchains.SuiChains()[uint64(chainSelector)]
		entrypointEncoder := suibindings.NewCCIPEntrypointArgEncoder(metadata.RegistryObj, metadata.DeployerStateObj)

		return sui.NewExecutor(c.Client, c.Signer, suiEncoder, entrypointEncoder, metadata.McmsPackageID, metadata.Role, cfg.timelockProposal.ChainMetadata[chainSelector].MCMAddress, metadata.AccountObj, metadata.RegistryObj, metadata.TimelockObj)

	case chainsel.FamilyTon:
		tonEncoder, ok := encoder.(*ton.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for TON chain %d: expected *ton.Encoder, got %T", chainSelector, encoder)
		}
		c := cfg.blockchains.TonChains()[uint64(chainSelector)]
		opts := ton.ExecutorOpts{
			Encoder: tonEncoder,
			Client:  c.Client,
			Wallet:  c.Wallet,
			Amount:  tlb.MustFromTON(defaultTONExecutorAmount),
		}

		return ton.NewExecutor(opts)

	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}

// getTimelockExecutorWithChainOverride returns a timelock executor for the given chain selector.
func getTimelockExecutorWithChainOverride(cfg *forkConfig, chainSelector types.ChainSelector) (sdk.TimelockExecutor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	var executor sdk.TimelockExecutor
	switch family {
	case chainsel.FamilyEVM:
		c := cfg.blockchains.EVMChains()[uint64(chainSelector)]
		executor = evm.NewTimelockExecutor(c.Client, c.DeployerKey)
	case chainsel.FamilySolana:
		c := cfg.blockchains.SolanaChains()[uint64(chainSelector)]
		executor = solana.NewTimelockExecutor(c.Client, *c.DeployerKey)
	case chainsel.FamilyAptos:
		c := cfg.blockchains.AptosChains()[uint64(chainSelector)]
		executor = aptos.NewTimelockExecutor(c.Client, c.DeployerSigner)
	case chainsel.FamilySui:
		c := cfg.blockchains.SuiChains()[uint64(chainSelector)]
		metadata, err := suiMetadataFromProposal(chainSelector, cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		entrypointEncoder := suibindings.NewCCIPEntrypointArgEncoder(metadata.RegistryObj, metadata.DeployerStateObj)
		executor, err = sui.NewTimelockExecutor(c.Client, c.Signer, entrypointEncoder, metadata.McmsPackageID, metadata.RegistryObj, metadata.AccountObj)
		if err != nil {
			return nil, fmt.Errorf("error creating sui timelock executor: %w", err)
		}
	case chainsel.FamilyTon:
		c := cfg.blockchains.TonChains()[uint64(chainSelector)]
		opts := ton.TimelockExecutorOpts{
			Client: c.Client,
			Wallet: c.Wallet,
			Amount: tlb.MustFromTON(defaultTONExecutorAmount),
		}

		return ton.NewTimelockExecutor(opts)
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}

	return executor, nil
}
