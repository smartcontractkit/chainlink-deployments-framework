package changeset

import (
	"context"
	"fmt"
	"math/big"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

// RequireDeployerKeyBalance checks if the deployer key has a minimum balance
// on all chains. Useful to consider as a check in the beginning of a changeset
// to prevent running out of funds.
func RequireDeployerKeyBalance(ctx context.Context, allChains map[uint64]evm.Chain, minBalancesByChain map[uint64]*big.Int) error {
	for chainSel, minBalance := range minBalancesByChain {
		addr := allChains[chainSel].DeployerKey.From
		bal, err := allChains[chainSel].Client.BalanceAt(ctx, addr, nil)
		if err != nil {
			return err
		}
		if bal.Cmp(minBalance) < 0 {
			return fmt.Errorf("address %s has insufficient balance %v, required %v", addr.Hex(), bal, minBalance)
		}
	}

	return nil
}
