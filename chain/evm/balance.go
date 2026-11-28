package evm

import (
	"context"
	"fmt"
	"math/big"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type BalanceReader struct {
}

func (BalanceReader) Family() string {
	return chainsel.FamilyEVM
}

func (BalanceReader) Decimals(common.BlockChain) uint8 {
	return 18
}

func (b BalanceReader) Address(c common.BlockChain) (string, error) {
	evmChain, ok := c.(Chain)
	if !ok {
		return "", fmt.Errorf("invalid chain type: expected *Chain, got %T", c)
	}
	if evmChain.DeployerKey == nil {
		return "", fmt.Errorf("no deployer key configured for chain %s", evmChain.Name())
	}

	return evmChain.DeployerKey.From.Hex(), nil
}

func (BalanceReader) Balance(ctx context.Context, c common.BlockChain) (*big.Int, error) {
	evmChain, ok := c.(Chain)
	if !ok {
		return nil, fmt.Errorf("invalid chain type: expected *Chain, got %T", c)
	}

	return evmChain.Client.BalanceAt(ctx, evmChain.DeployerKey.From, nil)
}
