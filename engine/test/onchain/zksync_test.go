package onchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewContainerLoaderZKSync(t *testing.T) {
	t.Parallel()

	loader := NewZKSyncContainerLoader()
	require.NotNil(t, loader)

	// Should have the predefined ZKSync selectors
	require.NotNil(t, loader.selectors)
	want := zksyncSelectors
	assert.Equal(t, want, loader.selectors)

	// Note: We can't actually call the factory without starting containers,
	// but we can verify it exists and has the correct signature
	require.NotNil(t, loader.factory)
}
