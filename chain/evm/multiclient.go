package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

const (
	// Default retry configuration for RPC calls
	RPCDefaultRetryAttempts = 1
	RPCDefaultRetryDelay    = 1000 * time.Millisecond
	RPCDefaultRetryTimeout  = 10 * time.Second

	// Default retry configuration for dialing RPC endpoints
	RPCDefaultDialRetryAttempts = 1
	RPCDefaultDialRetryDelay    = 1000 * time.Millisecond
	RPCDefaultDialTimeout       = 10 * time.Second

	// Default timeout for health checks
	RPCDefaultHealthCheckTimeout = 2 * time.Second
)

type RetryConfig struct {
	Attempts     uint
	Delay        time.Duration
	Timeout      time.Duration
	DialAttempts uint
	DialDelay    time.Duration
	DialTimeout  time.Duration
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		Attempts:     RPCDefaultRetryAttempts,
		Delay:        RPCDefaultRetryDelay,
		Timeout:      RPCDefaultRetryTimeout,
		DialAttempts: RPCDefaultDialRetryAttempts,
		DialDelay:    RPCDefaultDialRetryDelay,
		DialTimeout:  RPCDefaultDialTimeout,
	}
}

// MultiClient should comply with the OnchainClient interface
var _ OnchainClient = &MultiClient{}

type MultiClient struct {
	*ethclient.Client
	Backups     []*ethclient.Client
	RetryConfig RetryConfig
	lggr        logger.Logger
	chainName   string
	mu          sync.RWMutex
}

// rpcHealthCheck performs a basic health check on the RPC client by calling eth_blockNumber
func (mc *MultiClient) rpcHealthCheck(ctx context.Context, client *ethclient.Client) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, RPCDefaultHealthCheckTimeout)
	defer cancel()

	// Try to get the latest block number
	_, err := client.BlockNumber(timeoutCtx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

func NewMultiClient(lggr logger.Logger, rpcsCfg RPCConfig, opts ...func(client *MultiClient)) (*MultiClient, error) {
	if len(rpcsCfg.RPCs) == 0 {
		return nil, errors.New("no RPCs provided, need at least one")
	}
	// Set the chain name
	chain, exists := chainsel.ChainBySelector(rpcsCfg.ChainSelector)
	if !exists {
		return nil, fmt.Errorf("chain with selector %d not found", rpcsCfg.ChainSelector)
	}
	mc := MultiClient{lggr: lggr, chainName: chain.Name}

	mc.RetryConfig = defaultRetryConfig()

	for _, opt := range opts {
		opt(&mc)
	}

	clients := make([]*ethclient.Client, 0, len(rpcsCfg.RPCs))
	for i, rpc := range rpcsCfg.RPCs {
		client, err := mc.dialWithRetry(rpc, lggr)
		if err != nil {
			lggr.Warnf("failed to dial client %d for RPC '%s' - %s (%d), trying with the next one: %v", i, rpc.Name, chain.Name, chain.Selector, err)

			continue
		}
		if err := mc.rpcHealthCheck(context.Background(), client); err != nil {
			lggr.Warnf("health check failed for client %d for RPC '%s' - %s (%d), trying with the next one: %v", i, rpc.Name, chain.Name, chain.Selector, err)
			client.Close()

			continue
		}
		clients = append(clients, client)
	}

	if len(clients) == 0 {
		return nil, errors.New("no valid RPC clients created")
	}

	mc.Client = clients[0]
	mc.Backups = clients[1:]

	return &mc, nil
}

func (mc *MultiClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return mc.retryWithBackups(ctx, "SendTransaction", func(ct context.Context, client *ethclient.Client) error {
		return client.SendTransaction(ct, tx)
	})
}

func (mc *MultiClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var result []byte
	err := mc.retryWithBackups(ctx, "CallContract", func(ct context.Context, client *ethclient.Client) error {
		var err error
		result, err = client.CallContract(ct, msg, blockNumber)

		return err
	})

	return result, err
}

func (mc *MultiClient) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
	var result []byte
	err := mc.retryWithBackups(ctx, "CallContractAtHash", func(ct context.Context, client *ethclient.Client) error {
		var err error
		result, err = client.CallContractAtHash(ct, msg, blockHash)

		return err
	})

	return result, err
}

func (mc *MultiClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups(ctx, "CodeAt", func(ct context.Context, client *ethclient.Client) error {
		var err error
		code, err = client.CodeAt(ct, account, blockNumber)

		return err
	})

	return code, err
}

func (mc *MultiClient) CodeAtHash(ctx context.Context, account common.Address, blockHash common.Hash) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups(ctx, "CodeAtHash", func(ct context.Context, client *ethclient.Client) error {
		var err error
		code, err = client.CodeAtHash(ct, account, blockHash)

		return err
	})

	return code, err
}

func (mc *MultiClient) NonceAt(ctx context.Context, account common.Address, block *big.Int) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups(ctx, "NonceAt", func(ct context.Context, client *ethclient.Client) error {
		var err error
		count, err = client.NonceAt(ct, account, block)

		return err
	})

	return count, err
}

func (mc *MultiClient) NonceAtHash(ctx context.Context, account common.Address, blockHash common.Hash) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups(ctx, "NonceAtHash", func(ct context.Context, client *ethclient.Client) error {
		var err error
		count, err = client.NonceAtHash(ct, account, blockHash)

		return err
	})

	return count, err
}

func (mc *MultiClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var header *types.Header
	err := mc.retryWithBackups(ctx, "HeaderByNumber", func(ct context.Context, client *ethclient.Client) error {
		var err error
		header, err = client.HeaderByNumber(ct, number)

		return err
	})

	return header, err
}

func (mc *MultiClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var gasPrice *big.Int
	err := mc.retryWithBackups(ctx, "SuggestGasPrice", func(ct context.Context, client *ethclient.Client) error {
		var err error
		gasPrice, err = client.SuggestGasPrice(ct)

		return err
	})

	return gasPrice, err
}

func (mc *MultiClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	var gasTipCap *big.Int
	err := mc.retryWithBackups(ctx, "SuggestGasTipCap", func(ct context.Context, client *ethclient.Client) error {
		var err error
		gasTipCap, err = client.SuggestGasTipCap(ct)

		return err
	})

	return gasTipCap, err
}

func (mc *MultiClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups(ctx, "PendingCodeAt", func(ct context.Context, client *ethclient.Client) error {
		var err error
		code, err = client.PendingCodeAt(ct, account)

		return err
	})

	return code, err
}

func (mc *MultiClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups(ctx, "PendingNonceAt", func(ct context.Context, client *ethclient.Client) error {
		var err error
		count, err = client.PendingNonceAt(ct, account)

		return err
	})

	return count, err
}

func (mc *MultiClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	var gas uint64
	err := mc.retryWithBackups(ctx, "EstimateGas", func(ct context.Context, client *ethclient.Client) error {
		var err error
		gas, err = client.EstimateGas(ct, call)

		return err
	})

	return gas, err
}

func (mc *MultiClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var balance *big.Int
	err := mc.retryWithBackups(ctx, "BalanceAt", func(ct context.Context, client *ethclient.Client) error {
		var err error
		balance, err = client.BalanceAt(ct, account, blockNumber)

		return err
	})

	return balance, err
}

func (mc *MultiClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var logs []types.Log
	err := mc.retryWithBackups(ctx, "FilterLogs", func(ct context.Context, client *ethclient.Client) error {
		var err error
		logs, err = client.FilterLogs(ct, q)

		return err
	})

	return logs, err
}

func (mc *MultiClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := mc.retryWithBackups(ctx, "SubscribeFilterLogs", func(ct context.Context, client *ethclient.Client) error {
		var err error
		sub, err = client.SubscribeFilterLogs(ct, q, ch)

		return err
	})

	return sub, err
}

// WaitMined waits for a transaction to be mined and returns the receipt.
// Note: retryConfig timeout settings are not used for this operation, a timeout can be set in the context.
func (mc *MultiClient) WaitMined(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	mc.lggr.Debugf("Waiting for tx %s to be mined for chain %s", tx.Hash().Hex(), mc.chainName)
	// no retries here because we want to wait for the tx to be mined
	resultCh := make(chan *types.Receipt)
	doneCh := make(chan struct{})

	waitMined := func(client *ethclient.Client, tx *types.Transaction) {
		mc.lggr.Debugf("Waiting for tx %s to be mined with chain %s", tx.Hash().Hex(), mc.chainName)
		receipt, err := bind.WaitMined(ctx, client, tx)
		if err != nil {
			mc.lggr.Warnf("WaitMined error %v with chain %s", err, mc.chainName)
			return
		}
		select {
		case resultCh <- receipt:
		case <-doneCh:
			return
		}
	}

	for _, client := range mc.clients() {
		go waitMined(client, tx)
	}
	var receipt *types.Receipt
	select {
	case receipt = <-resultCh:
		close(doneCh)
		mc.lggr.Debugf("Tx %s mined with chain %s", tx.Hash().Hex(), mc.chainName)

		return receipt, nil
	case <-ctx.Done():
		mc.lggr.Warnf("WaitMined context done %v", ctx.Err())
		close(doneCh)

		return nil, ctx.Err()
	}
}

func (mc *MultiClient) retryWithBackups(ctx context.Context, opName string, op func(context.Context, *ethclient.Client) error) error {
	var err error
	traceID := uuid.New()

	for rpcIndex, client := range mc.clients() {
		retryCount := 0
		err2 := retry.Do(func() error {
			timeoutCtx, cancel := ensureTimeout(ctx, mc.RetryConfig.Timeout)
			defer cancel()

			err = op(timeoutCtx, client)
			if err != nil {
				mc.lggr.Warnf("traceID %q: chain %q: op: %q: client index %d: failed execution - retryable error '%s'", traceID.String(), mc.chainName, opName, rpcIndex, maybeDataErr(err))
				return err
			}

			// If the operation was successful, check if we need to reorder the RPCs
			mc.reorderRPCs(rpcIndex)

			return nil
		}, retry.Attempts(mc.RetryConfig.Attempts), retry.Delay(mc.RetryConfig.Delay),
			retry.OnRetry(func(n uint, err error) { retryCount++ }))
		if err2 == nil {
			if retryCount > 0 {
				mc.lggr.Infof("traceID %q: chain %q: op: %q: client index %d: successfully executed after %d retry", traceID.String(), mc.chainName, opName, rpcIndex, retryCount)
			}

			return nil
		}
		mc.lggr.Infof("traceID %q: chain %q: op: %q: client index %d: failed, trying next client", traceID.String(), mc.chainName, opName, rpcIndex)
	}

	return errors.Join(err, fmt.Errorf("all backup clients failed for chain %q", mc.chainName))
}

func (mc *MultiClient) dialWithRetry(rpc RPC, lggr logger.Logger) (*ethclient.Client, error) {
	endpoint, err := rpc.ToEndpoint()
	if err != nil {
		return nil, err
	}

	traceID := uuid.New()
	var client *ethclient.Client
	retryCount := 0
	err = retry.Do(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), mc.RetryConfig.DialTimeout)
		defer cancel()

		var err2 error
		mc.lggr.Debugf("traceID %q: chain %q: rpc: %q: dialing endpoint '%s'", traceID.String(), mc.chainName, rpc.Name, endpoint)
		client, err2 = ethclient.DialContext(ctx, endpoint)
		if err2 != nil {
			lggr.Warnf("traceID %q: chain %q: rpc: %q: dialing failed - retryable error: %s: %v", traceID.String(), mc.chainName, rpc.Name, endpoint, err2)
			return err2
		}

		return nil
	}, retry.Attempts(mc.RetryConfig.DialAttempts), retry.Delay(mc.RetryConfig.DialDelay),
		retry.OnRetry(func(n uint, err error) { retryCount++ }))

	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to dial endpoint '%s' for RPC %s for chain %s after retries", endpoint, rpc.Name, mc.chainName))
	}
	if retryCount > 0 {
		lggr.Infof("traceID %q: chain %q: rpc: %q: successfully dialed endpoint '%s' after %d retries", traceID.String(), mc.chainName, rpc.Name, endpoint, retryCount)
	}

	return client, nil
}

// ensureTimeout checks if the parent context has a deadline.
// If it does, it returns a new cancelable context using the parent's deadline.
// If it doesn't, it creates a new context with the specified timeout.
func ensureTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	// check if the parent context already has a deadline
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		// derive a new cancelable context from the parent context with the same deadline
		return context.WithCancel(parent)
	}

	// create a new context with the specified timeout
	return context.WithTimeout(parent, timeout)
}

// reorderRPCs reorders the RPCs based on the latest call.
// If the default RPC failed all attempts, it will be moved to the end of the backup list.
// If backup RPCs also failed, they will be moved to the end of the backup list.
// If the primary RPC worked, it will remain the first in the list.
func (mc *MultiClient) reorderRPCs(rpcIndex int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if rpcIndex < 1 || len(mc.Backups) == 0 {
		return // No need to reorder if the first RPC is still the default or we don't have backups
	}

	// Find the index of the backupRPC
	newDefaultRPCIndex := rpcIndex - 1
	newDefaultRPC := mc.Backups[newDefaultRPCIndex]

	// Reorder the failed backups to the end of the list
	reordered := make([]*ethclient.Client, 0, len(mc.Backups))
	reordered = append(reordered, mc.Backups[newDefaultRPCIndex+1:]...)
	reordered = append(reordered, mc.Backups[:newDefaultRPCIndex]...)
	reordered = append(reordered, mc.Client)

	mc.Backups = reordered
	mc.Client = newDefaultRPC
}

func (mc *MultiClient) clients() []*ethclient.Client {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return append([]*ethclient.Client{mc.Client}, mc.Backups...)
}

func maybeDataErr(err error) error {
	//revive:disable
	var d rpc.DataError
	ok := errors.As(err, &d)
	if ok {
		return fmt.Errorf("%s: %v", d.Error(), d.ErrorData())
	}

	return err
}
