package keystore

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateKeyPair(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	return key
}

func Test_NewKeystore(t *testing.T) {
	t.Parallel()

	privKey := generateKeyPair(t)
	ks, addr := NewKeystore(privKey)

	expectedAddr := address.PubkeyToAddress(privKey.PublicKey)
	require.Contains(t, ks.Keys, expectedAddr.String())
	assert.Equal(t, privKey, ks.Keys[expectedAddr.String()])
	assert.Equal(t, expectedAddr, addr)
}

func Test_Keystore_ImportECDSA(t *testing.T) {
	t.Parallel()

	ks := &Keystore{Keys: make(map[string]*ecdsa.PrivateKey)}
	privKey := generateKeyPair(t)

	addr := ks.ImportECDSA(privKey)
	assert.Equal(t, address.PubkeyToAddress(privKey.PublicKey), addr)
	assert.Equal(t, privKey, ks.Keys[addr.String()])
}

func Test_Keystore_Sign(t *testing.T) {
	t.Parallel()

	privKey := generateKeyPair(t)
	addrStr := address.PubkeyToAddress(privKey.PublicKey).String()

	ks := &Keystore{
		Keys: map[string]*ecdsa.PrivateKey{
			addrStr: privKey,
		},
	}

	t.Run("successful sign", func(t *testing.T) {
		t.Parallel()

		hash := crypto.Keccak256([]byte("test data"))

		sig, err := ks.Sign(context.Background(), addrStr, hash)
		require.NoError(t, err)
		require.Len(t, sig, 65) // ECDSA signature length
	})

	t.Run("key not found", func(t *testing.T) {
		t.Parallel()

		_, err := ks.Sign(context.Background(), "invalid_address", []byte("hash"))
		require.Error(t, err)
		assert.EqualError(t, err, "no such key")
	})

	t.Run("nil hash returns nil without error", func(t *testing.T) {
		t.Parallel()

		sig, err := ks.Sign(context.Background(), addrStr, nil)
		require.NoError(t, err)
		assert.Nil(t, sig)
	})
}

func Test_Keystore_Accounts(t *testing.T) {
	t.Parallel()

	priv1 := generateKeyPair(t)
	priv2 := generateKeyPair(t)

	addr1 := address.PubkeyToAddress(priv1.PublicKey).String()
	addr2 := address.PubkeyToAddress(priv2.PublicKey).String()

	ks := &Keystore{
		Keys: map[string]*ecdsa.PrivateKey{
			addr1: priv1,
			addr2: priv2,
		},
	}

	accounts, err := ks.Accounts(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{addr1, addr2}, accounts)
}

func Test_Keystore_Decrypt(t *testing.T) {
	t.Parallel()

	ks := &Keystore{Keys: make(map[string]*ecdsa.PrivateKey)}
	_, err := ks.Decrypt(t.Context(), "account1", []byte("ciphertext"))
	require.ErrorContains(t, err, "decrypt not implemented")
}
