package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryDatastore(t *testing.T) {
	t.Parallel()

	config := MemoryDataStoreConfig{}
	store := NewMemoryDataStore(t, config)
	defer store.Close()
	assert.NotNil(t, store)
}
