package rpcclient

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/avast/retry-go/v4"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/keystore"

	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
)

// confirmRetryOpts returns the retry options for confirming transactions.
// It wraps a context and sets retry count and delay based on provided config.
func confirmRetryOpts(ctx context.Context, c *tron.ConfirmRetryOptions) []retry.Option {
	return []retry.Option{
		retry.Context(ctx),
		retry.Attempts(c.RetryAttempts),
		retry.Delay(c.RetryDelay),
		retry.DelayType(retry.FixedDelay),
	}
}

// Client is a wrapper around the Tron RPC client that provides additional functionality
// such as signing, sending, and confirming transactions. It abstracts signing logic
// and retry-based confirmation flows.
type Client struct {
	Client   sdk.FullNodeClient // Underlying Tron full node client
	Keystore *keystore.Keystore // Keystore used to sign transactions
	Account  address.Address    // Address used for signing transactions
}

// New creates a new Client instance with the provided Tron RPC client, keystore, and account.
// This prepares the client to sign and send transactions using the given identity.
func New(client sdk.FullNodeClient, keystore *keystore.Keystore, account address.Address) *Client {
	return &Client{
		Client:   client,
		Keystore: keystore,
		Account:  account,
	}
}

// SendAndConfirmTx builds, signs, sends, and confirms a transaction.
// It applies any provided options for retries (delay and attempts) and returns the transaction info.
// This is the main entry point for broadcasting a transaction and ensuring it is mined successfully.
func (c *Client) SendAndConfirmTx(
	ctx context.Context,
	tx *common.Transaction,
	opts *tron.ConfirmRetryOptions,
) (*soliditynode.TransactionInfo, error) {
	// Initialize the configuration with defaults or provided options.
	option := tron.DefaultConfirmRetryOptions()
	if opts != nil {
		option = opts
	}

	// Decode the TxID from hex into bytes, required for signing
	txIdBytes, err := hex.DecodeString(tx.TxID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	// Sign the transaction using the client's keystore and the decoded TxID
	signature, err := c.Keystore.Sign(context.Background(), c.Account.String(), txIdBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Attach the signature to the transaction
	tx.AddSignatureBytes(signature)

	// Broadcast the signed transaction to the Tron network
	broadcastResponse, err := c.Client.BroadcastTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Confirm the transaction by polling for success/failure based on the TxID
	return c.confirmTx(broadcastResponse.TxID, confirmRetryOpts(ctx, option)...)
}

// confirmTx checks the transaction receipt by its ID, retrying until it is confirmed or fails.
// It uses the retry-go library with the configured retry options.
func (c *Client) confirmTx(
	txId string,
	retryOpts ...retry.Option,
) (*soliditynode.TransactionInfo, error) {
	var receipt *soliditynode.TransactionInfo

	// Retry loop to poll for transaction confirmation
	err := retry.Do(func() error {
		var err error
		receipt, err = c.Client.GetTransactionInfoById(txId)
		if err != nil {
			// Still not found or confirmed, continue retrying
			return fmt.Errorf("error fetching tx info: %w", err)
		}

		// Check the transaction result status
		//nolint:exhaustive // handled via default case
		switch result := receipt.Receipt.Result; result {
		// An empty result or success indicates confirmation
		// TODO: investigate why when confirming a non-contract transaction, the result is empty (see SendTrxWithSendAndConfirm test)
		case "", soliditynode.TransactionResultSuccess:
			return nil
		case soliditynode.TransactionResultDefault, soliditynode.TransactionResultUnknown:
			// Keep retrying until status changes
			return fmt.Errorf("transaction %s is not yet confirmed, result: %v", txId, result)
		default:
			// Any other status means unrecoverable failure
			return retry.Unrecoverable(fmt.Errorf("transaction %s failed, result: %v", txId, result))
		}
	}, retryOpts...)

	if err != nil {
		// Return the last receipt (if any) and error for debugging
		return receipt, fmt.Errorf("error confirming transaction: %w", err)
	}

	return receipt, err
}

// CheckContractDeployed checks if a contract is deployed at the given address.
// This ensures the contract code exists at the address before interacting with it.
func (c *Client) CheckContractDeployed(address address.Address) error {
	_, err := c.Client.GetContract(address)
	if err != nil {
		return fmt.Errorf("error checking contract deployment: %w", err)
	}

	return nil
}
