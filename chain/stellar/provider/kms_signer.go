package provider

import (
	"context"

	bindings "github.com/smartcontractkit/chainlink-stellar/bindings"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

// keypairGenKMS is a KeypairGenerator that signs with an AWS KMS Ed25519 key.
type keypairGenKMS struct {
	// ctx is cached because KeypairGenerator.Generate has no context parameter.
	// Generate runs synchronously while the chain is loaded, so the loader's
	// context is the correct scope for the AWS calls it makes. The signer it
	// returns does not retain this context; it derives a fresh one per signature.
	//
	//nolint:containedctx
	ctx       context.Context
	keyID     string
	keyRegion string
}

var _ KeypairGenerator = (*keypairGenKMS)(nil)

// KeypairGenKMS creates a KeypairGenerator backed by an AWS KMS Ed25519 key.
func KeypairGenKMS(ctx context.Context, keyID, keyRegion string) KeypairGenerator {
	return &keypairGenKMS{
		ctx:       ctx,
		keyID:     keyID,
		keyRegion: keyRegion,
	}
}

// Generate builds a Stellar signer backed by an AWS KMS Ed25519 key.
func (k *keypairGenKMS) Generate() (bindings.Signer, error) {
	return stellar.NewKmsSigner(k.ctx, k.keyID, k.keyRegion)
}
