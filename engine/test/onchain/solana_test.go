package onchain

import (
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewContainerLoaderSolana(t *testing.T) {
	t.Parallel()

	programsPath := "/test/programs"
	programIDs := map[string]string{
		"test_program": "11111111111111111111111111111111",
	}

	loader := NewSolanaContainerLoader(programsPath, programIDs)
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := getTestSelectorsByFamily(chainselectors.FamilySolana)
	assert.Equal(t, want, loader.selectors)

	// Note: We can't actually call the factory without starting containers,
	// but we can verify it exists and has the correct signature
	require.NotNil(t, loader.factory)
	assert.IsType(t, ChainFactory(nil), loader.factory)
}
