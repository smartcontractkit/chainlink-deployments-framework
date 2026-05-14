package evm_test

import (
	"math/big"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	mcmsbindings "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
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
			selector:   chainsel.ETHEREUM_MAINNET.Selector,
			wantString: "ethereum-mainnet (5009297550715157269)",
			wantName:   chainsel.ETHEREUM_MAINNET.Name,
			wantFamily: chainsel.FamilyEVM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := evm.Chain{
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

	c := evm.Chain{Selector: chainsel.ETHEREUM_MAINNET.Selector}
	got, err := c.NetworkType()
	require.NoError(t, err)
	assert.Equal(t, chainsel.NetworkTypeMainnet, got)

	c = evm.Chain{Selector: 0}
	_, err = c.NetworkType()
	require.Error(t, err)
}

func TestChainMetadata_IsNetworkType(t *testing.T) {
	t.Parallel()

	c := evm.Chain{Selector: chainsel.ETHEREUM_MAINNET.Selector}

	assert.True(t, c.IsNetworkType(chainsel.NetworkTypeMainnet))
	assert.False(t, c.IsNetworkType(chainsel.NetworkTypeTestnet))
}

func TestChain_ReadOnly(t *testing.T) {
	t.Parallel()

	anvilKey := blockchain.DefaultAnvilPrivateKey
	anvilConfig := provider.CTFAnvilChainProviderConfig{
		Name:                  "anvil-main-blockchain",
		Once:                  &sync.Once{},
		ConfirmFunctor:        provider.ConfirmFuncGeth(1 * time.Second),
		Image:                 "f4hrenh9it/foundry:latest",
		Port:                  strconv.Itoa(getFreePortForIntegration(t)),
		DeployerTransactorGen: provider.TransactorFromRaw(anvilKey),
		T:                     t,
	}
	chainSelector := chainsel.GETH_TESTNET.Selector
	anvilProvider := provider.NewCTFAnvilChainProvider(chainSelector, anvilConfig)
	chain, err := anvilProvider.Initialize(t.Context())
	require.NoError(t, err)

	roChain, err := chain.ReadOnly()
	require.NoError(t, err)
	roEVMChain, ok := roChain.(evm.Chain)
	require.True(t, ok)
	evmChain, ok := chain.(evm.Chain)
	require.True(t, ok)

	// read with read-only client should work
	balance, err := roEVMChain.Client.BalanceAt(t.Context(), gethcommon.HexToAddress(blockchain.DefaultAnvilPublicKey), nil)
	require.NoError(t, err)
	require.Equal(t, lo.Must(new(big.Int).SetString("10000000000000000000000", 10)), balance)

	// write with read-write client should work
	_, _, _, err = mcmsbindings.DeployCallProxy(evmChain.DeployerKey, evmChain.Client, gethcommon.Address{}) //nolint:dogsled // not interested in all return values
	require.NoError(t, err)

	// write with read-only client should fail
	_, _, _, err = mcmsbindings.DeployCallProxy(roEVMChain.DeployerKey, roEVMChain.Client, gethcommon.Address{}) //nolint:dogsled // not interested in all return values
	require.ErrorContains(t, err, "Out of gas: gas required exceeds allowance")
}

// ----- helpers -----

func getFreePortForIntegration(t *testing.T) int {
	t.Helper()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}
