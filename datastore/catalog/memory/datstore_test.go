package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDatastore(t *testing.T) {
	t.Parallel()

	config := MemoryDataStoreConfig{}
	store, err := NewMemoryDataStore(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()
	assert.NotNil(t, store)
}
