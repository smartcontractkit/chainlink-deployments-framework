package stellar

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"testing"

	kmsv2 "github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

const kmsTestKeyID = "test-ed25519-key-id"

// newKmsTestKeypairDER returns a fresh Ed25519 keypair with the public key
// encoded as PKIX DER, matching the format AWS KMS returns.
func newKmsTestKeypairDER(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, []byte) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return pub, priv, der
}

func TestKmsSigner_Sign(t *testing.T) {
	t.Parallel()

	pub, priv, der := newKmsTestKeypairDER(t)
	message := []byte("message to sign")

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()
	client.EXPECT().Sign(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, in *kmsv2.SignInput, _ ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error) {
			return &kmsv2.SignOutput{Signature: ed25519.Sign(priv, in.Message)}, nil
		}).Once()

	signer, err := newKmsSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)

	sig, err := signer.Sign(message)
	require.NoError(t, err)
	require.Len(t, sig, ed25519.SignatureSize)
	require.True(t, ed25519.Verify(pub, message, sig))
}

func TestKmsSigner_SignAfterConstructionContextDone(t *testing.T) {
	t.Parallel()

	pub, priv, der := newKmsTestKeypairDER(t)
	message := []byte("message to sign")

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()
	client.EXPECT().Sign(mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, in *kmsv2.SignInput, _ ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error) {
			// The signer must not hand KMS the construction context.
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			return &kmsv2.SignOutput{Signature: ed25519.Sign(priv, in.Message)}, nil
		}).Once()

	ctx, cancel := context.WithCancel(t.Context())
	signer, err := newKmsSigner(ctx, kmsTestKeyID, client)
	require.NoError(t, err)

	// A chain outlives the context that loaded it, so signing must still work
	// once that context is done.
	cancel()

	sig, err := signer.Sign(message)
	require.NoError(t, err)
	require.True(t, ed25519.Verify(pub, message, sig))
}

func TestKmsSigner_SignDecorated(t *testing.T) {
	t.Parallel()

	pub, priv, der := newKmsTestKeypairDER(t)
	message := []byte("message to sign")

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()
	client.EXPECT().Sign(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, in *kmsv2.SignInput, _ ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error) {
			return &kmsv2.SignOutput{Signature: ed25519.Sign(priv, in.Message)}, nil
		}).Once()

	signer, err := newKmsSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)

	decorated, err := signer.SignDecorated(message)
	require.NoError(t, err)

	// Hint is the last four bytes of the public key.
	require.Equal(t, []byte(pub[len(pub)-4:]), decorated.Hint[:])
	require.Len(t, decorated.Signature, ed25519.SignatureSize)
	require.True(t, ed25519.Verify(pub, message, []byte(decorated.Signature)))
}

func TestKmsSigner_Address(t *testing.T) {
	t.Parallel()

	pub, _, der := newKmsTestKeypairDER(t)

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()

	signer, err := newKmsSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)

	expected, err := strkey.Encode(strkey.VersionByteAccountID, pub)
	require.NoError(t, err)

	require.Equal(t, expected, signer.Address())
}

func TestKmsSigner_KeypairFull(t *testing.T) {
	t.Parallel()

	_, _, der := newKmsTestKeypairDER(t)

	client := kmsmocks.NewMockEd25519Client(t)
	client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
		KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
		PublicKey: der,
	}, nil).Once()

	signer, err := newKmsSigner(t.Context(), kmsTestKeyID, client)
	require.NoError(t, err)
	require.Nil(t, signer.KeypairFull())
}
