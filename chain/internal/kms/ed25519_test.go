package kms

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"

	kmsv2 "github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

const testKeyID = "test-ed25519-key-id"

// newTestKeypairDER returns a fresh Ed25519 keypair with the public key encoded
// as PKIX DER.
func newTestKeypairDER(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, []byte) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return pub, priv, der
}

func Test_Ed25519KMSSigner_GetPublicKey(t *testing.T) {
	t.Parallel()

	pub, _, der := newTestKeypairDER(t)

	t.Run("happy path parses and caches the key", func(t *testing.T) {
		t.Parallel()

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: der,
		}, nil).Once()

		signer, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.NoError(t, err)

		got, err := signer.GetPublicKey(t.Context())
		require.NoError(t, err)
		require.Equal(t, pub, got)
	})

	t.Run("returns a copy that cannot mutate the cached key", func(t *testing.T) {
		t.Parallel()

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: der,
		}, nil).Once()

		signer, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.NoError(t, err)

		got, err := signer.GetPublicKey(t.Context())
		require.NoError(t, err)
		got[0] ^= 0xFF // mutating the returned slice must not affect the cache

		fresh, err := signer.GetPublicKey(t.Context())
		require.NoError(t, err)
		require.Equal(t, pub, fresh)
	})

	t.Run("rejects wrong key spec", func(t *testing.T) {
		t.Parallel()

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccSecgP256k1,
			PublicKey: der,
		}, nil)

		_, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.Error(t, err)
		require.ErrorContains(t, err, string(kmstypes.KeySpecEccNistEdwards25519))
	})

	t.Run("rejects malformed DER", func(t *testing.T) {
		t.Parallel()

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: []byte("not-valid-der"),
		}, nil)

		_, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.Error(t, err)
		require.ErrorContains(t, err, "cannot parse public key")
	})

	t.Run("rejects non-ed25519 key that still parses as PKIX", func(t *testing.T) {
		t.Parallel()

		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		ecDER, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
		require.NoError(t, err)

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: ecDER,
		}, nil)

		_, err = NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.Error(t, err)
		require.ErrorContains(t, err, "want ed25519.PublicKey")
	})

	t.Run("rejects empty key ID", func(t *testing.T) {
		t.Parallel()

		client := kmsmocks.NewMockEd25519Client(t)

		_, err := NewEd25519KMSSigner(t.Context(), "", client)
		require.Error(t, err)
		require.ErrorContains(t, err, "KMS key ID is required")
	})

	t.Run("rejects nil client", func(t *testing.T) {
		t.Parallel()

		_, err := NewEd25519KMSSigner(t.Context(), testKeyID, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "KMS client is required")
	})
}

func Test_Ed25519KMSSigner_Sign(t *testing.T) {
	t.Parallel()

	t.Run("signs with RAW + ED25519_SHA_512", func(t *testing.T) {
		t.Parallel()

		pub, priv, der := newTestKeypairDER(t)
		message := []byte("message to sign")

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: der,
		}, nil)
		client.EXPECT().Sign(mock.Anything, mock.Anything).RunAndReturn(
			func(_ context.Context, in *kmsv2.SignInput, _ ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error) {
				require.Equal(t, kmstypes.MessageTypeRaw, in.MessageType)
				require.Equal(t, kmstypes.SigningAlgorithmSpecEd25519Sha512, in.SigningAlgorithm)
				require.Equal(t, message, in.Message)

				return &kmsv2.SignOutput{Signature: ed25519.Sign(priv, in.Message)}, nil
			})

		signer, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.NoError(t, err)

		sig, err := signer.Sign(t.Context(), message)
		require.NoError(t, err)
		require.Len(t, sig, ed25519.SignatureSize)
		require.True(t, ed25519.Verify(pub, message, sig))
	})

	t.Run("rejects signature of unexpected length", func(t *testing.T) {
		t.Parallel()

		_, _, der := newTestKeypairDER(t)

		client := kmsmocks.NewMockEd25519Client(t)
		client.EXPECT().GetPublicKey(mock.Anything, mock.Anything).Return(&kmsv2.GetPublicKeyOutput{
			KeySpec:   kmstypes.KeySpecEccNistEdwards25519,
			PublicKey: der,
		}, nil)
		client.EXPECT().Sign(mock.Anything, mock.Anything).Return(&kmsv2.SignOutput{
			Signature: []byte("too-short"),
		}, nil)

		signer, err := NewEd25519KMSSigner(t.Context(), testKeyID, client)
		require.NoError(t, err)

		_, err = signer.Sign(t.Context(), []byte("msg"))
		require.Error(t, err)
		require.ErrorContains(t, err, "unexpected signature length")
	})
}
