package tron_test

import (
	"sync"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/testdata"
)

func TestChain_ChainInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
		wantFamily string
	}{
		{
			name:       "returns correct info",
			selector:   chainsel.TRON_MAINNET.Selector,
			wantString: "tron-mainnet (1546563616611573945)",
			wantName:   chainsel.TRON_MAINNET.Name,
			wantFamily: chainsel.FamilyTron,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tron.Chain{
				ChainMetadata: tron.ChainMetadata{Selector: tt.selector},
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func Test_DefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfirmRetryOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultConfirmRetryOptions()
		assert.Equal(t, uint(180), opts.RetryAttempts)
		assert.Equal(t, 500*time.Millisecond, opts.RetryDelay)
	})

	t.Run("DefaultDeployOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultDeployOptions()
		assert.Equal(t, 100_000_000, opts.FeeLimit)
		assert.Equal(t, 100, opts.CurPercent)
		assert.Equal(t, 50_000_000, opts.OeLimit)
		assert.Equal(t, tron.DefaultConfirmRetryOptions(), opts.ConfirmRetryOptions)
	})

	t.Run("DefaultTriggerOptions", func(t *testing.T) {
		t.Parallel()
		opts := tron.DefaultTriggerOptions()
		assert.Equal(t, int32(10_000_000), opts.FeeLimit)
		assert.Equal(t, int64(0), opts.TAmount)
		assert.Equal(t, tron.DefaultConfirmRetryOptions(), opts.ConfirmRetryOptions)
	})
}

func TestChain_ReadOnly(t *testing.T) {
	t.Parallel()

	// anvilKey := blockchain.DefaultAnvilPrivateKey
	signerGen, err := provider.SignerGenCTFDefault()
	require.NoError(t, err)
	ctfConfig := provider.CTFChainProviderConfig{
		Once:              &sync.Once{},
		DeployerSignerGen: signerGen,
	}
	chainSelector := chainsel.GETH_TESTNET.Selector
	chain, err := provider.NewCTFChainProvider(t, chainSelector, ctfConfig).Initialize(t.Context())
	require.NoError(t, err)

	tronChain, ok := chain.(tron.Chain)
	require.True(t, ok)
	roChain, err := chain.ReadOnly()
	require.NoError(t, err)
	roTronChain, ok := roChain.(tron.Chain)
	require.True(t, ok)

	// read with read-only client should work
	account, err := roTronChain.Client.GetAccount(tronChain.Address)
	require.NoError(t, err)
	require.Equal(t, int64(10000000000), account.Balance)

	// write with read-write client should work
	deployOptions := tron.DefaultDeployOptions()
	deployOptions.FeeLimit = 1_000_000_000
	_, _, err = tronChain.DeployContractAndConfirm(t.Context(), "LinkToken",
		testdata.LinkTokenMetaDataABI, testdata.LinkTokenMetaDataBIN, nil, deployOptions)
	require.NoError(t, err)

	// write with read-only client should fail
	_, _, err = roTronChain.DeployContractAndConfirm(t.Context(), "LinkToken",
		testdata.LinkTokenMetaDataABI, testdata.LinkTokenMetaDataBIN, nil, deployOptions)
	require.ErrorContains(t, err, "account [")
	require.ErrorContains(t, err, "] does not exist")
}
