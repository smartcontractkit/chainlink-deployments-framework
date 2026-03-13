package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

func TestAs(t *testing.T) {
	t.Parallel()

	// create a CustomMetadata instance
	orig := testMetadata{
		Field:         "test",
		ChainSelector: chainsel.APTOS_MAINNET.Selector,
	}

	// put it in an `any` type and use As to convert it back
	var a any = orig
	typed, err := As[testMetadata](a)
	require.NoError(t, err)
	require.Equal(t, orig, typed)
}

func TestAs_UseNumberForAnyField(t *testing.T) {
	t.Parallel()

	type testWithAny struct {
		Value any `json:"value"`
	}

	const expected = "16015286601757825753"
	src := map[string]any{"value": json.Number(expected)}

	typed, err := As[testWithAny](src)
	require.NoError(t, err)

	n, ok := typed.Value.(json.Number)
	require.True(t, ok, "Value should decode as json.Number")
	require.Equal(t, expected, n.String())
}
