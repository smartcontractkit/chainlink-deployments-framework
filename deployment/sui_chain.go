package deployment

import (
	"crypto/ed25519"

	"github.com/pattonkan/sui-go/suiclient"
)

// SuiChain represents an Sui chain.
type SuiChain struct {
	Selector uint64
	Client   *suiclient.ClientImpl
	// TODO: sui-go currently does not have a working Signer interface, so we
	// have the raw private key for now.
	DeployerKey ed25519.PrivateKey
	URL         string

	Confirm func(txHash string, opts ...any) error
}
