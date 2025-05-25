package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

var _ chain.Provider = (*RPCChainProvider)(nil)

type RPCChainProvider struct {
	selector uint64
	rpcs     []cldf.RPC
	auth     *bind.TransactOpts

	// Optional custom logger
	logger logger.Logger

	chain *evm.Chain
}

func NewRPCChainProvider(selector uint64) (*RPCChainProvider, error) {
	p := &RPCChainProvider{
		selector: selector,
	}

	return p, nil
}

func (p *RPCChainProvider) WithLogger(lggr logger.Logger) *RPCChainProvider {
	p.logger = lggr
	return p
}

func (p *RPCChainProvider) WithRPCs(rpcs []cldf.RPC) *RPCChainProvider {
	p.rpcs = rpcs
	return p
}

func (p *RPCChainProvider) WithAuth(auth *bind.TransactOpts) *RPCChainProvider {
	p.auth = auth
	return p
}

func (p *RPCChainProvider) Initialize() error {
	// Init the client
	client, err := cldf.NewMultiClient(p.logger, cldf.RPCConfig{
		ChainSelector: p.selector,
		RPCs:          p.rpcs,
	})
	if err != nil {
		return fmt.Errorf("failed to create multi client: %w", err)
	}

	chainMD := common.NewChainMetadata(p.selector)

	p.chain = &evm.Chain{
		Selector:    p.selector,
		Client:      client,
		DeployerKey: p.auth,
		Confirm: func(tx *types.Transaction) (uint64, error) {
			if tx == nil {
				return 0, fmt.Errorf("tx was nil, nothing to confirm chain %s", chainMD.Name)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			receipt, err := bind.WaitMined(ctx, client, tx)
			if err != nil {
				return 0, fmt.Errorf(
					"failed to get confirmed receipt for chain %s: %w", chainMD.Name, err,
				)
			}
			if receipt == nil {
				return 0, fmt.Errorf("receipt was nil for tx %s chain %s", tx.Hash().Hex(), chainMD.Name)
			}

			// if receipt.Status == 0 {
			// 	errReason, err := cldf.GetErrorReasonFromTx(ec, chainCfg.DeployerKey.From, tx, receipt)
			// 	if err == nil && errReason != "" {
			// 		return blockNumber, fmt.Errorf("tx %s reverted,error reason: %s chain %s", tx.Hash().Hex(), errReason, chainInfo.ChainName)
			// 	}
			// 	return blockNumber, fmt.Errorf("tx %s reverted, could not decode error reason chain %s", tx.Hash().Hex(), chainInfo.ChainName)
			// }

			return receipt.BlockNumber.Uint64(), nil
		},
	}

	// Validate the chain

	return nil
}

func (*RPCChainProvider) Name() string {
	return "EVM RPC Chain"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

func (p *RPCChainProvider) Chain() *evm.Chain {
	return p.chain
}
