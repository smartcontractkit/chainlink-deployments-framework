package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// ConfirmFunctor is an interface for creating a confirmation function for transactions on the
// EVM chain.
type ConfirmFunctor interface {
	// Generate returns a function that confirms transactions on the EVM chain.
	Generate(
		ctx context.Context, selector uint64, client evm.OnchainClient, from common.Address,
	) (evm.ConfirmFunc, error)
}

// ConfirmFuncGeth returns a ConfirmFunctor that uses the Geth client to confirm transactions.
func ConfirmFuncGeth(waitMinedTimeout time.Duration, opts ...func(*confirmFuncGeth)) ConfirmFunctor {
	cf := &confirmFuncGeth{
		tickInterval:     1 * time.Second, // the same value we have in bind.WaitMined hardcoded in "go-ethereum"
		waitMinedTimeout: waitMinedTimeout,
	}
	for _, o := range opts {
		o(cf)
	}
	return cf
}

func WithTickInterval(interval time.Duration) func(*confirmFuncGeth) {
	return func(o *confirmFuncGeth) {
		o.tickInterval = interval
	}
}

// confirmFuncGeth implements the ConfirmFunctor interface which generates a confirmation function
// for transactions using the Geth client.
type confirmFuncGeth struct {
	tickInterval     time.Duration
	waitMinedTimeout time.Duration
}

// Generate returns a function that confirms transactions using the Geth client.
func (g *confirmFuncGeth) Generate(
	ctx context.Context, selector uint64, client evm.OnchainClient, from common.Address,
) (evm.ConfirmFunc, error) {
	return func(tx *types.Transaction) (uint64, error) {
		var blockNum uint64
		if tx == nil {
			return 0, fmt.Errorf("tx was nil, nothing to confirm for selector: %d", selector)
		}

		ctxTimeout, cancel := context.WithTimeout(ctx, g.waitMinedTimeout)
		defer cancel()

		receipt, err := WaitMinedWithInterval(ctxTimeout, g.tickInterval, client, tx.Hash())
		if err != nil {
			return 0, fmt.Errorf("tx %s failed to confirm for selector %d: %w",
				tx.Hash().Hex(), selector, err,
			)
		}
		if receipt == nil {
			return blockNum, fmt.Errorf("receipt was nil for tx %s for selector %d",
				tx.Hash().Hex(), selector,
			)
		}

		blockNum = receipt.BlockNumber.Uint64()

		if receipt.Status == 0 {
			reason, err := getErrorReasonFromTx(ctxTimeout, client, from, tx, receipt)
			if err == nil && reason != "" {
				return 0, fmt.Errorf("tx %s reverted for selector %d: %s",
					tx.Hash().Hex(), selector, reason,
				)
			}

			return blockNum, fmt.Errorf("tx %s reverted, could not decode error reason for selector %d",
				tx.Hash().Hex(), selector,
			)
		}

		return blockNum, nil
	}, nil
}

// ConfirmFuncSeth returns a ConfirmFunctor that uses the Seth client to confirm transactions.
// It requires the RPC URL, a list of directories where Geth wrappers are located, and an optional
// configuration file path for the Seth client. If you do not have a configuration file, you can
// pass an empty string.
func ConfirmFuncSeth(
	rpcURL string, waitMinedTimeout time.Duration, gethWrapperDirs []string, configFilePath string,
) ConfirmFunctor {
	return &confirmFuncSeth{
		rpcURL:           rpcURL,
		waitMinedTimeout: waitMinedTimeout,
		gethWrappersDirs: gethWrapperDirs,
		configFilePath:   configFilePath,
	}
}

// confirmFuncSeth implements the ConfirmFunctor interface which generates a confirmation function
// for transactions using the Seth client.
type confirmFuncSeth struct {
	rpcURL           string
	waitMinedTimeout time.Duration
	gethWrappersDirs []string
	configFilePath   string
}

// Generate returns a function that confirms transactions using the Seth client. The provided
// client must be a MultiClient.
func (g *confirmFuncSeth) Generate(
	ctx context.Context, selector uint64, client evm.OnchainClient, from common.Address,
) (evm.ConfirmFunc, error) {
	// Convert the client to a MultiClient because we need to use the multi-client's WaitMined
	// method.
	multiClient, ok := client.(*rpcclient.MultiClient)
	if !ok {
		return nil, fmt.Errorf("expected client to be of type *rpcclient.MultiClient, got %T", client)
	}

	// Get the ChainID from the selector
	chainIDStr, err := chainsel.GetChainIDFromSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", selector, err)
	}

	chainID, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chain ID %s: %w", chainIDStr, err)
	}

	// Setup the seth client
	sethClient, err := newSethClient(
		g.rpcURL, chainID, g.gethWrappersDirs, g.configFilePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup seth client: %w", err)
	}

	return func(tx *types.Transaction) (uint64, error) {
		ctxTimeout, cancel := context.WithTimeout(ctx, g.waitMinedTimeout)
		defer cancel()

		if _, err := multiClient.WaitMined(ctxTimeout, tx); err != nil {
			return 0, err
		}

		decoded, err := sethClient.DecodeTx(tx)
		if err != nil {
			return 0, err
		}

		if decoded.Receipt == nil {
			return 0, fmt.Errorf("no receipt found for transaction %s even though it wasn't reverted. This should not happen", tx.Hash().String())
		}

		return decoded.Receipt.BlockNumber.Uint64(), nil
	}, nil
}

// WaitMinedWithInterval is a custom function that allows to get receipts faster for networks with instant blocks
func WaitMinedWithInterval(ctx context.Context, tick time.Duration, b bind.DeployBackend, txHash common.Hash) (*types.Receipt, error) {
	queryTicker := time.NewTicker(tick)
	defer queryTicker.Stop()
	for {
		receipt, err := b.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}
