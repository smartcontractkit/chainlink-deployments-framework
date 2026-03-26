package onchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	suiprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/provider"
)

func Test_NewSuiContainerLoader(t *testing.T) {
	t.Parallel()

	loader := NewSuiContainerLoader()
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	wantSelectors := getTestSelectorsByFamily(chainselectors.FamilySui)
	assert.Equal(t, wantSelectors, loader.selectors)

	// Note: We can't actually call the factory without starting containers,
	// but we can verify it exists.
	require.NotNil(t, loader.factory)
}

func Test_NewSuiContainerLoaderWithConfig(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	seed := make([]byte, ed25519.SeedSize)
	_, err := rand.Read(seed)
	require.NoError(t, err)

	seeded := ed25519.NewKeyFromSeed(seed)
	seedBytes := seeded[:32]
	testPrivateKey := hex.EncodeToString(seedBytes)

	var once sync.Once

	config := suiprov.CTFChainProviderConfig{
		Once:              &once,
		DeployerSignerGen: suiprov.AccountGenPrivateKey(testPrivateKey),
	}

	loader := NewSuiContainerLoaderWithConfig(config)
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	wantSelectors := getTestSelectorsByFamily(chainselectors.FamilySui)
	assert.Equal(t, wantSelectors, loader.selectors)

	// Factory should be configured with the provided config
	require.NotNil(t, loader.factory)
}
