package onchain

import (
	"testing"
	"time"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewEVMSimLoaderEVM(t *testing.T) {
	t.Parallel()

	loader := NewEVMSimLoader()
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := getTestSelectorsByFamily(chainselectors.FamilyEVM)
	assert.Equal(t, want, loader.selectors)

	// Note: We can't actually call the factory without starting simulated backends,
	// but we can verify it exists.
	require.NotNil(t, loader.factory)
}

func Test_NewEVMSimLoaderEVMWithConfig(t *testing.T) {
	t.Parallel()

	config := EVMSimLoaderConfig{
		NumAdditionalAccounts: 5,
		BlockTime:             time.Second,
	}

	loader := NewEVMSimLoaderWithConfig(config)
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := getTestSelectorsByFamily(chainselectors.FamilyEVM)
	assert.Equal(t, want, loader.selectors)

	// Factory should be configured with the provided config
	require.NotNil(t, loader.factory)
}
