package deployment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

const (
	// Default retry configuration for RPC calls
	RPCDefaultRetryAttempts = 1
	RPCDefaultRetryDelay    = 1000 * time.Millisecond

	// Default retry configuration for dialing RPC endpoints
	RPCDefaultDialRetryAttempts = 1
	RPCDefaultDialRetryDelay    = 1000 * time.Millisecond
)

type RetryConfig struct {
	Attempts     uint
	Delay        time.Duration
	DialAttempts uint
	DialDelay    time.Duration
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		Attempts:     RPCDefaultRetryAttempts,
		Delay:        RPCDefaultRetryDelay,
		DialAttempts: RPCDefaultDialRetryAttempts,
		DialDelay:    RPCDefaultDialRetryDelay,
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
			lggr.Warnf("failed to dial client %d for RPC '%s' trying with the next one: %v", i, rpc.Name, err)
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
	return mc.retryWithBackups("SendTransaction", func(client *ethclient.Client) error {
		return client.SendTransaction(ctx, tx)
	})
}

func (mc *MultiClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var result []byte
	err := mc.retryWithBackups("CallContract", func(client *ethclient.Client) error {
		var err error
		result, err = client.CallContract(ctx, msg, blockNumber)

		return err
	})

	return result, err
}

func (mc *MultiClient) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
	var result []byte
	err := mc.retryWithBackups("CallContractAtHash", func(client *ethclient.Client) error {
		var err error
		result, err = client.CallContractAtHash(ctx, msg, blockHash)

		return err
	})

	return result, err
}

func (mc *MultiClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups("CodeAt", func(client *ethclient.Client) error {
		var err error
		code, err = client.CodeAt(ctx, account, blockNumber)

		return err
	})

	return code, err
}

func (mc *MultiClient) CodeAtHash(ctx context.Context, account common.Address, blockHash common.Hash) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups("CodeAtHash", func(client *ethclient.Client) error {
		var err error
		code, err = client.CodeAtHash(ctx, account, blockHash)

		return err
	})

	return code, err
}

func (mc *MultiClient) NonceAt(ctx context.Context, account common.Address, block *big.Int) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups("NonceAt", func(client *ethclient.Client) error {
		var err error
		count, err = client.NonceAt(ctx, account, block)

		return err
	})

	return count, err
}

func (mc *MultiClient) NonceAtHash(ctx context.Context, account common.Address, blockHash common.Hash) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups("NonceAtHash", func(client *ethclient.Client) error {
		var err error
		count, err = client.NonceAtHash(ctx, account, blockHash)

		return err
	})

	return count, err
}

func (mc *MultiClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var header *types.Header
	err := mc.retryWithBackups("HeaderByNumber", func(client *ethclient.Client) error {
		var err error
		header, err = client.HeaderByNumber(ctx, number)

		return err
	})

	return header, err
}

func (mc *MultiClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var gasPrice *big.Int
	err := mc.retryWithBackups("SuggestGasPrice", func(client *ethclient.Client) error {
		var err error
		gasPrice, err = client.SuggestGasPrice(ctx)

		return err
	})

	return gasPrice, err
}

func (mc *MultiClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	var gasTipCap *big.Int
	err := mc.retryWithBackups("SuggestGasTipCap", func(client *ethclient.Client) error {
		var err error
		gasTipCap, err = client.SuggestGasTipCap(ctx)

		return err
	})

	return gasTipCap, err
}

func (mc *MultiClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var code []byte
	err := mc.retryWithBackups("PendingCodeAt", func(client *ethclient.Client) error {
		var err error
		code, err = client.PendingCodeAt(ctx, account)

		return err
	})

	return code, err
}

func (mc *MultiClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var count uint64
	err := mc.retryWithBackups("PendingNonceAt", func(client *ethclient.Client) error {
		var err error
		count, err = client.PendingNonceAt(ctx, account)

		return err
	})

	return count, err
}

func (mc *MultiClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	var gas uint64
	err := mc.retryWithBackups("EstimateGas", func(client *ethclient.Client) error {
		var err error
		gas, err = client.EstimateGas(ctx, call)

		return err
	})

	return gas, err
}

func (mc *MultiClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var balance *big.Int
	err := mc.retryWithBackups("BalanceAt", func(client *ethclient.Client) error {
		var err error
		balance, err = client.BalanceAt(ctx, account, blockNumber)

		return err
	})

	return balance, err
}

func (mc *MultiClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var logs []types.Log
	err := mc.retryWithBackups("FilterLogs", func(client *ethclient.Client) error {
		var err error
		logs, err = client.FilterLogs(ctx, q)

		return err
	})

	return logs, err
}

func (mc *MultiClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := mc.retryWithBackups("SubscribeFilterLogs", func(client *ethclient.Client) error {
		var err error
		sub, err = client.SubscribeFilterLogs(ctx, q, ch)

		return err
	})

	return sub, err
}

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

	for _, client := range append([]*ethclient.Client{mc.Client}, mc.Backups...) {
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

func (mc *MultiClient) retryWithBackups(opName string, op func(*ethclient.Client) error) error {
	var err error
	traceID := uuid.New()
	for i, client := range append([]*ethclient.Client{mc.Client}, mc.Backups...) {
		retryCount := 0
		err2 := retry.Do(func() error {
			err = op(client)
			if err != nil {
				mc.lggr.Warnf("traceID %q: chain %q: op: %q: client index %d: failed execution - retryable error '%s'", traceID.String(), mc.chainName, opName, i, MaybeDataErr(err))
				return err
			}

			return nil
		}, retry.Attempts(mc.RetryConfig.Attempts), retry.Delay(mc.RetryConfig.Delay),
			retry.OnRetry(func(n uint, err error) { retryCount++ }))
		if err2 == nil {
			if retryCount > 0 {
				mc.lggr.Infof("traceID %q: chain %q: op: %q: client index %d: successfully executed after %d retry", traceID.String(), mc.chainName, opName, i, retryCount)
			}

			return nil
		}
		mc.lggr.Infof("traceID %q: chain %q: op: %q: client index %d: failed, trying next client", traceID.String(), mc.chainName, opName, i)
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
		var err2 error
		mc.lggr.Debugf("traceID %q: chain %q: rpc: %q: dialing endpoint '%s'", traceID.String(), mc.chainName, rpc.Name, endpoint)
		client, err2 = ethclient.Dial(endpoint)
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
