package onchain

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/samber/lo"
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
	want := append(
		[]uint64{chainselectors.GETH_TESTNET.Selector},
		getTestSelectorsByFamily(chainselectors.FamilyEVM)...,
	)
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
		AdminAccount:          lo.Must(crypto.HexToECDSA("26a6528a1d63fffc4ce9f109d407bb584f3fce17a09033608fcb31c47c163756")),
	}

	loader := NewEVMSimLoaderWithConfig(config)
	require.NotNil(t, loader)

	// Should have the same selectors as getTestSelectorsByFamily returns
	require.NotNil(t, loader.selectors)
	want := append(
		[]uint64{chainselectors.GETH_TESTNET.Selector},
		getTestSelectorsByFamily(chainselectors.FamilyEVM)...,
	)
	assert.Equal(t, want, loader.selectors)

	// Factory should be configured with the provided config
	require.NotNil(t, loader.factory)
}
