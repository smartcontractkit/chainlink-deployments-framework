package ton

import (
	"context"
	"crypto/ed25519"
	"fmt"

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
	ChainMetadata                            // Contains canonical chain identifier
	Client              ton.APIClientWrapped // APIClient for Lite Server connection
	Wallet              *wallet.Wallet       // Wallet abstraction (signing, sending)
	WalletAddress       *address.Address     // Address of deployer wallet
	WalletVersionConfig wallet.VersionConfig // Wallet version configuration
	URL                 string               // Liteserver URL
	HTTPURL             string               // HTTP URL
	Confirm             ConfirmFunc          // Function to confirm transactions
}

// MakeDefaultConfirmFunc creates a default ConfirmFunc that waits for transaction trace.
var MakeDefaultConfirmFunc = func(c ton.APIClientWrapped) ConfirmFunc {
	return func(ctx context.Context, tx *tlb.Transaction) error {
		return tracetracking.WaitForTrace(ctx, c, tx, bindings.DefaultTraceStopCondition)
	}
}

func (c Chain) ReadOnly() (common.BlockChain, error) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key for read-only chain %v: %w", c, err)
	}

	c.Wallet, err = wallet.FromPrivateKeyWithOptions(c.Client, privateKey, c.WalletVersionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate wallet for read-only chain %v: %w", c, err)
	}

	return c, nil
}
