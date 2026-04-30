package tron

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// ChainMetadata = generic metadata from the framework
type ChainMetadata = chaincommon.ChainMetadata

type ConfirmRetryOptions struct {
	RetryAttempts uint          // Max number of retries for confirming a transaction.
	RetryDelay    time.Duration // Delay between retries for confirming a transaction.
}

// DeployOptions defines optional parameters for deploying a smart contract.
type DeployOptions struct {
	OeLimit             int                  // Max energy the creator is willing to provide during execution.
	CurPercent          int                  // Percentage of resource consumption charged to the contract caller (0–100).
	FeeLimit            int                  // Max TRX to be used for deploying the contract (gas limit in Tron terms).
	ConfirmRetryOptions *ConfirmRetryOptions // Retry options for confirming the transaction.
}

// TriggerOptions defines optional parameters for triggering (calling) a smart contract.
type TriggerOptions struct {
	FeeLimit            int32                // Max TRX to be used for this transaction call.
	TAmount             int64                // Amount of TRX to transfer along with the contract call (like msg.value).
	ConfirmRetryOptions *ConfirmRetryOptions // Retry options for confirming the transaction.
}

type RPCClient interface {
	SendAndConfirmTx(ctx context.Context, tx *common.Transaction,
		opts *ConfirmRetryOptions) (*soliditynode.TransactionInfo, error)
	CheckContractDeployed(address address.Address) error
	WithSignHash(fn func(ctx context.Context, txHash []byte) ([]byte, error)) RPCClient
}

// Chain represents a Tron chain
type Chain struct {
	ChainMetadata                                                          // Chain selector and metadata
	Client        sdk.CombinedClient                                       // Combined client for Tron operations
	SignHash      func(ctx context.Context, txHash []byte) ([]byte, error) // Function for signing transaction hashes
	Address       address.Address                                          // Address of the account used for transactions
	URL           string                                                   // Optional: Client URL
	RPCClient     RPCClient

	// SendAndConfirm provides a utility function to send a transaction and waits for confirmation.
	SendAndConfirm func(ctx context.Context, tx *common.Transaction, opts *ConfirmRetryOptions) (*soliditynode.TransactionInfo, error)

	// DeployContractAndConfirm provides a utility function to deploy a contract and waits for confirmation.
	DeployContractAndConfirm func(
		ctx context.Context, contractName string, abi string, bytecode string, params []interface{}, opts *DeployOptions,
	) (address.Address, *soliditynode.TransactionInfo, error)

	// TriggerContractAndConfim provides a utility function to send a contract transaction and waits for confirmation.
	TriggerContractAndConfirm func(
		ctx context.Context, contractAddr address.Address, functionName string, params []interface{}, opts *TriggerOptions,
	) (*soliditynode.TransactionInfo, error)
}

// DefaultConfirmRetryOptions returns standard retry options used across contract deployment and invocation.
// Defaults to 180 retries with a 500ms delay between each attempt.
func DefaultConfirmRetryOptions() *ConfirmRetryOptions {
	return &ConfirmRetryOptions{
		RetryAttempts: 180,
		RetryDelay:    500 * time.Millisecond,
	}
}

// DefaultDeployOptions returns default options used when deploying a contract.
// It includes a high fee and energy limit suitable for development/testing, and standard retry behavior.
func DefaultDeployOptions() *DeployOptions {
	return &DeployOptions{
		FeeLimit:            100_000_000, // Default fee limit (in SUN).
		CurPercent:          100,         // Caller pays full cost.
		OeLimit:             50_000_000,  // Default energy limit.
		ConfirmRetryOptions: DefaultConfirmRetryOptions(),
	}
}

// DefaultTriggerOptions returns default options for calling smart contract methods.
// These defaults ensure calls succeed on local/dev environments without TRX transfer.
func DefaultTriggerOptions() *TriggerOptions {
	return &TriggerOptions{
		FeeLimit:            10_000_000, // Default fee limit (in SUN).
		TAmount:             0,          // No TRX transferred by default.
		ConfirmRetryOptions: DefaultConfirmRetryOptions(),
	}
}

func (c Chain) ReadOnly() (chaincommon.BlockChain, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key for read-only chain %v: %w", c, err)
	}

	c.Address = address.PubkeyToAddress(privateKey.PublicKey)
	c.SignHash = func(ctx context.Context, txHash []byte) ([]byte, error) {
		return crypto.Sign(txHash, privateKey)
	}
	c.RPCClient = c.RPCClient.WithSignHash(c.SignHash)
	c.SendAndConfirm = func(ctx context.Context, tx *common.Transaction, opts *ConfirmRetryOptions) (*soliditynode.TransactionInfo, error) {
		options := DefaultConfirmRetryOptions()
		if opts != nil {
			options = opts
		}

		// Send transaction and wait for confirmation
		return c.RPCClient.SendAndConfirmTx(ctx, tx, options)
	}
	c.DeployContractAndConfirm = func(
		ctx context.Context, contractName string, abi string, bytecode string, params []any, opts *DeployOptions,
	) (address.Address, *soliditynode.TransactionInfo, error) {
		options := DefaultDeployOptions()
		if opts != nil {
			options = opts
		}

		// Create deploy contract transaction
		deployResponse, err := c.Client.DeployContract(
			c.Address, contractName, abi, bytecode, options.OeLimit, options.CurPercent, options.FeeLimit, params,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
		}

		// Send transaction and wait for confirmation
		txInfo, err := c.RPCClient.SendAndConfirmTx(ctx, &deployResponse.Transaction, options.ConfirmRetryOptions)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to confirm deploy contract transaction: %w", err)
		}

		// Parse resulting contract address
		contractAddr, err := address.StringToAddress(txInfo.ContractAddress)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse contract address: %w", err)
		}

		// Ensure contract is actually deployed on-chain
		if err := c.RPCClient.CheckContractDeployed(contractAddr); err != nil {
			return nil, nil, fmt.Errorf("contract deployment check failed: %w", err)
		}

		return contractAddr, txInfo, nil
	}
	c.TriggerContractAndConfirm = func(
		ctx context.Context, contractAddr address.Address, functionName string, params []any, opts *TriggerOptions,
	) (*soliditynode.TransactionInfo, error) {
		options := DefaultTriggerOptions()
		if opts != nil {
			options = opts
		}

		// Ensure contract is actually deployed on-chain
		if err := c.RPCClient.CheckContractDeployed(contractAddr); err != nil {
			return nil, fmt.Errorf("contract deployment check failed: %w", err)
		}

		// Create trigger contract transaction
		contractResponse, err := c.Client.TriggerSmartContract(
			c.Address, contractAddr, functionName, params, options.FeeLimit, options.TAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create trigger contract transaction: %w", err)
		}

		// Send transaction and wait for confirmation
		return c.RPCClient.SendAndConfirmTx(ctx, contractResponse.Transaction, options.ConfirmRetryOptions)
	}

	return c, nil
}
