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

	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

// ConfirmRetryOpts returns the retry options for confirming transactions.
func ConfirmRetryOpts(ctx context.Context, c cldf_tron.ConfirmRetryOptions) []retry.Option {
	return []retry.Option{
		retry.Context(ctx),
		retry.Attempts(c.RetryAttempts),
		retry.Delay(c.RetryDelay),
		retry.DelayType(retry.FixedDelay),
	}
}

// Client is a wrapper around the Tron RPC client that provides additional functionality
// such as sending and confirming transactions.
type Client struct {
	Client   sdk.FullNodeClient
	Keystore *keystore.Keystore
	Account  address.Address
}

// New creates a new Client instance with the provided Tron RPC client, keystore, and account.
func New(client sdk.FullNodeClient, keystore *keystore.Keystore, account address.Address) *Client {
	return &Client{
		Client:   client,
		Keystore: keystore,
		Account:  account,
	}
}

// SendAndConfirmTx builds, signs, sends, and confirms a transaction.
// It applies any provided options for retries (delay and attempts) and returns the transaction info
func (c *Client) SendAndConfirmTx(
	ctx context.Context,
	tx *common.Transaction,
	opts ...cldf_tron.ConfirmRetryOptions,
) (*soliditynode.TransactionInfo, error) {
	// Initialize the configuration with defaults or provided options.
	option := cldf_tron.DefaultConfirmRetryOptions()
	if len(opts) > 0 {
		option = opts[0]
	}

	txIdBytes, err := hex.DecodeString(tx.TxID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	signature, err := c.Keystore.Sign(context.Background(), c.Account.String(), txIdBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	tx.AddSignatureBytes(signature)

	broadcastResponse, err := c.Client.BroadcastTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Confirm the transaction
	return c.confirmTx(broadcastResponse.TxID, ConfirmRetryOpts(ctx, option)...)
}

// confirmTx checks the transaction receipt by its ID, retrying until it is confirmed or fails.
func (c *Client) confirmTx(
	txId string,
	retryOpts ...retry.Option,
) (*soliditynode.TransactionInfo, error) {
	var receipt *soliditynode.TransactionInfo

	err := retry.Do(func() error {
		var err error
		receipt, err = c.Client.GetTransactionInfoById(txId)
		if err != nil {
			// Still not found or confirmed
			return fmt.Errorf("error fetching tx info: %w", err)
		}

		//nolint:exhaustive // handled via default case
		switch result := receipt.Receipt.Result; result {
		case soliditynode.TransactionResultSuccess:
			return nil
		case soliditynode.TransactionResultDefault, soliditynode.TransactionResultUnknown:
			return fmt.Errorf("transaction %s is not yet confirmed, result: %v", txId, result) // Retry
		default:
			return retry.Unrecoverable(fmt.Errorf("transaction %s failed, result: %v", txId, result)) // Fail
		}
	}, retryOpts...)

	if err != nil {
		return receipt, fmt.Errorf("error confirming transaction: %w", err)
	}

	return receipt, err
}

func (c *Client) CheckContractDeployed(address address.Address) error {
	_, err := c.Client.GetContract(address)
	if err != nil {
		return fmt.Errorf("error checking contract deployment: %w", err)
	}

	return nil
}
