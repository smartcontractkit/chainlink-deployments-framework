package provider

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

// SignerGenerator generates a Signer for the deployer account.
type SignerGenerator interface {
	Generate(ctx context.Context) (solana.Signer, error)
}

var _ SignerGenerator = (*signerGenKMS)(nil)

// SignerGenKMS returns a SignerGenerator backed by an AWS KMS Ed25519 key.
func SignerGenKMS(keyID, keyRegion string) SignerGenerator {
	return &signerGenKMS{keyID: keyID, keyRegion: keyRegion}
}

// signerGenKMS is a SignerGenerator that signs with an AWS KMS Ed25519 key.
type signerGenKMS struct {
	keyID     string
	keyRegion string
}

// Generate builds a Signer backed by the AWS KMS Ed25519 key.
func (g *signerGenKMS) Generate(ctx context.Context) (solana.Signer, error) {
	return solana.NewKMSSigner(ctx, g.keyID, g.keyRegion)
}
