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

	return applyGasPrice(ctx, client, opts, weiToBigInt(cfg.DefaultGasPriceWei), weiToBigInt(cfg.DefaultGasTipCapWei))
}

// applyGasPrice checks the latest block header to determine whether the chain supports EIP-1559.
// When tipOverride is non-nil it is used as GasTipCap instead of SuggestGasTipCap.
func applyGasPrice(ctx context.Context, client Client, opts *bind.TransactOpts, gasPrice, tipOverride *big.Int) error {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest header: %w", err)
	}

	if header.BaseFee != nil {
		var tip *big.Int
		if tipOverride != nil {
			tip = tipOverride
		} else {
			tip, err = client.SuggestGasTipCap(ctx)
			if err != nil {
				return fmt.Errorf("failed to get suggested tip cap: %w", err)
			}
			if gasPrice != nil && gasPrice.Cmp(tip) < 0 {
				tip = new(big.Int).Set(gasPrice)
			}
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

// ApplyBoostOverrides returns a copy of base with gas limit and price overrides applied.
// When gasPriceWei is non-zero on an EIP-1559 chain, gasPriceWei is used as GasFeeCap and
// the existing GasTipCap on base is preserved when set.
func ApplyBoostOverrides(
	ctx context.Context,
	client Client,
	base *bind.TransactOpts,
	gasLimit, gasPriceWei uint64,
) (*bind.TransactOpts, error) {
	if base == nil {
		return nil, errors.New("transact opts are nil")
	}
	if gasLimit == 0 && gasPriceWei == 0 {
		return base, nil
	}

	opts := *base
	if gasLimit > 0 {
		opts.GasLimit = gasLimit
	}
	if gasPriceWei == 0 {
		return &opts, nil
	}

	gasPrice := weiToBigInt(gasPriceWei)
	if client == nil {
		applyLegacyGasPrice(&opts, gasPrice)
		return &opts, nil
	}

	if err := applyGasPrice(ctx, client, &opts, gasPrice, opts.GasTipCap); err != nil {
		return nil, err
	}

	return &opts, nil
}

// IsEIP1559Header reports whether the header indicates EIP-1559 support.
func IsEIP1559Header(header *types.Header) bool {
	return header != nil && header.BaseFee != nil
}
