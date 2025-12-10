package onchain

import (
	"sync"
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tonprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/provider"
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
}

func Test_NewTonContainerLoaderWithConfig(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := tonprov.CTFChainProviderConfig{
		Once:  &once,
		Image: "ghcr.io/neodix42/mylocalton-docker:latest",
		CustomEnv: map[string]string{
			"VERSION_CAPABILITIES":        "11",
			"NEXT_BLOCK_GENERATION_DELAY": "0.5",
		},
	}

	loader := NewTonContainerLoaderWithConfig(config)
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := getTestSelectorsByFamily(chainselectors.FamilyTon)
	assert.Equal(t, want, loader.selectors)

	// Factory should be configured with the provided config
	require.NotNil(t, loader.factory)
}
