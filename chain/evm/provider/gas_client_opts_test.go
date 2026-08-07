package provider

import (
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func Test_multiClientOpts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         *gas.Config
		wantLen     int
		wantMaxCap  uint64
		checkBaseFn bool
	}{
		{name: "nil config", cfg: nil, wantLen: 1, checkBaseFn: true},
		{name: "empty config", cfg: &gas.Config{}, wantLen: 1, checkBaseFn: true},
		{
			name:        "with max tx gas limit",
			cfg:         &gas.Config{MaxTxGasLimit: gas.EIP7825MaxTxGasLimit},
			wantLen:     2,
			wantMaxCap:  gas.EIP7825MaxTxGasLimit,
			checkBaseFn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var baseCalled bool
			base := []func(*rpcclient.MultiClient){
				func(*rpcclient.MultiClient) { baseCalled = true },
			}

			opts := multiClientOpts(tt.cfg, base)
			require.Len(t, opts, tt.wantLen)
			_ = append(opts, func(*rpcclient.MultiClient) {})
			require.Len(t, base, 1, "multiClientOpts must not alias the input slice")

			mc := &rpcclient.MultiClient{}
			for _, opt := range opts {
				opt(mc)
			}
			if tt.checkBaseFn {
				require.True(t, baseCalled)
			}
			require.Equal(t, tt.wantMaxCap, mc.MaxTxGasLimit())
		})
	}
}

//nolint:paralleltest // Initialize cannot run in parallel due to seth log initialization race
func Test_providerInitialize_gasConfig(t *testing.T) {
	chainSelector := chainsel.TEST_1000.Selector

	mockSrv := newFakeRPCServer(t)
	rpc := rpcclient.RPC{
		Name:               "Test",
		HTTPURL:            mockSrv.URL,
		PreferredURLScheme: rpcclient.URLSchemePreferenceHTTP,
	}
	confirm := ConfirmFuncGeth(1 * time.Second)

	tests := []struct {
		name      string
		zkSync    bool
		gasConfig *gas.Config
		wantErr   string
		assert    func(t *testing.T, chain evm.Chain)
	}{
		{
			name:      "RPC applies gas limit defaults",
			gasConfig: &gas.Config{DefaultGasLimit: 7_500_000},
			assert: func(t *testing.T, chain evm.Chain) {
				t.Helper()
				require.Equal(t, uint64(7_500_000), chain.DeployerKey.GasLimit)
			},
		},
		{
			name:      "RPC applies max tx gas limit to client",
			gasConfig: &gas.Config{MaxTxGasLimit: gas.EIP7825MaxTxGasLimit},
			assert: func(t *testing.T, chain evm.Chain) {
				t.Helper()
				requireMultiClientMaxTxGasLimit(t, chain.Client, gas.EIP7825MaxTxGasLimit)
			},
		},
		{
			name: "RPC caps default gas limit at max tx gas limit",
			gasConfig: &gas.Config{
				DefaultGasLimit: 20_000_000,
				MaxTxGasLimit:   gas.EIP7825MaxTxGasLimit,
			},
			assert: func(t *testing.T, chain evm.Chain) {
				t.Helper()
				require.Equal(t, gas.EIP7825MaxTxGasLimit, chain.DeployerKey.GasLimit)
				requireMultiClientMaxTxGasLimit(t, chain.Client, gas.EIP7825MaxTxGasLimit)
			},
		},
		{
			name:      "RPC fails when apply defaults errors",
			gasConfig: &gas.Config{DefaultGasPriceWei: 210_000_000_000},
			wantErr:   "failed to apply gas defaults",
		},
		{
			name:      "ZkSync applies gas limit defaults to EVM deployer key",
			zkSync:    true,
			gasConfig: &gas.Config{DefaultGasLimit: 7_500_000},
			assert: func(t *testing.T, chain evm.Chain) {
				t.Helper()
				require.True(t, chain.IsZkSyncVM)
				require.Equal(t, uint64(7_500_000), chain.DeployerKey.GasLimit)
			},
		},
		{
			name:      "ZkSync applies max tx gas limit to client",
			zkSync:    true,
			gasConfig: &gas.Config{MaxTxGasLimit: gas.EIP7825MaxTxGasLimit},
			assert: func(t *testing.T, chain evm.Chain) {
				t.Helper()
				requireMultiClientMaxTxGasLimit(t, chain.Client, gas.EIP7825MaxTxGasLimit)
			},
		},
		{
			name:      "ZkSync fails when apply defaults errors",
			zkSync:    true,
			gasConfig: &gas.Config{DefaultGasPriceWei: 210_000_000_000},
			wantErr:   "failed to apply gas defaults",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got evm.Chain
				err error
			)

			if tt.zkSync {
				p := NewZkSyncRPCChainProvider(chainSelector, ZkSyncRPCChainProviderConfig{
					DeployerTransactorGen: TransactorRandom(),
					ZkSyncSignerGen:       ZKSyncSignerRandom(),
					RPCs:                  []rpcclient.RPC{rpc},
					ConfirmFunctor:        confirm,
					GasConfig:             tt.gasConfig,
				})
				blockchain, initErr := p.Initialize(t.Context())
				err = initErr
				got, _ = blockchain.(evm.Chain)
			} else {
				p := NewRPCChainProvider(chainSelector, RPCChainProviderConfig{
					DeployerTransactorGen: TransactorRandom(),
					RPCs:                  []rpcclient.RPC{rpc},
					ConfirmFunctor:        confirm,
					GasConfig:             tt.gasConfig,
				})
				blockchain, initErr := p.Initialize(t.Context())
				err = initErr
				got, _ = blockchain.(evm.Chain)
			}

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}

func requireMultiClientMaxTxGasLimit(t *testing.T, client evm.OnchainClient, want uint64) {
	t.Helper()

	mc, ok := client.(*rpcclient.MultiClient)
	require.True(t, ok, "expected chain client to be *rpcclient.MultiClient")
	require.Equal(t, want, mc.MaxTxGasLimit())
}
