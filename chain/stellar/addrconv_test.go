package stellar

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAccountAddress = "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7"

func TestAddressToBytes(t *testing.T) {
	t.Parallel()

	t.Run("account address (G…)", func(t *testing.T) {
		t.Parallel()

		b, err := AddressToBytes(testAccountAddress)
		require.NoError(t, err)
		require.NotNil(t, b)
		assert.Len(t, b, 32, "account strkey decodes to 32 raw bytes")

		// Round-trips back to the same strkey.
		encoded, err := strkey.Encode(strkey.VersionByteAccountID, b)
		require.NoError(t, err)
		assert.Equal(t, testAccountAddress, encoded)
	})

	t.Run("contract address (C…)", func(t *testing.T) {
		t.Parallel()

		// Generate a contract strkey in-test to avoid hardcoding a real one.
		original, err := strkey.Encode(strkey.VersionByteContract, make([]byte, 32))
		require.NoError(t, err)

		b, err := AddressToBytes(original)
		require.NoError(t, err)
		require.NotNil(t, b)
		assert.Len(t, b, 32, "contract strkey decodes to 32 raw bytes")

		encoded, err := strkey.Encode(strkey.VersionByteContract, b)
		require.NoError(t, err)
		assert.Equal(t, original, encoded)
	})

	t.Run("invalid address", func(t *testing.T) {
		t.Parallel()

		b, err := AddressToBytes("not-a-stellar-address")
		require.Error(t, err)
		assert.Nil(t, b)
		assert.Contains(t, err.Error(), "invalid Stellar address format")
	})

	t.Run("wrong-length strkey rejected", func(t *testing.T) {
		t.Parallel()

		// strkey.Decode validates only the version byte + checksum, not the payload
		// length. Build a valid-checksum account strkey from a 16-byte payload (not 32)
		// and confirm AddressToBytes rejects it rather than returning a short slice.
		short, err := strkey.Encode(strkey.VersionByteAccountID, make([]byte, 16))
		require.NoError(t, err)

		b, err := AddressToBytes(short)
		require.Error(t, err)
		assert.Nil(t, b)
		assert.Contains(t, err.Error(), "invalid Stellar address format")
	})
}

func TestAddressConverter(t *testing.T) {
	t.Parallel()

	c := AddressConverter{}

	t.Run("supports stellar", func(t *testing.T) {
		t.Parallel()
		assert.True(t, c.Supports(chain_selectors.FamilyStellar))
	})

	t.Run("does not support other families", func(t *testing.T) {
		t.Parallel()
		assert.False(t, c.Supports(chain_selectors.FamilyEVM))
	})

	t.Run("converts account address via ConvertToBytes", func(t *testing.T) {
		t.Parallel()

		b, err := c.ConvertToBytes(testAccountAddress)
		require.NoError(t, err)
		assert.Len(t, b, 32)
	})
}
