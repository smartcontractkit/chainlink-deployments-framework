package stellar

import (
	"encoding/hex"
	"testing"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStellarKeypairSigner(t *testing.T) {
	t.Parallel()

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := NewStellarKeypairSigner(kp)
	require.NotNil(t, signer)

	assert.Equal(t, kp.Address(), signer.Address())
}

func TestStellarKeypairSigner_Sign(t *testing.T) {
	t.Parallel()

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := NewStellarKeypairSigner(kp)

	message := []byte("test message")
	sig, err := signer.Sign(message)
	require.NoError(t, err)
	assert.NotNil(t, sig)

	// Verify the signature using the keypair
	err = kp.Verify(message, sig)
	assert.NoError(t, err, "signature should be valid")
}

func TestStellarKeypairSigner_SignDecorated(t *testing.T) {
	t.Parallel()

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := NewStellarKeypairSigner(kp)

	message := []byte("test message")
	decoratedSig, err := signer.SignDecorated(message)
	require.NoError(t, err)

	// XDR DecoratedSignature should have hint and signature
	// Note: decoratedSig.Hint is xdr.SignatureHint which wraps [4]byte
	expectedHint := kp.Hint()
	assert.Equal(t, expectedHint[:], decoratedSig.Hint[:])
	assert.NotEmpty(t, decoratedSig.Signature)
}

func TestStellarKeypairSigner_Address(t *testing.T) {
	t.Parallel()

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := NewStellarKeypairSigner(kp)
	address := signer.Address()

	assert.NotEmpty(t, address)
	assert.Equal(t, kp.Address(), address)
}

func TestKeypairFromHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hexKey  string
		wantErr string
	}{
		{
			name:   "valid 32-byte hex key without prefix",
			hexKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:   "valid 32-byte hex key with 0x prefix",
			hexKey: "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:    "invalid hex string",
			hexKey:  "not_valid_hex",
			wantErr: "failed to decode hex key",
		},
		{
			name:    "wrong length - too short",
			hexKey:  "0123456789abcdef",
			wantErr: "invalid key length: expected 32 bytes, got 8",
		},
		{
			name:    "wrong length - too long",
			hexKey:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: "invalid key length: expected 32 bytes, got 40",
		},
		{
			name:    "empty string",
			hexKey:  "",
			wantErr: "invalid key length: expected 32 bytes, got 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kp, err := KeypairFromHex(tt.hexKey)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, kp)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, kp)

			// Verify the keypair can sign
			message := []byte("test")
			sig, err := kp.Sign(message)
			require.NoError(t, err)
			assert.NotNil(t, sig)

			// Verify the signature
			err = kp.Verify(message, sig)
			require.NoError(t, err)
		})
	}
}

func TestKeypairFromHex_ConsistentAddress(t *testing.T) {
	t.Parallel()

	// Use a known seed to test consistency
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Parse twice
	kp1, err := KeypairFromHex(hexKey)
	require.NoError(t, err)

	kp2, err := KeypairFromHex(hexKey)
	require.NoError(t, err)

	// Both should produce the same address
	assert.Equal(t, kp1.Address(), kp2.Address())

	// Verify the keypairs can sign and verify each other's signatures
	message := []byte("test message")

	sig1, err := kp1.Sign(message)
	require.NoError(t, err)

	sig2, err := kp2.Sign(message)
	require.NoError(t, err)

	// Signatures should be identical for the same message and keypair
	assert.Equal(t, sig1, sig2)

	// Each keypair should be able to verify the other's signature
	err = kp1.Verify(message, sig2)
	require.NoError(t, err)

	err = kp2.Verify(message, sig1)
	require.NoError(t, err)
}

func TestKeypairFromHex_RealStellarKey(t *testing.T) {
	t.Parallel()

	// We can't directly get the raw seed from SDK's Full keypair,
	// so we'll just verify that our KeypairFromHex works with a properly formatted hex
	testHex := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	reconstructedKp, err := KeypairFromHex(testHex)
	require.NoError(t, err)

	// Verify it can sign
	message := []byte("test")
	sig, err := reconstructedKp.Sign(message)
	require.NoError(t, err)

	err = reconstructedKp.Verify(message, sig)
	require.NoError(t, err)

	// Verify the address is a valid Stellar address format (starts with G)
	address := reconstructedKp.Address()
	assert.NotEmpty(t, address)
	assert.Equal(t, 'G', rune(address[0]), "Stellar public addresses start with 'G'")
}

func TestKeypairFromHex_WithAndWithoutPrefix(t *testing.T) {
	t.Parallel()

	hexWithout := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hexWith := "0x" + hexWithout

	kp1, err := KeypairFromHex(hexWithout)
	require.NoError(t, err)

	kp2, err := KeypairFromHex(hexWith)
	require.NoError(t, err)

	// Both should produce the same address
	assert.Equal(t, kp1.Address(), kp2.Address())
}

func TestStellarSigner_Interface(t *testing.T) {
	t.Parallel()

	// Verify stellarKeypairSigner implements StellarSigner
	var _ StellarSigner = (*stellarKeypairSigner)(nil)

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := NewStellarKeypairSigner(kp)
	require.NotNil(t, signer)

	// Test all interface methods
	address := signer.Address()
	assert.NotEmpty(t, address)

	message := []byte("test")
	sig, err := signer.Sign(message)
	require.NoError(t, err)
	assert.NotNil(t, sig)

	decoratedSig, err := signer.SignDecorated(message)
	require.NoError(t, err)
	assert.NotEmpty(t, decoratedSig.Signature)
}

func TestKeypairFromHex_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hexKey  string
		wantErr string
	}{
		{
			name:   "all zeros",
			hexKey: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:   "all ones",
			hexKey: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			name:   "uppercase hex",
			hexKey: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
		},
		{
			name:   "mixed case hex",
			hexKey: "AbCdEf0123456789aBcDeF0123456789AbCdEf0123456789aBcDeF0123456789",
		},
		{
			name:    "odd-length hex after 0x removal",
			hexKey:  "0x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
			wantErr: "invalid key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kp, err := KeypairFromHex(tt.hexKey)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, kp)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, kp)

			// Verify it can be used
			message := []byte("test")
			sig, err := kp.Sign(message)
			require.NoError(t, err)

			err = kp.Verify(message, sig)
			assert.NoError(t, err)
		})
	}
}

func TestKeypairFromHex_ByteConversion(t *testing.T) {
	t.Parallel()

	// Test with known bytes
	expectedBytes := make([]byte, 32)
	for i := range expectedBytes {
		expectedBytes[i] = byte(i)
	}

	hexKey := hex.EncodeToString(expectedBytes)
	require.Len(t, hexKey, 64, "hex encoding of 32 bytes should be 64 chars")

	kp, err := KeypairFromHex(hexKey)
	require.NoError(t, err)
	require.NotNil(t, kp)

	// Verify the keypair works
	message := []byte("test")
	sig, err := kp.Sign(message)
	require.NoError(t, err)

	err = kp.Verify(message, sig)
	assert.NoError(t, err)
}
