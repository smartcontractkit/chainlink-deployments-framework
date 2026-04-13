package proposalutils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// TransactionForChain builds an mcmstypes.Transaction for the given chain selector.
// It currently supports EVM and Solana chains; other chain families return an error.
func TransactionForChain(
	chain uint64, toAddress string, data []byte, value *big.Int, contractType string, tags []string,
) (mcmstypes.Transaction, error) {
	chainFamily, err := mcmstypes.GetChainSelectorFamily(mcmstypes.ChainSelector(chain))
	if err != nil {
		return mcmstypes.Transaction{}, fmt.Errorf("failed to get chain family for chain %d: %w", chain, err)
	}

	var tx mcmstypes.Transaction

	switch chainFamily {
	case chain_selectors.FamilyEVM:
		if !common.IsHexAddress(toAddress) {
			return mcmstypes.Transaction{}, fmt.Errorf("invalid EVM address: %s", toAddress)
		}
		tx = mcmsevmsdk.NewTransaction(common.HexToAddress(toAddress), data, value, contractType, tags)

	case chain_selectors.FamilySolana:
		accounts := []*solana.AccountMeta{} // FIXME: how to pass accounts to support solana?
		var err error
		tx, err = mcmssolanasdk.NewTransaction(toAddress, data, value, accounts, contractType, tags)
		if err != nil {
			return mcmstypes.Transaction{}, fmt.Errorf("failed to create solana transaction: %w", err)
		}

	default:
		return mcmstypes.Transaction{}, fmt.Errorf("unsupported chain family %s", chainFamily)
	}

	return tx, nil
}

// BatchOperationForChain creates an mcmstypes.BatchOperation containing a single transaction
// for the given chain selector. It delegates to TransactionForChain, so it supports EVM and
// Solana chains.
func BatchOperationForChain(
	chain uint64, toAddress string, data []byte, value *big.Int, contractType string, tags []string,
) (mcmstypes.BatchOperation, error) {
	tx, err := TransactionForChain(chain, toAddress, data, value, contractType, tags)
	if err != nil {
		return mcmstypes.BatchOperation{}, fmt.Errorf("failed to create transaction for chain: %w", err)
	}

	return mcmstypes.BatchOperation{
		ChainSelector: mcmstypes.ChainSelector(chain),
		Transactions:  []mcmstypes.Transaction{tx},
	}, nil
}
