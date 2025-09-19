package mcmsutils

import (
	"context"
	"errors"
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

type Executor struct {
	env fdeployment.Environment
}

func NewExecutor(e fdeployment.Environment) *Executor {
	return &Executor{env: e}
}

func (e *Executor) ExecuteTimelock(ctx context.Context, proposal *mcmslib.TimelockProposal) error {
	executors := make(map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor, 0)
	callProxies := make([]string, len(proposal.Operations))

	// Generate executors for each operation
	for i, op := range proposal.Operations {
		selector := uint64(op.ChainSelector)

		b, err := e.env.BlockChains.GetBySelector(selector)
		if err != nil {
			return fmt.Errorf("get blockchain for chain %d: %w", op.ChainSelector, err) // Not found in the list of available chains
		}

		// Don't love that we are putting chain specific logic here.
		if b.Family() == chainselectors.FamilyEVM {
			var proxyAddr string
			proxyAddr, err = findCallProxyAddress(e.env.DataStore.Addresses(), selector)
			if err != nil {
				return fmt.Errorf("find call proxy address for chain %d: %w", selector, err)
			}
			callProxies[i] = proxyAddr
		}

		if executors[op.ChainSelector] != nil {
			continue
		}

		execFactory, err := GetTimelockExecutorFactory(b)
		if err != nil {
			return fmt.Errorf("get executor factory for chain %d: %w", selector, err)
		}

		executor, err := execFactory.Make()
		if err != nil {
			return fmt.Errorf("make executor for chain %d: %w", selector, err)
		}
		executors[op.ChainSelector] = executor
	}

	executable, err := mcmslib.NewTimelockExecutable(ctx, proposal, executors)
	if err != nil {
		return fmt.Errorf("new timelock executable: %w", err)
	}

	// Wait until ready. Times out after 10 seconds.
	if err := retry.Do(
		func() error {
			return executable.IsReady(ctx)
		},
		retry.Attempts(100),
		retry.Delay(100*time.Millisecond),
	); err != nil {
		return fmt.Errorf("proposal is not ready: %w", err)
	}

	// execute each operation sequentially
	for i, op := range proposal.Operations {
		executeOpts := make([]mcmslib.Option, 0)
		if callProxies[i] != "" {
			executeOpts = append(executeOpts, mcmslib.WithCallProxy(callProxies[i]))
		}

		tx, err := executable.Execute(ctx, i, executeOpts...)
		if err != nil {
			return fmt.Errorf("[ExecuteMCMSTimelockProposalV2] Execute failed: %w", err)
		}

		blockchain, err := e.env.BlockChains.GetBySelector(uint64(op.ChainSelector))
		if err != nil {
			return fmt.Errorf("get blockchain for chain %d: %w", op.ChainSelector, err)
		}

		// Confirm the transaction on the chain
		if err = confirmTransaction(blockchain, tx); err != nil {
			return fmt.Errorf("confirm transaction for chain %d: %w", op.ChainSelector, err)
		}
	}

	return nil
}

// Assumes a version of 1 with no qualifier
//
// We only support datastore.
func findCallProxyAddress(ds datastore.AddressRefStore, selector uint64) (string, error) {
	ref, err := ds.Get(
		datastore.NewAddressRefKey(selector, "CallProxy", semver.MustParse("1.0.0"), ""),
	)
	if err != nil {
		return "", err
	}

	return ref.Address, nil
}

func confirmTransaction(blockchain fchain.BlockChain, tx mcmstypes.TransactionResult) error {
	switch chain := blockchain.(type) {
	case fchainevm.Chain:
		evmTx, ok := tx.RawData.(*gethtypes.Transaction)
		if !ok {
			return errors.New("tx is not a gethtypes.Transaction")
		}

		if _, err := chain.Confirm(evmTx); err != nil {
			return fmt.Errorf("confirm EVM transaction: %w", err)
		}
	case fchainaptos.Chain:
		aptosTx, ok := tx.RawData.(*aptosapi.PendingTransaction)
		if !ok {
			return errors.New("tx is not a aptosapi.PendingTransaction")
		}

		if err := chain.Confirm(aptosTx.Hash); err != nil {
			return fmt.Errorf("confirm Aptos transaction: %w", err)
		}
	case fchainsolana.Chain:
		// NOOP: no need to confirm transaction on solana as the MCMS sdk confirms it internally
	}

	return nil
}
