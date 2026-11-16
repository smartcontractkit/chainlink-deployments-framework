package solana

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"testing"

	kmsv2 "github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	sollib "github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

const kmsTestKeyID = "test-ed25519-key-id"

// newKMSTestClient returns a mock KMS client backed by a fresh Ed25519 key, along with the key's
// Solana address.
func newKMSTestClient(t *testing.T) (*kmsmocks.MockEd25519Client, sollib.PublicKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()
	client.EXPECT().Sign(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, in *kmsv2.SignInput, _ ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error) {
			return &kmsv2.SignOutput{Signature: ed25519.Sign(priv, in.Message)}, nil
		}).Maybe()

	return client, sollib.PublicKeyFromBytes(pub)
}

func TestKMSSigner_Sign(t *testing.T) {
	t.Parallel()

	client, wantPubKey := newKMSTestClient(t)

	signer, err := newKMSSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)
	require.Equal(t, wantPubKey, signer.PublicKey())

	message := []byte("message to sign")
	sig, err := signer.Sign(t.Context(), message)
	require.NoError(t, err)
	require.True(t, sig.Verify(wantPubKey, message))
}

func TestKMSSigner_SignsTransaction(t *testing.T) {
	t.Parallel()

	client, pubKey := newKMSTestClient(t)

	signer, err := newKMSSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)

	tx, err := sollib.NewTransaction(
		[]sollib.Instruction{sollib.NewInstruction(
			sollib.SystemProgramID,
			sollib.AccountMetaSlice{sollib.Meta(pubKey).WRITE().SIGNER()},
			[]byte{},
		)},
		sollib.Hash{},
		sollib.TransactionPayer(pubKey),
	)
	require.NoError(t, err)

	message, err := tx.Message.MarshalBinary()
	require.NoError(t, err)
	sig, err := signer.Sign(t.Context(), message)
	require.NoError(t, err)
	tx.Signatures = []sollib.Signature{sig}

	require.NoError(t, tx.VerifySignatures())
}

func TestKMSSigner_WrongKeySpec(t *testing.T) {
	t.Parallel()

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec: kmstypes.KeySpecEccSecgP256k1,
	}, nil).Once()

	_, err := newKMSSigner(t.Context(), kmsTestKeyID, client)
	require.ErrorContains(t, err, "key spec")
}

func TestNewKMSSigner_MissingKeyID(t *testing.T) {
	t.Parallel()

	_, err := NewKMSSigner(t.Context(), "", "us-west-2")
	require.ErrorContains(t, err, "KMS key ID is required")
}
