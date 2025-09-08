package provider

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
)

// AccountGenerator is an interface for generating Sui accounts.
type AccountGenerator interface {
	Generate() (sui.SuiSigner, error)
}

var (
	_ AccountGenerator = (*accountGenPrivateKey)(nil)
)

// accountGenPrivateKey is an account generator that creates an account from the private key.
type accountGenPrivateKey struct {
	// privateKey is the hex formatted private key used to generate the Aptos account.
	privateKey string
}

// AccountGenPrivateKey creates a new instance of accountGenPrivateKey with the provided private key.
func AccountGenPrivateKey(privateKey string) *accountGenPrivateKey {
	return &accountGenPrivateKey{
		privateKey: privateKey,
	}
}

// Generate generates an Sui account from the provided private key. It returns an error if the
// private key string cannot be parsed.
func (g *accountGenPrivateKey) Generate() (sui.SuiSigner, error) {
	return sui.NewSignerFromHexPrivateKey(g.privateKey)
}
