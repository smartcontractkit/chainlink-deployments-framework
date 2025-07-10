package tron

import (
	"context"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// ChainMetadata = generic metadata from the framework
type ChainMetadata = common.ChainMetadata

// DeployOptions defines optional parameters for deploying a smart contract.
type DeployOptions struct {
	FeeLimit    int64 // Max TRX to be used for deploying the contract (gas limit in TRON terms).
	CurPercent  int64 // Percentage of resource consumption charged to the contract caller (0–100).
	EnergyLimit int64 // Max energy the creator is willing to provide during execution.
}

// TriggerOptions defines optional parameters for triggering (calling) a smart contract.
type TriggerOptions struct {
	FeeLimit     int64  // Max TRX to be used for this transaction call.
	TAmount      int64  // Amount of TRX to transfer along with the contract call (like msg.value).
	TTokenID     string // (Optional) TRC-10 token ID to transfer with the call.
	TTokenAmount int64  // Amount of the TRC-10 token to send with the call.
}

// Chain represents a TRON chain
type Chain struct {
	ChainMetadata                    // Chain selector and metadata
	Client        *client.GrpcClient // gRPC client to TRON full node
	Keystore      *keystore.KeyStore // Keystore for managing accounts and signing transactions
	Account       keystore.Account   // Account abstraction for deployer wallet
	URL           string             // Liteserver URL
	DeployerSeed  string             // Optional: mnemonic or raw seed

	// SendAndConfirmTx provides a utility function to send a transaction and waits for confirmation.
	SendAndConfirmTx func(ctx context.Context, tx *api.TransactionExtention) (*core.TransactionInfo, error)

	// DeployContractAndConfirm provides a utility function to deploy a contract and waits for confirmation.
	DeployContractAndConfirm func(
		ctx context.Context, contractName string, abi *core.SmartContract_ABI, bytecode string, opts ...DeployOptions,
	) (*core.TransactionInfo, error)

	// TriggerContractAndConfim provides a utility function to send a transaction and waits for confirmation.
	TriggerContractAndConfirm func(
		ctx context.Context, contractAddr eth_common.Address, functionName string, jsonParams string, opts ...TriggerOptions,
	) (*core.TransactionInfo, error)
}
