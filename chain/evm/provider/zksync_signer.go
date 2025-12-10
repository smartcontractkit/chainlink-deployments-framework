package provider

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/zksync-sdk/zksync2-go/types"
)

// zkSyncSigner implements the Signer interface for ZkSync signing using KMS. To prevent using
// the EVM interpreter, traditional deployment using EIP-712 typed data signing is required.
//
// This signer should only be used for contract deployment on ZkSync networks. It does not
// implement the SignMessage and SignTransaction methods, as it is not intended for signing
// arbitrary messages or transactions. Instead, it only implements SignTypedData, which is used for
// contract deployment in ZkSync.
//
// The SignTypedData implementation is based on the [ZKSync SDK] and is used internally by the SDK
// to sign typed data for contract deployment.
//
// [Signer interface]: https://github.com/zksync-sdk/zksync2-go/blob/2c742ee399c63cbf9fb72febfe9a1ad962992a54/accounts/signer.go#L18-L29
// [ZkSync SDK]: https://github.com/zksync-sdk/zksync2-go/blob/2c742ee399c63cbf9fb72febfe9a1ad962992a54/accounts/signer.go#L134-L164
type zkSyncSigner struct {
	// address is the zkSync address of the signer.
	address common.Address
	// chainID is the chain ID of the zkSync network.
	chainID *big.Int
	// signHash is a function that signs the hash of the typed data. This is used to sign the
	// prefixed data hash in the SignTypedData method.
	signHash func([]byte) ([]byte, error)
}

// newZkSyncSigner creates a new zkSyncKMSSigner instance.
func newZkSyncSigner(
	address common.Address, chainID *big.Int, signHashFunc func([]byte) ([]byte, error),
) *zkSyncSigner {
	return &zkSyncSigner{
		address:  address,
		chainID:  chainID,
		signHash: signHashFunc,
	}
}

// PrivateKey returns nil as this zkSyncSigner does not have access to the private key directly.
func (s *zkSyncSigner) PrivateKey() *ecdsa.PrivateKey {
	return nil
}

// GetAddress returns the zkSync address of the signer.
func (s *zkSyncSigner) Address() common.Address {
	return s.address
}

// ChainID returns the chain ID of the zkSync network this signer is associated with.
func (s *zkSyncSigner) ChainID() *big.Int {
	return s.chainID
}

// SignMessage is not implemented for zkSyncSigner as it is not intended for signing arbitrary
// messages.
func (s *zkSyncSigner) SignMessage(ctx context.Context, message []byte) ([]byte, error) {
	return nil, errors.New("SignMessage not implemented")
}

// SignTransaction is not implemented for zkSyncSigner as it is not intended for signing
// arbitrary transactions.
func (s *zkSyncSigner) SignTransaction(ctx context.Context, tx *types.Transaction) ([]byte, error) {
	return nil, errors.New("SignTransaction not implemented")
}

// SignTypedData signs the given typed data using the zkSync signing method. It hashes the domain
// and message according to the EIP-712 standard, prefixes the data, and then signs the resulting
// hash using the signHash function provided during initialization.
func (s *zkSyncSigner) SignTypedData(ctx context.Context, typedData *apitypes.TypedData) ([]byte, error) {
	domain, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to get hash of typed data domain: %w", err)
	}

	dataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash of typed message: %w", err)
	}

	prefixedData := fmt.Appendf(nil, "\x19\x01%s%s", string(domain), string(dataHash))
	prefixedDataHash := crypto.Keccak256(prefixedData)

	sig, err := s.signHash(prefixedDataHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash of typed data: %w", err)
	}

	// crypto.Sign uses the traditional implementation where v is either 0 or 1,
	// while Ethereum uses newer implementation where v is either 27 or 28.
	if sig[64] < 27 {
		sig[64] += 27
	}

	return sig, nil
}
