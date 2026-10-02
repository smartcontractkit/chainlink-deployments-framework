package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewStellarContainerLoader(t *testing.T) {
	t.Parallel()

	loader := NewStellarContainerLoader()
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	wantSelectors := getTestSelectorsByFamily(chainselectors.FamilyStellar)
	assert.Equal(t, wantSelectors, loader.selectors)

	// Note: We can't actually call the factory without starting containers,
	// but we can verify it exists.
	require.NotNil(t, loader.factory)
}
