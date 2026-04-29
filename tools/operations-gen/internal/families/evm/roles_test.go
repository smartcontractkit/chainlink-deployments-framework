package evm_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

func TestResolveRoleField(t *testing.T) {
	t.Parallel()

	t.Run("default admin role", func(t *testing.T) {
		t.Parallel()
		got, err := evm.ResolveRoleField("DEFAULT_ADMIN_ROLE")
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("rejects raw hex without prefix", func(t *testing.T) {
		t.Parallel()
		input := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
		_, err := evm.ResolveRoleField(input)
		require.ErrorContains(t, err, "role must be a human-readable role name")
	})

	t.Run("rejects raw hex with prefix", func(t *testing.T) {
		t.Parallel()
		_, err := evm.ResolveRoleField("0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
		require.ErrorContains(t, err, "role must be a human-readable role name")
	})

	t.Run("known role name hash", func(t *testing.T) {
		t.Parallel()
		got, err := evm.ResolveRoleField("SOME_ROLE")
		require.NoError(t, err)
		const want = "daf0a44b794c6bf415956f72f4e104c140acfb95b835de7a73437dc7a7fd9aae"
		require.Equal(t, want, hex.EncodeToString(got[:]))
	})
}

func TestFormatRoleGoLiteral(t *testing.T) {
	t.Parallel()

	t.Run("zero role", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "[32]byte{}", evm.FormatRoleGoLiteral([32]byte{}))
	})

	t.Run("non-zero role", func(t *testing.T) {
		t.Parallel()
		role := [32]byte{0x01, 0x02, 0xff}
		want := "[32]byte{0x01, 0x02, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}"
		require.Equal(t, want, evm.FormatRoleGoLiteral(role))
	})
}
