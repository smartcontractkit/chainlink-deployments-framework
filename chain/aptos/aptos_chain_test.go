package aptos_test

import (
	"testing"
	"time"

	aptosgosdk "github.com/aptos-labs/aptos-go-sdk"
	gethcommon "github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	aptosmcms "github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/testutils"
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
			selector:   chainsel.APTOS_MAINNET.Selector,
			wantString: "aptos-mainnet (4741433654826277614)",
			wantName:   chainsel.APTOS_MAINNET.Name,
			wantFamily: chainsel.FamilyAptos,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := aptos.Chain{
				Selector: tt.selector,
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func TestChainMetadata_NetworkType(t *testing.T) {
	t.Parallel()

	c := aptos.Chain{Selector: chainsel.APTOS_MAINNET.Selector}
	got, err := c.NetworkType()
	require.NoError(t, err)
	assert.Equal(t, chainsel.NetworkTypeMainnet, got)

	c = aptos.Chain{Selector: 0}
	_, err = c.NetworkType()
	require.Error(t, err)
}

func TestChainMetadata_IsNetworkType(t *testing.T) {
	t.Parallel()

	c := aptos.Chain{Selector: chainsel.APTOS_MAINNET.Selector}

	assert.True(t, c.IsNetworkType(chainsel.NetworkTypeMainnet))
	assert.False(t, c.IsNetworkType(chainsel.NetworkTypeTestnet))
}

func TestChain_ReadOnly(t *testing.T) {
	t.Parallel()

	ctfProvider := provider.NewCTFChainProvider(t, chainsel.APTOS_LOCALNET.Selector, provider.CTFChainProviderConfig{
		Once:              testutils.DefaultNetworkOnce,
		DeployerSignerGen: provider.AccountGenCTFDefault(),
	})
	chain, err := ctfProvider.Initialize(t.Context())
	require.NoError(t, err)

	roChain, err := chain.ReadOnly()
	require.NoError(t, err)
	roAptosChain, ok := roChain.(aptos.Chain)
	require.True(t, ok)
	aptosChain, ok := chain.(aptos.Chain)
	require.True(t, ok)

	// read with read-only client should work
	accountAddress := aptosgosdk.AccountAddress(gethcommon.HexToHash(blockchain.DefaultAptosAccount))
	balance, err := roAptosChain.Client.AccountAPTBalance(accountAddress)
	require.NoError(t, err)
	require.Equal(t, uint64(1100100000000), balance)

	// write with read-write client should work
	seed := aptosmcms.DefaultSeed + time.Now().String()
	_, tx, _, err := aptosmcms.DeployToResourceAccount(aptosChain.DeployerSigner, aptosChain.Client, seed)
	require.NoError(t, err)
	_, err = roAptosChain.Client.WaitForTransaction(tx.Hash)
	require.NoError(t, err)

	// write with read-only client should fail
	_, _, _, err = aptosmcms.DeployToResourceAccount(roAptosChain.DeployerSigner, roAptosChain.Client, seed) //nolint:dogsled
	require.ErrorContains(t, err, "INSUFFICIENT_BALANCE_FOR_TRANSACTION_FEE")
}
