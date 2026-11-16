package rpcclient

import (
	"context"
	"errors"
	"testing"

	sollib "github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"
)

// remoteSigner is a Signer that does not expose its private key, like a KMS signer.
type remoteSigner struct {
	key sollib.PrivateKey
	err error
}

func (s *remoteSigner) PublicKey() sollib.PublicKey { return s.key.PublicKey() }

func (s *remoteSigner) Sign(_ context.Context, message []byte) (sollib.Signature, error) {
	if s.err != nil {
		return sollib.Signature{}, s.err
	}

	return s.key.Sign(message)
}

// newTestTx builds a transaction paid by payer that also requires a signature from other.
func newTestTx(t *testing.T, payer, other sollib.PublicKey) *sollib.Transaction {
	t.Helper()

	tx, err := sollib.NewTransaction(
		[]sollib.Instruction{sollib.NewInstruction(
			sollib.SystemProgramID,
			sollib.AccountMetaSlice{
				sollib.Meta(payer).WRITE().SIGNER(),
				sollib.Meta(other).SIGNER(),
			},
			[]byte{},
		)},
		sollib.Hash{},
		sollib.TransactionPayer(payer),
	)
	require.NoError(t, err)

	return tx
}

func Test_signTx(t *testing.T) {
	t.Parallel()

	deployerKey := sollib.NewWallet().PrivateKey
	otherKey := sollib.NewWallet().PrivateKey

	tests := []struct {
		name     string
		deployer Signer
		signers  map[sollib.PublicKey]sollib.PrivateKey
		wantErr  string
	}{
		{
			name:     "remote deployer and additional signer",
			deployer: &remoteSigner{key: deployerKey},
			signers:  map[sollib.PublicKey]sollib.PrivateKey{otherKey.PublicKey(): otherKey},
		},
		{
			name:     "private key deployer and additional signer",
			deployer: NewPrivateKeySigner(deployerKey),
			signers:  map[sollib.PublicKey]sollib.PrivateKey{otherKey.PublicKey(): otherKey},
		},
		{
			name:     "missing additional signer",
			deployer: &remoteSigner{key: deployerKey},
			signers:  map[sollib.PublicKey]sollib.PrivateKey{},
			wantErr:  "not found",
		},
		{
			name:     "deployer fails to sign",
			deployer: &remoteSigner{key: deployerKey, err: errors.New("kms unavailable")},
			signers:  map[sollib.PublicKey]sollib.PrivateKey{otherKey.PublicKey(): otherKey},
			wantErr:  "kms unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx := newTestTx(t, deployerKey.PublicKey(), otherKey.PublicKey())

			err := signTx(t.Context(), tx, tt.deployer, tt.signers)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, tx.Signatures, 2)
			require.NoError(t, tx.VerifySignatures())
		})
	}
}
