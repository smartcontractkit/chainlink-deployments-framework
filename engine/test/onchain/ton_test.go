package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewContainerLoaderTon(t *testing.T) {
	t.Parallel()

	loader := NewTonContainerLoader()
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := getTestSelectorsByFamily(chainselectors.FamilyTon)
	assert.Equal(t, want, loader.selectors)

	// Note: We can't actually call the factory without starting containers,
	// but we can verify it exists and has the correct signature
	require.NotNil(t, loader.factory)
	assert.IsType(t, ChainFactory(nil), loader.factory)
}
