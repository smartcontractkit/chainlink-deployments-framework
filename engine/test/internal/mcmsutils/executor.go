package mcmsutils

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	aptosapi "github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/avast/retry-go/v4"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// Executor provides functionality to execute MCMS proposals and timelock proposals across
// multiple blockchain networks. It handles the orchestration of proposal execution, transaction
// confirmation, and retry logic for timelock operations.
type Executor struct {
	// env contains the deployment environment with blockchain configurations
	env fdeployment.Environment

	// newExecutable creates a new mcmsExecutable. This can be overridden for testing.
	newExecutable func(
		proposal *mcmslib.Proposal,
		executors map[mcmstypes.ChainSelector]mcmssdk.Executor,
	) (mcmsExecutable, error)

	// newTimelockExecutable creates a new timelockExecutable. This can be overridden for
	// testing.
	newTimelockExecutable func(
		ctx context.Context,
		timelockProposal *mcmslib.TimelockProposal,
		executors map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor,
	) (timelockExecutable, error)

	// retryAttempts is the number of attempts to retry when the timelock is not ready.
	retryAttempts uint
	// retryDelay is the delay between retry attempts.
	retryDelay time.Duration
}

// NewExecutor creates a new Executor with default configuration.
//
// The executor is initialized with:
// - Default executable factory functions for MCMS and timelock operations
// - 50 retry attempts for timelock readiness checks
// - 100ms delay between retry attempts
func NewExecutor(e fdeployment.Environment) *Executor {
	return &Executor{
		env:                   e,
		newExecutable:         newDefaultExecutable,
		newTimelockExecutable: newDefaultTimelockExecutable,
		retryAttempts:         50,
		retryDelay:            100 * time.Millisecond,
	}
}

// ExecuteMCMS executes a multi-chain multi-sig proposal across all specified chains.
//
// The execution process includes:
// 1. Validating the proposal structure and metadata
// 2. Creating chain-specific executors for each target chain
// 3. Setting the merkle root on each chain to initialize the proposal
// 4. Executing each operation sequentially and confirming transactions
//
// Returns an error if any step fails
func (e *Executor) ExecuteMCMS(ctx context.Context, proposal *mcmslib.Proposal) error {
	// Validate the proposal to ensure it is valid ensuring that all chain metadata is present.
	if err := proposal.Validate(); err != nil {
		return fmt.Errorf("failed to validate MCMS proposal: %w", err)
	}

	// Determine the blockchains to use for the operations ensuring the environment has the
	// necessary chains configured in the proposal's ChainMetadata.
	blockchains, err := getBlockchainsForProposal(e.env, proposal)
	if err != nil {
		return fmt.Errorf("failed to get blockchains from environment: %w", err)
	}

	// Get the encoders for the proposal which are required to create the executors
	encoders, err := proposal.GetEncoders()
	if err != nil {
		return fmt.Errorf("failed to retrieve encoders from MCMS proposal: %w", err)
	}

	// Generate executors for each chain
	executors := make(map[mcmstypes.ChainSelector]mcmssdk.Executor, 0)
	for selector := range proposal.ChainMetadata {
		b := blockchains[selector]

		execFactory, ferr := GetExecutorFactory(b, encoders[selector])
		if ferr != nil {
			return fmt.Errorf("failed to create executor factory for chain selector %d (%s): %w", selector, b.Name(), ferr)
		}

		executor, merr := execFactory.Make()
		if merr != nil {
			return fmt.Errorf("failed to create executor for chain selector %d (%s): %w", selector, b.Name(), merr)
		}
		executors[selector] = executor
	}

	executable, err := e.newExecutable(proposal, executors)
	if err != nil {
		return fmt.Errorf("failed to create executable from MCMS proposal and executors: %w", err)
	}

	// Call SetRoot on each chain to initialize the proposal for each chain in the proposal metadata
	for selector := range proposal.ChainMetadata {
		txResult, err := executable.SetRoot(ctx, selector)
		if err != nil {
			return fmt.Errorf("failed to set root for chain selector %d: %w", selector, err)
		}

		b := blockchains[selector]

		if err := confirmTransaction(b, txResult); err != nil {
			return fmt.Errorf("failed to confirm SetRoot transaction for chain selector %d (%s): %w", selector, b.Name(), err)
		}
	}

	// Execute each operation in the proposal sequentially
	for i, op := range proposal.Operations {
		txResult, err := executable.Execute(ctx, i)
		if err != nil {
			return fmt.Errorf("failed to execute operation %d on chain selector %d: %w", i, op.ChainSelector, err)
		}

		b := blockchains[op.ChainSelector]

		if err := confirmTransaction(b, txResult); err != nil {
			return fmt.Errorf(
				"failed to confirm execute transaction for operation %d on chain selector %d (%s): %w",
				i, op.ChainSelector, b.Name(), err,
			)
		}
	}

	return nil
}

// ExecuteTimelock executes a timelock proposal, which involves both MCMS execution
// and timelock-specific operations. The execution process includes:
//
//  1. Validating the timelock proposal
//  2. Converting the timelock proposal to an MCMS proposal and executing it
//  3. For scheduled actions, executing operations on the timelock contract after
//     waiting for the proposal to become ready (with retry logic)
//  4. Handling chain-specific configurations like call proxies for EVM chains
//
// Returns early for non-scheduled actions. For scheduled actions, waits up to
// 5 seconds (configurable via retryAttempts and retryDelay) for the timelock
// to become ready before executing operations.
func (e *Executor) ExecuteTimelock(ctx context.Context, timelockProposal *mcmslib.TimelockProposal) error {
	// Validate the proposal to ensure it is valid. This ensures that all chain metadata is present.
	if err := timelockProposal.Validate(); err != nil {
		return fmt.Errorf("failed to validate MCMS proposal: %w", err)
	}

	// Determine the blockchains to use for the operations. This ensures the environment has the
	// necessary chains configured in the proposal's ChainMetadata.
	blockchains, err := getBlockchainsForTimelockProposal(e.env, timelockProposal)
	if err != nil {
		return fmt.Errorf("failed to get blockchains from environment: %w", err)
	}

	// Convert the timelock proposal to an MCMS proposal
	proposal, err := convertTimelock(ctx, *timelockProposal)
	if err != nil {
		return fmt.Errorf("failed to convert timelock proposal to MCMS proposal: %w", err)
	}

	// Execute the proposal against the MCMS Contract
	if err = e.ExecuteMCMS(ctx, proposal); err != nil {
		return fmt.Errorf("failed to execute MCMS proposal: %w", err)
	}

	// Return early if the action is not on a schedule because we don't need to execute the
	// proposal on the Timelock contract.
	if timelockProposal.Action != mcmstypes.TimelockActionSchedule {
		return nil
	}

	// Now we execute the proposal on the Timelock contract

	// Generate executors for each blockchain
	executors := make(map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor, 0)
	for selector, b := range blockchains {
		execFactory, gerr := GetTimelockExecutorFactory(b)
		if gerr != nil {
			return fmt.Errorf("failed to create timelock executor factory for chain selector %d (%s): %w", selector, b.Name(), gerr)
		}

		executor, merr := execFactory.Make()
		if merr != nil {
			return fmt.Errorf("failed to create timelock executor for chain selector %d (%s): %w", selector, b.Name(), merr)
		}
		executors[selector] = executor
	}

	// Generate call proxies for each EVM operation.
	callProxies := make([]string, len(timelockProposal.Operations))
	for i, op := range timelockProposal.Operations {
		b := blockchains[op.ChainSelector]

		// Don't love that we are putting chain specific logic here.
		if b.Family() == chainselectors.FamilyEVM {
			var proxyAddr string
			proxyAddr, err = findCallProxyAddress(e.env.DataStore.Addresses(), uint64(op.ChainSelector))
			if err != nil {
				return fmt.Errorf(
					"ensure CallProxy is deployed and configured in datastore: %w", err,
				)
			}

			callProxies[i] = proxyAddr
		}
	}

	executable, err := e.newTimelockExecutable(ctx, timelockProposal, executors)
	if err != nil {
		return fmt.Errorf("failed to create timelock executable from proposal and executors: %w", err)
	}

	// Wait until ready. Times out after 5 seconds.
	if err := retry.Do(
		func() error {
			return executable.IsReady(ctx)
		},
		retry.Attempts(e.retryAttempts),
		retry.Delay(e.retryDelay),
	); err != nil {
		return fmt.Errorf(
			"timelock proposal is not ready for execution: this may indicate insufficient signatures or proposal not yet scheduled: %w", err)
	}

	// Execute each operation in the proposal sequentially
	for i, op := range timelockProposal.Operations {
		b := blockchains[op.ChainSelector]

		executeOpts := make([]mcmslib.Option, 0)
		if callProxies[i] != "" {
			executeOpts = append(executeOpts, mcmslib.WithCallProxy(callProxies[i]))
		}

		tx, err := executable.Execute(ctx, i, executeOpts...)
		if err != nil {
			return fmt.Errorf("failed to execute timelock operation %d on chain selector %d: %w", i, op.ChainSelector, err)
		}

		// Confirm the transaction on the chain
		if err = confirmTransaction(b, tx); err != nil {
			return fmt.Errorf(
				"failed to confirm timelock execution transaction for operation %d on chain selector %d (%s): %w",
				i, op.ChainSelector, b.Name(), err,
			)
		}
	}

	return nil
}

// findCallProxyAddress retrieves the CallProxy contract address for a given chain selector.
// It looks up the address in the datastore using version 1.0.0 with no qualifier.
// Currently only supports datastore-based address resolution.
func findCallProxyAddress(ds datastore.AddressRefStore, selector uint64) (string, error) {
	ref, err := ds.Get(
		datastore.NewAddressRefKey(selector, "CallProxy", semver.MustParse("1.0.0"), ""),
	)
	if err != nil {
		return "", fmt.Errorf("CallProxy address not found in datastore (chain selector: %d, version: 1.0.0): %w", selector, err)
	}

	return ref.Address, nil
}

// confirmTransaction confirms a transaction on the appropriate blockchain based on its type.
// Supports EVM, Aptos, and Solana chains with chain-specific confirmation logic.
// For Solana chains, confirmation is handled internally by the MCMS SDK.
func confirmTransaction(blockchain fchain.BlockChain, tx mcmstypes.TransactionResult) error {
	switch chain := blockchain.(type) {
	case fchainevm.Chain:
		evmTx, ok := tx.RawData.(*gethtypes.Transaction)
		if !ok {
			return fmt.Errorf("invalid transaction type for EVM chain %s: expected *gethtypes.Transaction, got %T", chain.Name(), tx.RawData)
		}

		if _, err := chain.Confirm(evmTx); err != nil {
			return fmt.Errorf("failed to confirm EVM transaction %s on chain %s: %w", evmTx.Hash().Hex(), chain.Name(), err)
		}
	case fchainaptos.Chain:
		aptosTx, ok := tx.RawData.(*aptosapi.PendingTransaction)
		if !ok {
			return fmt.Errorf("invalid transaction type for Aptos chain %s: expected *aptosapi.PendingTransaction, got %T", chain.Name(), tx.RawData)
		}

		if err := chain.Confirm(aptosTx.Hash); err != nil {
			return fmt.Errorf("failed to confirm Aptos transaction %s on chain %s: %w", aptosTx.Hash, chain.Name(), err)
		}
	case fchainsolana.Chain:
		// NOOP: no need to confirm transaction on solana as the MCMS sdk confirms it internally
	default:
		return fmt.Errorf("unsupported blockchain type for transaction confirmation: %T", blockchain)
	}

	return nil
}

// getBlockchainsForProposal extracts and validates blockchain instances from the environment
// for all chains referenced in the proposal's metadata.
//
// Returns a map of chain selectors to their corresponding blockchain instances.
func getBlockchainsForProposal(
	env fdeployment.Environment, proposal *mcmslib.Proposal,
) (map[mcmstypes.ChainSelector]fchain.BlockChain, error) {
	blockchains := make(map[mcmstypes.ChainSelector]fchain.BlockChain, 0)
	for selector := range proposal.ChainMetadata {
		// Skip if already added
		if _, ok := blockchains[selector]; ok {
			continue
		}

		b, err := env.BlockChains.GetBySelector(uint64(selector))
		if err != nil {
			return nil, fmt.Errorf(
				"blockchain not found for chain selector %d: ensure the chain is configured in the provided environment: %w",
				selector, err,
			)
		}

		blockchains[selector] = b
	}

	return blockchains, nil
}

// getBlockchainsForTimelockProposal extracts and validates blockchain instances from the environment
// for all chains referenced in the timelock proposal's metadata.
//
// Returns a map of chain selectors to their corresponding blockchain instances.
func getBlockchainsForTimelockProposal(
	env fdeployment.Environment, proposal *mcmslib.TimelockProposal,
) (map[mcmstypes.ChainSelector]fchain.BlockChain, error) {
	blockchains := make(map[mcmstypes.ChainSelector]fchain.BlockChain, 0)
	for selector := range proposal.ChainMetadata {
		// Skip if already added
		if _, ok := blockchains[selector]; ok {
			continue
		}

		b, err := env.BlockChains.GetBySelector(uint64(selector))
		if err != nil {
			return nil, fmt.Errorf(
				"blockchain not found for chain selector %d: ensure the chain is configured in the provided environment: %w",
				selector, err,
			)
		}

		blockchains[selector] = b
	}

	return blockchains, nil
}

// mcmsExecutable defines the interface for executing MCMS proposals.
// It provides methods to set the merkle root and execute individual operations.
type mcmsExecutable interface {
	// SetRoot initializes the proposal on the specified chain by setting the merkle root
	SetRoot(ctx context.Context, chainSelector mcmstypes.ChainSelector) (mcmstypes.TransactionResult, error)
	// Execute runs the operation at the specified index
	Execute(ctx context.Context, index int) (mcmstypes.TransactionResult, error)
}

// newDefaultExecutable creates a standard MCMS executable using the mcmslib.NewExecutable constructor.
// This is the default factory function used by Executor unless overridden for testing.
func newDefaultExecutable(
	proposal *mcmslib.Proposal,
	executors map[mcmstypes.ChainSelector]mcmssdk.Executor,
) (mcmsExecutable, error) {
	return mcmslib.NewExecutable(proposal, executors)
}

// timelockExecutable defines the interface for executing timelock proposals.
// It provides methods to check readiness and execute operations with optional configurations.
type timelockExecutable interface {
	// IsReady checks if the timelock proposal is ready for execution
	IsReady(ctx context.Context) error
	// Execute runs the operation at the specified index with optional execution parameters
	Execute(ctx context.Context, index int, opts ...mcmslib.Option) (mcmstypes.TransactionResult, error)
}

// newDefaultTimelockExecutable creates a standard timelock executable using the mcmslib.NewTimelockExecutable constructor.
// This is the default factory function used by Executor unless overridden for testing.
func newDefaultTimelockExecutable(
	ctx context.Context,
	timelockProposal *mcmslib.TimelockProposal,
	executors map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor,
) (timelockExecutable, error) {
	return mcmslib.NewTimelockExecutable(ctx, timelockProposal, executors)
}
