package provider

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	zkAccounts "github.com/zksync-sdk/zksync2-go/accounts"
)

// ZkSyncSignerGenerator is an interface for
type ZkSyncSignerGenerator interface {
	Generate(chainID *big.Int) (zkAccounts.Signer, error)
}

var (
	_ ZkSyncSignerGenerator = (*zkSyncSignerFromRaw)(nil)
	_ ZkSyncSignerGenerator = (*zkSyncSignerFromKMS)(nil)
)

// ZkSyncSignerFromRaw returns a generator which creates a ZkSync signer from a raw private key.
func ZkSyncSignerFromRaw(privKey string) *zkSyncSignerFromRaw {
	return &zkSyncSignerFromRaw{
		privKey: privKey,
	}
}

// zkSyncSignerFromRaw is a ZkSyncSignerGenerator that creates a ZkSync signer from a raw private
// key in hex format.
type zkSyncSignerFromRaw struct {
	privKey string
}

// Generate parses the raw private key and returns the ZkSync signer.
func (g *zkSyncSignerFromRaw) Generate(chainID *big.Int) (zkAccounts.Signer, error) {
	return zkAccounts.NewECDSASignerFromRawPrivateKey(common.FromHex(g.privKey), chainID)
}

// ZkSyncSignerRandom returns a generator which creates a ZkSync signer with a random private key.
func ZKSyncSignerRandom() ZkSyncSignerGenerator {
	return &zkSyncSignerRandom{}
}

// ZkSyncSignerRandom is a ZkSyncSignerGenerator that creates a ZkSync signer with a random private
// key.
type zkSyncSignerRandom struct{}

// Generate generates a random key and returns the ZkSync signer.
func (g *zkSyncSignerRandom) Generate(chainID *big.Int) (zkAccounts.Signer, error) {
	return zkAccounts.NewRandomBaseSigner(chainID)
}

// ZkSyncSignerFromKMS returns a generator which creates a ZkSync signer using AWS KMS.
//
// It requires the KMS key ID, region, and optionally an AWS profile name. If the AWS profile
// name is not provided, it defaults to using the environment variables to determine the AWS
// profile.
func ZkSyncSignerFromKMS(keyID, keyRegion, awsProfileName string) (*zkSyncSignerFromKMS, error) {
	signer, err := NewKMSSigner(keyID, keyRegion, awsProfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS signer: %w", err)
	}

	return &zkSyncSignerFromKMS{
		signer: signer,
	}, nil
}

// zkSyncSignerFromKMS is a ZkSyncSignerGenerator that creates a ZkSync signer using KMS.
type zkSyncSignerFromKMS struct {
	signer *KMSSigner
}

// Generate uses KMS to create a zksync accounts.Signer instance for signing transactions.
func (g *zkSyncSignerFromKMS) Generate(chainID *big.Int) (zkAccounts.Signer, error) {
	addr, err := g.signer.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address from KMS signer: %w", err)
	}

	return newZkSyncSigner(
		addr,
		chainID,
		g.signer.SignHash,
	), nil
}
