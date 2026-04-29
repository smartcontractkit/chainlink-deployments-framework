package sui_test

import (
	"sync"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

func TestChain_ChainInfot(t *testing.T) {
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
			selector:   chainsel.SUI_MAINNET.Selector,
			wantString: "sui-mainnet (17529533435026248318)",
			wantName:   chainsel.SUI_MAINNET.Name,
			wantFamily: chainsel.FamilySui,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := sui.Chain{
				ChainMetadata: sui.ChainMetadata{Selector: tt.selector},
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func TestChain_ReadOnly(t *testing.T) {
	t.Parallel()

	testPrivateKey := "0x417e8a7e38a3e5deaaf817d544719db736864384e0f5b513ffb511050cdcaa16"

	ctfProvider := provider.NewCTFChainProvider(t, chainsel.SUI_LOCALNET.Selector, provider.CTFChainProviderConfig{
		Once:              &sync.Once{},
		DeployerSignerGen: provider.AccountGenPrivateKey(testPrivateKey),
	})
	chain, err := ctfProvider.Initialize(t.Context())
	require.NoError(t, err)

	roChain, err := chain.ReadOnly()
	require.NoError(t, err)
	roSuiChain, ok := roChain.(sui.Chain)
	require.True(t, ok)
	suiChain, ok := chain.(sui.Chain)
	require.True(t, ok)

	signer, err := suiChain.Signer.GetAddress()
	require.NoError(t, err)
	roSigner, err := roSuiChain.Signer.GetAddress()
	require.NoError(t, err)

	// read with read-only client should work
	balanceReq := models.SuiXGetBalanceRequest{Owner: signer, CoinType: "0x2::sui::SUI"}
	balance, err := roSuiChain.Client.SuiXGetBalance(t.Context(), balanceReq)
	require.NoError(t, err)
	require.Equal(t, "1000000000000", balance.TotalBalance)

	// write with read-write client should work
	opts := &bind.CallOpts{WaitForExecution: true, GasBudget: pointer.To(uint64(400_000_000)), Signer: suiChain.Signer}
	_, _, err = mcms.PublishMCMS(t.Context(), opts, suiChain.Client, suiChain.URL)
	require.NoError(t, err)

	// write with read-only client should fail
	opts = &bind.CallOpts{WaitForExecution: true, GasBudget: pointer.To(uint64(400_000_000)), Signer: roSuiChain.Signer}
	_, _, err = mcms.PublishMCMS(t.Context(), opts, roSuiChain.Client, roSuiChain.URL)
	require.ErrorContains(t, err, "Cannot find gas coin for signer address "+roSigner+" with amount sufficient for the required gas budget")
}
