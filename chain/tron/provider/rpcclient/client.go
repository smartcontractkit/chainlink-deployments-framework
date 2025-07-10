package rpcclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
)

// confirmConfig defines the configuration for confirming transactions.
type confirmConfig struct {
	// RetryAttempts sets a fixed number of attempts for confirming transactions.
	// This is used specifically for confirmation retries.
	RetryAttempts uint
	// RetryDelay is the duration to wait between retry attempts.
	RetryDelay time.Duration
}

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
	Client   *client.GrpcClient
	Keystore *keystore.KeyStore
	Account  keystore.Account
}

// New creates a new Client instance with the provided Tron RPC client, keystore, and account.
func New(client *client.GrpcClient, keystore *keystore.KeyStore, account keystore.Account) *Client {
	return &Client{
		Client:   client,
		Keystore: keystore,
		Account:  account,
	}
}

// SendAndConfirmTx builds, signs, sends, and confirms a transaction.
// It applies any provided options for retries (delay and attempts) and returns the transaction receipt
func (c *Client) SendAndConfirmTx(
	ctx context.Context,
	tx *api.TransactionExtention,
	opts ...cldf_tron.ConfirmRetryOptions,
) (*core.TransactionInfo, error) {
	// Initialize the configuration with defaults or provided options.
	option := cldf_tron.DefaultConfirmRetryOptions()
	if len(opts) > 0 {
		option = opts[0]
	}

	signedTx, err := c.Keystore.SignTx(c.Account, tx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	_, err = c.Client.Broadcast(signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Confirm the transaction
	return c.confirmTx(ctx, tx.Txid, ConfirmRetryOpts(ctx, option)...)
}

// confirmTx checks the transaction receipt by its ID, retrying until it is confirmed or fails.
func (c *Client) confirmTx(
	ctx context.Context,
	txsig []byte,
	retryOpts ...retry.Option,
) (*core.TransactionInfo, error) {
	var receipt *core.TransactionInfo
	txID := hex.EncodeToString(txsig)

	err := retry.Do(func() error {
		var err error
		receipt, err = c.Client.GetTransactionInfoByID(txID)
		if err != nil {
			// Still not found or confirmed
			return fmt.Errorf("error fetching tx info: %w", err)
		}

		switch result := receipt.GetReceipt().GetResult(); result {
		case core.Transaction_Result_SUCCESS:
			return nil
		case core.Transaction_Result_DEFAULT, core.Transaction_Result_UNKNOWN:
			return fmt.Errorf("transaction %s is not yet confirmed, result: %v", txID, result) // Retry
		default:
			return retry.Unrecoverable(fmt.Errorf("transaction %s failed, result: %v", txID, result)) // Fail
		}
	}, retryOpts...)

	if err != nil {
		return receipt, fmt.Errorf("error confirming transaction: %w", err)
	}

	return receipt, err
}
