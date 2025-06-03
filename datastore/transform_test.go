package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// CustomMetadata is a placeholder type for testing purposes.
type CustomMetadata struct {
	ChainSelector uint64 `json:"chain_selector"`
	Field         string `json:"field"`
}

// Clone creates a deep copy of CustomMetadata.
func (cm CustomMetadata) Clone() CustomMetadata {
	return CustomMetadata{
		Field:         cm.Field,
		ChainSelector: cm.ChainSelector,
	}
}

func TestAs(t *testing.T) {
	t.Parallel()

	// create a CustomMetadata instance
	orig := CustomMetadata{
		Field:         "test",
		ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
	}

	// put it in an `any` type and use As to convert it back
	var a any = orig
	typed, err := As[CustomMetadata](a)
	require.NoError(t, err)
	require.Equal(t, orig, typed)
}
