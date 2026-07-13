package gas

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

// Client exposes the RPC methods required to apply gas price defaults.
type Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

// ApplyDefaults applies configured default gas limit and price settings to opts.
// When DefaultGasPriceWei is set, the latest block header is inspected to choose EIP-1559 vs legacy.
func ApplyDefaults(ctx context.Context, client Client, opts *bind.TransactOpts, cfg Config) error {
	if opts == nil {
		return errors.New("transact opts are nil")
	}

	if cfg.DefaultGasLimit > 0 {
		opts.GasLimit = cfg.DefaultGasLimit
	}

	if cfg.DefaultGasPriceWei == 0 {
		return nil
	}

	if client == nil {
		return errors.New("gas client is required when default_gas_price_wei is set")
	}

	return applyGasPrice(ctx, client, opts, weiToBigInt(cfg.DefaultGasPriceWei), weiToBigInt(cfg.DefaultGasTipCapWei))
}

func applyGasPrice(ctx context.Context, client Client, opts *bind.TransactOpts, gasPrice, tipOverride *big.Int) error {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest header: %w", err)
	}
	if header == nil {
		return errors.New("latest block header is nil")
	}

	if IsEIP1559Header(header) {
		if gasPrice == nil {
			return errors.New("gas fee cap is nil")
		}

		var tip *big.Int
		if tipOverride != nil {
			tip = tipOverride
		} else {
			tip, err = client.SuggestGasTipCap(ctx)
			if err != nil {
				return fmt.Errorf("failed to get suggested tip cap: %w", err)
			}
		}
		if tip == nil {
			return errors.New("gas tip cap is nil")
		}

		maxTip := new(big.Int).Sub(gasPrice, header.BaseFee)
		if maxTip.Sign() < 0 {
			return fmt.Errorf("default_gas_price_wei %s is below current base fee %s", gasPrice, header.BaseFee)
		}
		if tip.Cmp(maxTip) > 0 {
			tip = new(big.Int).Set(maxTip)
		}

		opts.GasPrice = nil
		opts.GasTipCap = tip
		opts.GasFeeCap = gasPrice
	} else {
		applyLegacyGasPrice(opts, gasPrice)
	}

	return nil
}

func applyLegacyGasPrice(opts *bind.TransactOpts, gasPrice *big.Int) {
	opts.GasPrice = gasPrice
	opts.GasFeeCap = nil
	opts.GasTipCap = nil
}

// IsEIP1559Header reports whether the header indicates EIP-1559 support.
func IsEIP1559Header(header *types.Header) bool {
	return header != nil && header.BaseFee != nil
}
