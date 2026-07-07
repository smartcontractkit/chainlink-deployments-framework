package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	mcms_types "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// ExecInfo contains information about an executed transaction.
// Defined as a struct in case we want to add more fields in the future without breaking existing usage.
type ExecInfo struct {
	// Hash is the transaction hash.
	Hash string
}

// WriteOutput is the output of a write operation.
type WriteOutput struct {
	// ChainSelector is the selector of the target chain.
	ChainSelector uint64 `json:"chainSelector"`
	// Tx is the prepared transaction (in MCMS format).
	Tx mcms_types.Transaction `json:"tx"`
	// ExecInfo is populated if the write was executed, contains info about the executed transaction.
	ExecInfo *ExecInfo `json:"execInfo,omitempty"`
}

func (o WriteOutput) Executed() bool {
	return o.ExecInfo != nil
}

type WriteParams[ARGS any, C any] struct {
	// Name is the name of the operation.
	Name string
	// Version is the version of the operation.
	Version *semver.Version
	// Description is a brief description of the operation.
	Description string
	// ContractType is the type of the target contract.
	ContractType deployment.ContractType
	// ContractABI is the ABI of the target contract.
	ContractABI string
	// Contract is the contract binding instance to use for this write operation.
	Contract C
	// IsAllowedCaller is a function that checks if the caller is allowed to call the function.
	IsAllowedCaller func(contract C, opts *bind.CallOpts, caller common.Address, input ARGS) (bool, error)
	// Validate is a function that validates the input arguments.
	Validate func(input ARGS) error
	// CallContract is a function that calls the desired write method on the contract.
	CallContract func(contract C, opts *bind.TransactOpts, input ARGS) (*eth_types.Transaction, error)
}

func NewWrite[ARGS any, C interface{ Address() common.Address }](params WriteParams[ARGS, C]) *operations.Operation[FunctionInput[ARGS], WriteOutput, evm.Chain] {
	return operations.NewOperation(
		params.Name,
		params.Version,
		params.Description,
		func(b operations.Bundle, chain evm.Chain, input FunctionInput[ARGS]) (WriteOutput, error) {
			// BEGIN Validation
			if params.Validate != nil {
				if err := params.Validate(input.Args); err != nil {
					return WriteOutput{}, fmt.Errorf("invalid args for %s: %w", params.Name, err)
				}
			}
			if params.ContractType == "" {
				return WriteOutput{}, fmt.Errorf("contract type must be specified for %s", params.Name)
			}
			if params.ContractABI == "" {
				return WriteOutput{}, fmt.Errorf("contract ABI must be specified for %s", params.Name)
			}
			if params.CallContract == nil {
				return WriteOutput{}, fmt.Errorf("callContract function must be defined for %s", params.Name)
			}
			if params.IsAllowedCaller == nil {
				return WriteOutput{}, fmt.Errorf("isAllowedCaller function must be defined for %s", params.Name)
			}
			// END Validation

			allowed, err := params.IsAllowedCaller(params.Contract, &bind.CallOpts{Context: b.GetContext()}, chain.DeployerKey.From, input.Args)
			if err != nil {
				return WriteOutput{}, fmt.Errorf("failed to check if %s is an allowed caller of %s against %s on %s: %w", chain.DeployerKey.From, params.Name, params.Contract.Address(), chain, err)
			}
			opts := deployment.SimTransactOpts()
			if allowed {
				opts = transactOptsWithGasOverrides(
					chain.DeployerKey,
					input.GasLimit,
					input.GasPrice,
					evm.GasLimitBufferBpsFromClient(chain.Client),
				)
			}
			var execInfo *ExecInfo
			tx, callErr := params.CallContract(params.Contract, opts, input.Args)
			if callErr == nil && tx == nil {
				return WriteOutput{}, fmt.Errorf("contract call returned nil transaction for %s against %s on %s", params.Name, params.Contract.Address(), chain)
			}
			if allowed {
				// If the call has actually been sent, we need check the call error and confirm the transaction.
				_, confirmErr := deployment.ConfirmIfNoErrorWithABI(chain, tx, params.ContractABI, callErr)
				if confirmErr != nil {
					return WriteOutput{}, fmt.Errorf("failed to confirm %s tx against %s on %s with args %+v: %w", params.Name, params.Contract.Address(), chain, input.Args, confirmErr)
				}
				execInfo = &ExecInfo{Hash: tx.Hash().Hex()}
				b.Logger.Debugw(fmt.Sprintf("Confirmed %s tx against %s on %s", params.Name, params.Contract.Address(), chain), "hash", tx.Hash().Hex(), "args", input.Args)
			} else if callErr != nil {
				// If we didn't execute the transaction, but there was an error preparing it, return the error.
				return WriteOutput{}, fmt.Errorf("failed to prepare %s tx against %s on %s with args %+v: %w", params.Name, params.Contract.Address(), chain, input.Args, callErr)
			} else {
				b.Logger.Debugw(fmt.Sprintf("Prepared %s tx against %s on %s", params.Name, params.Contract.Address(), chain), "args", input.Args)
			}

			return WriteOutput{
				ChainSelector: chain.Selector,
				ExecInfo:      execInfo,
				Tx: mcms_types.Transaction{
					OperationMetadata: mcms_types.OperationMetadata{
						ContractType: string(params.ContractType),
					},
					To:               params.Contract.Address().Hex(),
					Data:             tx.Data(),
					AdditionalFields: json.RawMessage(`{"value": 0}`),
				},
			}, nil
		},
	)
}

type ownableContract interface {
	Address() common.Address
	Owner(opts *bind.CallOpts) (common.Address, error)
}

// RetryContractCall retries contract read calls that can briefly fail after
// deployment while RPC nodes catch up.
func RetryContractCall[T any](
	opts *bind.CallOpts,
	waitLabel string,
	failureLabel string,
	contractAddress common.Address,
	check func() (T, error),
) (T, error) {
	// Retry with timeout to handle testnet flakiness where a newly deployed contract
	// may not be immediately visible to the RPC node
	const (
		timeout    = 5 * time.Second
		retryDelay = 500 * time.Millisecond
	)

	ctx := context.Background()
	if opts != nil && opts.Context != nil {
		ctx = opts.Context
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	var zero T

	for time.Now().Before(deadline) {
		result, err := check()
		if err == nil {
			return result, nil
		}

		// Check if this is a "contract not found" type error (empty response)
		// These errors typically contain "attempting to unmarshal an empty string"
		if strings.Contains(err.Error(), "empty string") || strings.Contains(err.Error(), "no contract code") {
			lastErr = err
			select {
			case <-ctx.Done():
				return zero, fmt.Errorf("context cancelled while waiting for %s of %s: %w", waitLabel, contractAddress, ctx.Err())
			case <-time.After(retryDelay):
			}

			continue
		}

		// For other errors, fail immediately
		return zero, fmt.Errorf("failed to %s of %s: %w", failureLabel, contractAddress, err)
	}

	return zero, fmt.Errorf("failed to %s of %s after %v: %w", failureLabel, contractAddress, timeout, lastErr)
}

func OnlyOwner[C ownableContract, ARGS any](contract C, opts *bind.CallOpts, caller common.Address, args ARGS) (bool, error) {
	owner, err := RetryContractCall(opts, "owner", "get owner", contract.Address(), func() (common.Address, error) {
		return contract.Owner(opts)
	})
	if err != nil {
		return false, err
	}

	return owner == caller, nil
}

type AccessControlContract interface {
	Address() common.Address
	HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error)
}

// HasRole reports whether account holds role on contract (OpenZeppelin IAccessControl-style HasRole).
// Includes retries for RPC flakiness after deploy.
func HasRole[C AccessControlContract](
	contract C,
	opts *bind.CallOpts,
	role [32]byte,
	account common.Address,
) (bool, error) {
	return RetryContractCall(opts, "role check", "check role", contract.Address(), func() (bool, error) {
		return contract.HasRole(opts, role, account)
	})
}

type AuthorizedCallersContract interface {
	Address() common.Address
	GetAllAuthorizedCallers(opts *bind.CallOpts) ([]common.Address, error)
}

// IsAuthorizedCaller returns whether caller is present in the contract's authorized caller set.
func IsAuthorizedCaller[C AuthorizedCallersContract](
	contract C,
	opts *bind.CallOpts,
	caller common.Address,
) (bool, error) {
	return RetryContractCall(opts, "authorized caller check", "check authorized caller", contract.Address(), func() (bool, error) {
		callers, err := contract.GetAllAuthorizedCallers(opts)
		if err != nil {
			return false, err
		}

		return slices.Contains(callers, caller), nil
	})
}

func AllCallersAllowed[C any, ARGS any](contract C, opts *bind.CallOpts, caller common.Address, args ARGS) (bool, error) {
	return true, nil
}

// NoCallersAllowed always returns false, forcing the write to be collected for a proposal
// rather than executed directly. Use this for operations that must execute atomically
// within an MCMS proposal alongside other owner-gated operations.
func NoCallersAllowed[C any, ARGS any](contract C, opts *bind.CallOpts, caller common.Address, args ARGS) (bool, error) {
	return false, nil
}

// NewBatchOperationFromWrites constructs an MCMS BatchOperation from a slice of WriteOutputs.
// It filters out any WriteOutputs that have already been executed.
// Returns an error if the WriteOutputs target multiple chains.
// If all WriteOutputs are executed, it returns an empty BatchOperation and no error.
func NewBatchOperationFromWrites(outs []WriteOutput) (mcms_types.BatchOperation, error) {
	if len(outs) == 0 {
		return mcms_types.BatchOperation{}, nil
	}

	var (
		chainSelector uint64
		txs           []mcms_types.Transaction
	)
	for _, out := range outs {
		if out.Executed() {
			continue // Skip executed transactions, they should not be included.
		}
		if len(txs) == 0 {
			chainSelector = out.ChainSelector
			txs = append(txs, out.Tx)

			continue
		}
		if out.ChainSelector != chainSelector {
			return mcms_types.BatchOperation{}, errors.New("failed to make batch operation: writes target multiple chains")
		}
		txs = append(txs, out.Tx)
	}

	// If there are no unexecuted writes, return an empty BatchOperation.
	if len(txs) == 0 {
		return mcms_types.BatchOperation{}, nil
	}

	return mcms_types.BatchOperation{
		ChainSelector: mcms_types.ChainSelector(chainSelector),
		Transactions:  txs,
	}, nil
}
