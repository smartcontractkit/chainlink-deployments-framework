package mcms

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
	"github.com/xssnick/tonutils-go/tlb"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// ErrOperationAlreadyExecuted is returned when an operation has already been executed.
var ErrOperationAlreadyExecuted = errors.New("operation already executed")

// setRootCommand sets the merkle root on the MCM contract.
func setRootCommand(ctx context.Context, lggr logger.Logger, cfg *forkConfig) error {
	if cfg.fork {
		lggr.Info("Fork mode is on, all transactions will be executed on a forked chain")
	}

	inspector, err := getInspectorFromChainSelector(cfg)
	if err != nil {
		return fmt.Errorf("failed to get inspector: %w", err)
	}

	proposalMerkleTree, err := cfg.proposal.MerkleTree()
	if err != nil {
		return fmt.Errorf("failed to compute the proposal's merkle tree: %w", err)
	}

	mcmAddress := cfg.proposal.ChainMetadata[types.ChainSelector(cfg.chainSelector)].MCMAddress
	mcmRoot, _, err := inspector.GetRoot(ctx, mcmAddress)
	if err != nil {
		return fmt.Errorf("failed to get the merkle tree root from the MCM contract (%v): %w", mcmAddress, err)
	}

	if mcmRoot == proposalMerkleTree.Root {
		lggr.Infof("Root %v already set in MCM contract %v", mcmRoot, mcmAddress)

		return nil
	}

	executable, err := createExecutable(cfg)
	if err != nil {
		return fmt.Errorf("error converting proposal to executable: %w", err)
	}

	tx, err := executable.SetRoot(ctx, types.ChainSelector(cfg.chainSelector))
	if err != nil {
		err = cldf.DecodeErr(bindings.ManyChainMultiSigABI, err)

		return fmt.Errorf("error setting root: %w", err)
	}

	err = confirmTransaction(ctx, lggr, tx, cfg)
	if err != nil {
		return fmt.Errorf("failed to confirm set root transaction: %w", err)
	}

	return nil
}

// executeChainCommand executes all operations on the chain.
func executeChainCommand(ctx context.Context, lggr logger.Logger, cfg *forkConfig, skipNonceErrors bool) error {
	executable, err := createExecutable(cfg)
	if err != nil {
		return fmt.Errorf("error converting proposal to executable: %w", err)
	}
	inspector, err := getInspectorFromChainSelector(cfg)
	if err != nil {
		return fmt.Errorf("failed to get inspector: %w", err)
	}

	if cfg.fork {
		lggr.Info("Fork mode is on, all transactions will be executed on a forked chain")
	}

	for i, op := range cfg.proposal.Operations {
		// TODO; consider multi-chain support
		if op.ChainSelector != types.ChainSelector(cfg.chainSelector) {
			continue
		}

		err := checkTxNonce(ctx, lggr, cfg, executable, inspector, i)
		if errors.Is(err, ErrOperationAlreadyExecuted) {
			return nil
		}
		if err != nil {
			return err
		}

		tx, err := executable.Execute(ctx, i)
		if err != nil {
			lggr.Errorf("error executing operation %d: %s", i, err)
			if skipNonceErrors {
				nonceErr, errNonceCheck := isNonceError(err, cfg.chainSelector)
				if errNonceCheck != nil {
					return fmt.Errorf("error checking nonce error: %w", err)
				}
				if nonceErr {
					lggr.Warnf("Skipping nonce error for operation %d", i)

					continue
				}
			}
			family, familyErr := chainsel.GetSelectorFamily(uint64(op.ChainSelector))
			if familyErr != nil {
				lggr.Errorf("error getting chain family: %w", familyErr)
			}
			switch family {
			case chainsel.FamilyEVM:
				err = cldf.DecodeErr(bindings.ManyChainMultiSigABI, err)

				return fmt.Errorf("error executing chain op %d: %w", i, err)
			}

			return err
		}
		lggr.Infof("Transaction sent: %s", tx.Hash)

		err = confirmTransaction(ctx, lggr, tx, cfg)
		if err != nil {
			return fmt.Errorf("unable to confirm execute(%d) transaction: %w", i, err)
		}
	}

	return nil
}

// timelockExecuteChainCommand executes timelock operations.
func timelockExecuteChainCommand(ctx context.Context, lggr logger.Logger, cfg *forkConfig) error {
	if cfg.timelockProposal == nil {
		return errors.New("expected proposal to be have non-nil *TimelockProposal")
	}

	executable, err := createTimelockExecutable(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create TimelockExecutable: %w", err)
	}

	executeOptions, err := timelockExecuteOptions(ctx, lggr, cfg)
	if err != nil {
		return fmt.Errorf("failed to get timelock execute options: %w", err)
	}

	for i := range cfg.timelockProposal.Operations {
		if uint64(cfg.timelockProposal.Operations[i].ChainSelector) == cfg.chainSelector {
			// Check if operation is done, if so, skip it
			if err := executable.IsOperationDone(ctx, i); err == nil {
				lggr.Warnf("Operation %d is already done, skipping...\n", i)

				continue
			}

			if err := executable.IsOperationReady(ctx, i); err != nil {
				return fmt.Errorf("operation %d is not ready to be executed: %w", i, err)
			}

			result, err := executable.Execute(ctx, i, executeOptions...)
			if err != nil {
				return fmt.Errorf("failed to execute operation %d: %w", i, err)
			}

			err = confirmTransaction(ctx, lggr, result, cfg)
			if err != nil {
				return fmt.Errorf("failed to confirm execute transaction: %w", err)
			}

			lggr.Infof("Operation %d executed successfully: %s\n", i, result)
		}
	}

	lggr.Infof("All operations executed successfully")

	return nil
}

// confirmTransaction waits for a transaction to be confirmed.
func confirmTransaction(ctx context.Context, lggr logger.Logger, tx types.TransactionResult, cfg *forkConfig) error {
	family, err := chainsel.GetSelectorFamily(cfg.chainSelector)
	if err != nil {
		return fmt.Errorf("error getting chain family: %w", err)
	}

	switch family {
	case chainsel.FamilyEVM:
		evmChain := cfg.blockchains.EVMChains()[cfg.chainSelector]
		block, err := evmChain.Confirm(tx.RawData.(*gethtypes.Transaction))
		if err == nil {
			lggr.Infof("Transaction %s confirmed in block %d", tx.Hash, block)

			return nil
		}
		lggr.Errorf("failed to confirm transaction %s: %s", tx.Hash, err)
		rcpt, rerr := evmChain.Client.TransactionReceipt(ctx, common.HexToHash(tx.Hash))
		if rerr != nil {
			return fmt.Errorf("failed to get transaction receipt for %s: %w", tx.Hash, rerr)
		}
		if rcpt == nil {
			return fmt.Errorf("got nil receipt for %s", tx.Hash)
		}
		if rcpt.Status == gethtypes.ReceiptStatusSuccessful {
			return nil
		}
		if cfg.proposalCtx != nil {
			// Decode via simulation to recover revert bytes
			pretty, ok := tryDecodeTxRevertEVM(ctx, evmChain.Client, tx.RawData.(*gethtypes.Transaction),
				bindings.ManyChainMultiSigABI, rcpt.BlockNumber, cfg.proposalCtx)
			if ok {
				return fmt.Errorf("tx %s reverted: %s", tx.Hash, pretty)
			}
		}

		return fmt.Errorf("transaction %s failed (block number %v): %w", tx.Hash, rcpt.BlockNumber, err)

	case chainsel.FamilyAptos:
		aptosChain := cfg.blockchains.AptosChains()[cfg.chainSelector]
		err := aptosChain.Confirm(tx.Hash)
		if err != nil {
			return fmt.Errorf("failed to confirm transaction %s: %w", tx.Hash, err)
		}
		lggr.Infof("Transaction %s confirmed", tx.Hash)

		return nil

	case chainsel.FamilyTon:
		tonChain := cfg.blockchains.TonChains()[cfg.chainSelector]
		tonTx, ok := tx.RawData.(*tlb.Transaction)
		if !ok {
			return fmt.Errorf("invalid transaction raw data type: %T", tx.RawData)
		}
		err := tonChain.Confirm(ctx, tonTx)
		if err != nil {
			return fmt.Errorf("failed to confirm transaction %s: %w", tx.Hash, err)
		}
		lggr.Infof("Transaction %s confirmed", tx.Hash)

		return nil

	default:
		return nil // not supported yet, pass through
	}
}

// checkTxNonce verifies the transaction nonce is valid before execution.
func checkTxNonce(
	ctx context.Context, lggr logger.Logger, cfg *forkConfig, executable *mcms.Executable, inspector sdk.Inspector, i int,
) error {
	mcmAddress := cfg.proposal.ChainMetadata[types.ChainSelector(cfg.chainSelector)].MCMAddress

	txNonce, err := executable.TxNonce(i)
	if err != nil {
		return fmt.Errorf("failed to get TxNonce for chain %d: %w", cfg.chainSelector, err)
	}

	opCount, err := inspector.GetOpCount(ctx, mcmAddress)
	if err != nil {
		return fmt.Errorf("failed to get opcount for chain %d: %w", cfg.chainSelector, err)
	}

	if txNonce < opCount {
		lggr.Infow("operation already executed", "index", i, "txNonce", txNonce, "opCount", opCount)

		return ErrOperationAlreadyExecuted
	}
	if txNonce > opCount {
		lggr.Warnw("txNonce too large", "index", i, "txNonce", txNonce, "opCount", opCount)

		return fmt.Errorf("txNonce too large for op %d (%d; expected %d)", i, txNonce, opCount)
	}

	return nil
}

// isNonceError checks if an error is a nonce-related error.
func isNonceError(rawErr error, selector uint64) (bool, error) {
	family, famErr := chainsel.GetSelectorFamily(selector)
	if famErr != nil {
		return false, famErr
	}

	switch family {
	case chainsel.FamilyEVM:
		decodedErr := cldf.DecodeErr(bindings.ManyChainMultiSigABI, rawErr)
		// Check if the error contains PostOpCountReached
		if strings.Contains(decodedErr.Error(), "PostOpCountReached") {
			return true, nil
		}

	case chainsel.FamilySolana:
		// Check if the error contains WrongNonce or PostOpCountReached
		if strings.Contains(rawErr.Error(), "WrongNonce") || strings.Contains(rawErr.Error(), "PostOpCountReached") {
			return true, nil
		}
	default:
		return false, nil
	}

	return false, nil
}

// timelockExecuteOptions returns options for timelock execution.
func timelockExecuteOptions(
	ctx context.Context, lggr logger.Logger, cfg *forkConfig,
) ([]mcms.Option, error) {
	var options []mcms.Option

	family, err := chainsel.GetSelectorFamily(cfg.chainSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get selector family: %w", err)
	}
	if family == chainsel.FamilyEVM {
		err := addCallProxyOption(ctx, lggr, cfg, &options)
		if err != nil {
			return options, fmt.Errorf("failed to add CallProxy option: %w", err)
		}
	}

	return options, nil
}

// addCallProxyOption adds the call proxy option if a CallProxy contract is found.
func addCallProxyOption(
	ctx context.Context, lggr logger.Logger, cfg *forkConfig, options *[]mcms.Option,
) error {
	timelockAddress, ok := cfg.timelockProposal.TimelockAddresses[types.ChainSelector(cfg.chainSelector)]
	if !ok {
		return fmt.Errorf("failed to find timelock address for chain selector %d", cfg.chainSelector)
	}

	evmChain, ok := cfg.blockchains.EVMChains()[cfg.chainSelector]
	if !ok {
		return fmt.Errorf("failed to find evm chain for selector %d", cfg.chainSelector)
	}

	timelockContract, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddress), evmChain.Client)
	if err != nil {
		return fmt.Errorf("failed to create timelock contract with address %v: %w", timelockAddress, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}

	role, err := timelockContract.EXECUTORROLE(callOpts)
	if err != nil {
		return fmt.Errorf("failed to get executor role from timelock contract: %w", err)
	}
	memberCount, err := timelockContract.GetRoleMemberCount(callOpts, role)
	if err != nil {
		return fmt.Errorf("failed to get executor member count from timelock contract: %w", err)
	}
	for i := range memberCount.Int64() {
		executorAddress, ierr := timelockContract.GetRoleMember(callOpts, role, big.NewInt(i))
		if ierr != nil {
			return fmt.Errorf("failed to get executor address from timelock contract: %w", ierr)
		}

		// search for executor address in the datastore
		callProxyRefs := cfg.env.DataStore.Addresses().Filter(
			datastore.AddressRefByAddress(executorAddress.Hex()),
			datastore.AddressRefByChainSelector(cfg.chainSelector),
			datastore.AddressRefByType("CallProxy"))

		if len(callProxyRefs) > 0 {
			*options = append(*options, mcms.WithCallProxy(executorAddress.Hex()))

			return nil
		}

		// if not found, search in the addressbook
		addressesForChain, ierr := cfg.env.ExistingAddresses.AddressesForChain(cfg.chainSelector) //nolint:staticcheck
		if ierr != nil {
			lggr.Infof("unable to get addresses for chain %d in addressbook: %s", cfg.chainSelector, ierr.Error())

			continue // ignore error; some domains don't use the addressbook anymore
		}
		for address, typeAndVersion := range addressesForChain {
			if address == executorAddress.Hex() && typeAndVersion.Type == "CallProxy" {
				*options = append(*options, mcms.WithCallProxy(executorAddress.Hex()))

				return nil
			}
		}
	}

	return fmt.Errorf("failed to find call proxy contract for timelock %v", timelockAddress)
}
