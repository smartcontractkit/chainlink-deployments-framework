package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

func TestAs(t *testing.T) {
	t.Parallel()

	// create a CustomMetadata instance
	orig := testMetadata{
		Field:         "test",
		ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
	}

	// put it in an `any` type and use As to convert it back
	var a any = orig
	typed, err := As[testMetadata](a)
	require.NoError(t, err)
	require.Equal(t, orig, typed)
}
