package ton

import (
	"context"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
)

// ConfirmFunc is a function that waits for the transaction to be confirmed, or returns an error.
type ConfirmFunc func(ctx context.Context, tx *tlb.Transaction) error

type ChainMetadata = common.ChainMetadata

// Chain represents a TON chain.
type Chain struct {
	ChainMetadata                  // Contains canonical chain identifier
	Client        *ton.APIClient   // APIClient for Lite Server connection
	Wallet        *wallet.Wallet   // Wallet abstraction (signing, sending)
	WalletAddress *address.Address // Address of deployer wallet
	URL           string           // Liteserver URL
	Confirm       ConfirmFunc      // Function to confirm transactions
}

// MakeDefaultConfirmFunc creates a default ConfirmFunc that waits for transaction trace.
var MakeDefaultConfirmFunc = func(c ton.APIClientWrapped) ConfirmFunc {
	return func(ctx context.Context, tx *tlb.Transaction) error {
		return tracetracking.WaitForTrace(ctx, c, tx, bindings.DefaultTraceStopCondition)
	}
}
