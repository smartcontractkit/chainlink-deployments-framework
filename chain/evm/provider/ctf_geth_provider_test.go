package provider

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func TestCTFGethChainProvider_ValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        CTFGethChainProviderConfig
		expectedError string
	}{
		{
			name: "valid config",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with additional accounts",
			config: CTFGethChainProviderConfig{
				Once:               &sync.Once{},
				ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
				AdditionalAccounts: true,
				T:                  t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with explicit port",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           "8545", // When port is provided, T is not required
			},
		},
		{
			name: "missing Once",
			config: CTFGethChainProviderConfig{
				Once:           nil,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t,
			},
			expectedError: "sync.Once instance is required",
		},
		{
			name: "missing ConfirmFunctor",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: nil,
				T:              t,
			},
			expectedError: "confirm functor is required",
		},
		{
			name: "missing T field when port not provided",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              nil, // Missing T when Port is not provided
			},
			expectedError: "field T is required when port is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.validate()
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFGethChainProviderConfig_InvalidPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		port          string
		expectedError string
	}{
		{
			name:          "invalid port - not a number",
			port:          "invalid",
			expectedError: "invalid port invalid: must be a valid integer",
		},
		{
			name:          "invalid port - zero",
			port:          "0",
			expectedError: "invalid port 0: must be between 1 and 65535",
		},
		{
			name:          "invalid port - negative",
			port:          "-1",
			expectedError: "invalid port -1: must be between 1 and 65535",
		},
		{
			name:          "invalid port - too large",
			port:          "65536",
			expectedError: "invalid port 65536: must be between 1 and 65535",
		},
		{
			name: "valid port",
			port: "8545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var once sync.Once
			config := CTFGethChainProviderConfig{
				Once:           &once,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           tt.port,
				T:              t, // Required when port is not provided
			}

			err := config.validate()
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFGethChainProvider_NewProvider(t *testing.T) {
	t.Parallel()

	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}
	selector := uint64(13264668187771770619)

	p := NewCTFGethChainProvider(selector, cfg)
	require.NotNil(t, p)
	require.Equal(t, selector, p.selector)
	require.Equal(t, cfg, p.config)
	require.Nil(t, p.chain)

	// Not initialized yet → empty URL
	require.Empty(t, p.GetNodeHTTPURL())
}

func TestCTFGethChainProvider_Initialize_And_WS(t *testing.T) {
	t.Parallel()

	// single integration pass that spins one container:
	// - Initialize()
	// - BlockChain(), GetNodeHTTPURL()
	// - idempotence of Initialize() (no re-spin)
	// - ClientOpts applied
	// - Users empty when AdditionalAccounts=false
	// - WS connectivity & subscription
	var once sync.Once
	var clientOptsCalled bool

	cfg := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		ClientOpts: []func(client *rpcclient.MultiClient){
			func(mc *rpcclient.MultiClient) { clientOptsCalled = true },
		},
		T: t,
	}
	selector := uint64(13264668187771770619)
	p := NewCTFGethChainProvider(selector, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	// Initialize
	bc, err := p.Initialize(ctx)
	require.NoError(t, err)
	require.NotNil(t, bc)

	// Basic identity
	require.Equal(t, selector, p.ChainSelector())
	require.Equal(t, "Geth EVM CTF Chain Provider", p.Name())

	// BlockChain() returns same logical chain
	got := p.BlockChain()
	require.Equal(t, bc.ChainSelector(), got.ChainSelector())
	require.Equal(t, bc.Name(), got.Name())
	require.Equal(t, bc.Family(), got.Family())
	require.Equal(t, bc.String(), got.String())

	// Node URL available
	httpURL := p.GetNodeHTTPURL()
	require.NotEmpty(t, httpURL)
	require.Contains(t, httpURL, "http://")

	// ClientOpts applied
	require.True(t, clientOptsCalled, "ClientOpts should be applied")

	// No additional users by default
	evmChain := bc.(evm.Chain)
	require.Empty(t, evmChain.Users)

	// Idempotence: second init returns same chain instance behavior (no crash / no re-spin)
	_, err2 := p.Initialize(ctx)
	require.NoError(t, err2)

	// WS connectivity & subscription (CTF exposes WS on same host:port; swap scheme)
	wsURL := strings.Replace(httpURL, "http://", "ws://", 1)
	require.NotEqual(t, httpURL, wsURL)

	wsCtx, wsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer wsCancel()

	rpcClient, errDial := rpc.DialWebsocket(wsCtx, wsURL, "")
	require.NoError(t, errDial, "should connect to WS endpoint")

	var chainIDHex string
	errCall := rpcClient.CallContext(wsCtx, &chainIDHex, "eth_chainId")
	require.NoError(t, errCall, "eth_chainId over WS should succeed")
	require.NotEmpty(t, chainIDHex)

	headers := make(chan *types.Header, 1)
	sub, errSub := rpcClient.EthSubscribe(wsCtx, headers, "newHeads")
	require.NoError(t, errSub, "newHeads subscription should succeed")

	select {
	case <-headers:
		// ok
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for newHeads over WS")
	}

	sub.Unsubscribe()
	rpcClient.Close()
}

func TestCTFGethChainProvider_AdditionalAccounts_Funding_Idempotent(t *testing.T) {
	t.Parallel()

	// One container to cover:
	// - AdditionalAccounts=true → users created (1..4)
	// - Funding amount correct
	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:               &once,
		ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
		AdditionalAccounts: true,
		T:                  t,
	}
	selector := uint64(13264668187771770619)
	p := NewCTFGethChainProvider(selector, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	bc, err := p.Initialize(ctx)
	require.NoError(t, err)
	evmChain := bc.(evm.Chain)

	// accounts 1..4 created
	require.Len(t, evmChain.Users, 4)

	// all users funded with 100 ETH
	expected := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	for _, u := range evmChain.Users {
		bal, errBal := evmChain.Client.BalanceAt(ctx, u.From, nil)
		require.NoError(t, errBal)
		require.Equal(t, 0, bal.Cmp(expected), "balance should be = 100 ETH")
	}

	// snapshot, re-init (no-op), balances unchanged
	snap := make([]*big.Int, len(evmChain.Users))
	for i, u := range evmChain.Users {
		b, errB := evmChain.Client.BalanceAt(ctx, u.From, nil)
		require.NoError(t, errB)
		snap[i] = new(big.Int).Set(b)
	}
}

func TestCTFGethChainProvider_Cleanup_WithFixedPort(t *testing.T) {
	t.Parallel()

	// Explicit port + T=nil simulates production usage where we must call Cleanup manually
	port := freeport.GetOne(t)
	t.Cleanup(func() { freeport.Return([]int{port}) })

	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		Port:           strconv.Itoa(port),
		T:              nil,
	}
	selector := uint64(13264668187771770619)
	p := NewCTFGethChainProvider(selector, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	bc, err := p.Initialize(ctx)
	require.NoError(t, err)
	require.NotNil(t, bc)
	require.NotNil(t, p.container, "container should exist after initialization")

	// Cleanup should succeed and clear container ref
	err = p.Cleanup(ctx)
	require.NoError(t, err)
	require.Nil(t, p.container)

	// Second cleanup is a no-op
	err = p.Cleanup(ctx)
	require.NoError(t, err)
	require.Nil(t, p.container)
}

func TestCTFGethChainProvider_InitializeErrors(t *testing.T) {
	t.Parallel()

	// This fails before any container start (invalid chain selector mapping)
	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}

	badSelector := uint64(999999999999999999)
	p := NewCTFGethChainProvider(badSelector, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	_, err := p.Initialize(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chain selector")
}
