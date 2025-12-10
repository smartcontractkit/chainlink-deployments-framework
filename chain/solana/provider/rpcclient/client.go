package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	sollib "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

// sendConfig defines the configuration for sending transactions.
type sendConfig struct {
	// RetryAttempts determines how many times to retry sending transactions. This applies to all
	// underlying RPC client calls. If set to 0, retries will continue indefinitely until success.
	RetryAttempts uint
	// RetryDelay is the duration to wait between retry attempts.
	RetryDelay time.Duration
	// ConfirmRetryAttempts sets a fixed number of attempts for confirming transactions.
	// This is used specifically for confirmation retries, independent of RetryAttempts and is not
	// configurable by the user.
	ConfirmRetryAttempts uint
	// TxModifiers is a slice of functions that modify the transaction before sending.
	// These can be used to add signers, set compute unit limits, adjust fees, etc.
	TxModifiers []TxModifier
	// Commitment specifies the desired commitment level for the transaction.
	// Currently set to Confirmed and not user-configurable, but may be made adjustable in the future.
	Commitment solrpc.CommitmentType
}

// RetryOpts returns the retry options for sending transactions.
func (c *sendConfig) RetryOpts(ctx context.Context) []retry.Option {
	return []retry.Option{
		retry.Context(ctx),
		retry.Attempts(c.RetryAttempts),
		retry.Delay(c.RetryDelay),
		retry.DelayType(retry.FixedDelay),
	}
}

// ConfirmRetryOpts returns the retry options for confirming transactions.
func (c *sendConfig) ConfirmRetryOpts(ctx context.Context) []retry.Option {
	return []retry.Option{
		retry.Context(ctx),
		retry.Attempts(c.ConfirmRetryAttempts),
		retry.Delay(c.RetryDelay),
		retry.DelayType(retry.FixedDelay),
	}
}

// sendAndConfirmConfigDefault provides a default configuration for sending and confirming
// transactions.
var sendAndConfirmConfigDefault = sendConfig{
	RetryAttempts:        1,
	RetryDelay:           50 * time.Millisecond,
	ConfirmRetryAttempts: 500,
	TxModifiers:          make([]TxModifier, 0),
	Commitment:           solrpc.CommitmentConfirmed,
}

// SendOpt is a functional option type that allows for configuring Send operations.
type SendOpt func(*sendConfig)

// WithRetry sets the number of retry attempts and the delay between retries for sending transactions.
func WithRetry(attempts uint, delay time.Duration) SendOpt {
	return func(config *sendConfig) {
		config.RetryAttempts = attempts
		config.RetryDelay = delay
	}
}

// WithTxModifiers allows adding transaction modifiers to the send configuration.
func WithTxModifiers(modifiers ...TxModifier) SendOpt {
	return func(config *sendConfig) {
		config.TxModifiers = append(config.TxModifiers, modifiers...)
	}
}

// Client is a wrapper around the solana RPC client that provides additional functionality
// such as sending transactions with lookup tables and handling retries.
type Client struct {
	*solrpc.Client

	DeployerKey sollib.PrivateKey
}

// New creates a new Client instance with the provided Solana RPC client and deployer's private
// key.
func New(client *solrpc.Client, deployerKey sollib.PrivateKey) *Client {
	return &Client{
		Client:      client,
		DeployerKey: deployerKey,
	}
}

// SendAndConfirmTx builds, signs, sends, and confirms a transaction using the given instructions.
// It applies any provided options for retries and transaction modification, fetches the latest blockhash,
// signs with the deployer's key, and waits for the transaction to be confirmed.
func (c *Client) SendAndConfirmTx(
	ctx context.Context,
	instructions []sollib.Instruction,
	opts ...SendOpt,
) (*solrpc.GetTransactionResult, error) {
	// Initialize the configuration with defaults or provided options.
	config := sendAndConfirmConfigDefault
	for _, opt := range opts {
		opt(&config)
	}

	// Fetch the latest blockhash to use in the transaction.
	hashRes, err := c.getLatestBlockhash(ctx, config.Commitment, config.RetryOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("error getting latest blockhash: %w", err)
	}

	// Construct the transaction with the blockhash and instructions
	tx, err := c.newTx(hashRes.Value.Blockhash, instructions)
	if err != nil {
		return nil, fmt.Errorf("error constructing transaction: %w", err)
	}

	// Build the signers map
	signers := map[sollib.PublicKey]sollib.PrivateKey{}
	signers[c.DeployerKey.PublicKey()] = c.DeployerKey

	// Apply TxModifiers to the transaction.
	for _, o := range config.TxModifiers {
		if err = o(tx, signers); err != nil {
			return nil, err
		}
	}

	// Sign the transaction
	if _, err = tx.Sign(func(pub sollib.PublicKey) *sollib.PrivateKey {
		// We know that the deployer key is always present in the signers map,
		// and is inserted into the transaction in `newTx`, so we can safely
		// retrieve it without checking for existence.
		priv := signers[pub]

		return &priv
	}); err != nil {
		return nil, err
	}

	// Send the transaction
	txsig, err := c.sendTx(ctx, tx, solrpc.TransactionOpts{
		SkipPreflight:       false, // Do not skipPreflight since it is expected to pass, preflight can help debug
		PreflightCommitment: config.Commitment,
	}, config.RetryOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("error sending transaction: %w", err)
	}

	// Confirm the transaction
	err = c.confirmTx(ctx, txsig, config.ConfirmRetryOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("error confirming transaction: %w", err)
	}

	// Get the transaction result
	transactionRes, err := c.getTransactionResult(ctx, txsig, config.Commitment, config.RetryOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("error getting transaction result: %w", err)
	}

	return transactionRes, nil
}

// newTx constructs a new Solana transaction with the provided recent blockhash and instructions.
// It does not include any lookup tables, but this can be extended in the future if needed.
func (c *Client) newTx(
	recentBlockHash sollib.Hash,
	instructions []sollib.Instruction,
) (*sollib.Transaction, error) {
	// No lookup tables required, can be made configurable later if needed
	lookupTables := sollib.TransactionAddressTables(
		map[sollib.PublicKey]sollib.PublicKeySlice{},
	)

	// Construct the transaction with the provided instructions and blockhash.
	return sollib.NewTransaction(
		instructions,
		recentBlockHash,
		lookupTables,
		sollib.TransactionPayer(c.DeployerKey.PublicKey()),
	)
}

// getLatestBlockhash fetches the latest blockhash from the Solana RPC client, retrying if
// necessary based on the provided retry options.
// It retries fetching the signature status based on the provided retry options.
func (c *Client) getLatestBlockhash(
	ctx context.Context, commitment solrpc.CommitmentType, retryOpts ...retry.Option,
) (*solrpc.GetLatestBlockhashResult, error) {
	var result *solrpc.GetLatestBlockhashResult

	err := retry.Do(func() error {
		var rerr error

		result, rerr = c.GetLatestBlockhash(ctx, commitment)

		return rerr
	}, retryOpts...)

	return result, err
}

// sendTx sends a transaction to the Solana network using the provided transaction options.
// It retries fetching the signature status based on the provided retry options.
func (c *Client) sendTx(
	ctx context.Context,
	tx *sollib.Transaction,
	txOpts solrpc.TransactionOpts,
	retryOpts ...retry.Option,
) (sollib.Signature, error) {
	var txsig sollib.Signature

	err := retry.Do(func() error {
		var rerr error

		txsig, rerr = c.SendTransactionWithOpts(ctx, tx, txOpts)
		if rerr != nil {
			// Handle specific RPC errors
			var rpcErr *jsonrpc.RPCError
			if errors.As(rerr, &rpcErr) {
				if strings.Contains(rpcErr.Message, "Blockhash not found") {
					// this can happen when the blockhash we retrieved above is not yet visible to
					// the rpc given we get the blockhash from the same rpc, this should not
					// happen, but we see it in practice. We attempt to retry to see if it
					// resolves.
					return fmt.Errorf("blockhash not found, retrying: %w", rerr)
				}

				return retry.Unrecoverable(
					fmt.Errorf("unexpected error (most likely contract related), will not retry: %w", rerr),
				)
			}

			// Not an RPC error — should only happen when we fail to hit the rpc service
			return fmt.Errorf("unexpected error (could not hit rpc service): %w", rerr)
		}

		return nil
	}, retryOpts...)

	return txsig, err
}

// confirmTx checks the status of a transaction signature until it is confirmed or finalized.
// It retries fetching the signature status based on the provided retry options.
func (c *Client) confirmTx(
	ctx context.Context,
	txsig sollib.Signature,
	retryOpts ...retry.Option,
) error {
	var status solrpc.ConfirmationStatusType

	return retry.Do(func() error {
		// Success
		if status == solrpc.ConfirmationStatusConfirmed || status == solrpc.ConfirmationStatusFinalized {
			return nil
		}

		statusRes, err := c.GetSignatureStatuses(ctx, true, txsig)
		if err != nil {
			// Retry if we hit an error fetching the signature status. Mainnet can be flakey.
			return err
		}

		if statusRes != nil && len(statusRes.Value) > 0 && statusRes.Value[0] != nil {
			status = statusRes.Value[0].ConfirmationStatus
		}

		return nil
	}, retryOpts...)
}

// getTransactionResult retrieves the result of a transaction by its signature.
// It retries fetching the transaction result based on the provided retry options.
func (c *Client) getTransactionResult(
	ctx context.Context,
	txsig sollib.Signature,
	commitment solrpc.CommitmentType,
	retryOpts ...retry.Option,
) (*solrpc.GetTransactionResult, error) {
	ver := uint64(0)

	var result *solrpc.GetTransactionResult
	err := retry.Do(func() error {
		var rerr error

		result, rerr = c.GetTransaction(ctx, txsig, &solrpc.GetTransactionOpts{
			Commitment:                     commitment,
			MaxSupportedTransactionVersion: &ver,
		})

		return rerr
	}, retryOpts...)

	return result, err
}
