package aptos

import (
	"github.com/aptos-labs/aptos-go-sdk"

	chain_common "github.com/smartcontractkit/chainlink-deployments-framework/chain/common"
)

// Chain represents an Aptos chain.
type Chain struct {
	chain_common.ChainInfoProvider

	Client         aptos.AptosRpcClient
	DeployerSigner aptos.TransactionSigner
	URL            string

	Confirm func(txHash string, opts ...any) error
}
