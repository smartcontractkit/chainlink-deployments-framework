package provider

import (
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	rpc := rpcclient.RPC{
		Name:               "Test",
		HTTPURL:            "http://localhost:8545",
		WSURL:              "ws://localhost:8546",
		PreferredURLScheme: rpcclient.URLSchemePreferenceHTTP,
	}

	confirmFuncGeth := ConfirmFuncGeth(10 * time.Millisecond)

	tests := []struct {
		name    string
		config  RPCChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor:        confirmFuncGeth,
			},
		},
		{
			name: "missing deployer transactor generator",
			config: RPCChainProviderConfig{
				RPCs:           []rpcclient.RPC{rpc},
				ConfirmFunctor: confirmFuncGeth,
			},
			wantErr: "deployer transactor generator is required",
		},
		{
			name: "missing confirm functor",
			config: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
			},
			wantErr: "confirm functor is required",
		},
		{
			name: "missing rpcs",
			config: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ConfirmFunctor:        confirmFuncGeth,
			},
			wantErr: "at least one RPC is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
func Test_RPCChainProvider_Initialize(t *testing.T) {
	var (
		chainSelector = chainsel.TEST_1000.Selector
		existingChain = &evm.Chain{}
		configPath    = writeSethConfigFile(t)
	)

	// Create a mock RPC server that always returns a valid response for eth_blockNumber
	mockSrv := newFakeRPCServer(t)

	// Define a general RPC configuration for use
	rpc := rpcclient.RPC{
		Name:               "Test",
		HTTPURL:            mockSrv.URL,
		PreferredURLScheme: rpcclient.URLSchemePreferenceHTTP,
	}

	gethConfirmFunc := ConfirmFuncGeth(1 * time.Second)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfig        RPCChainProviderConfig
		giveExistingChain *evm.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				UsersTransactorGen: []SignerGenerator{
					TransactorRandom(),
				},
				ConfirmFunctor: gethConfirmFunc,
			},
		},
		{
			name:         "valid initialization with logger",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				Logger:                logger.Test(t),
				ConfirmFunctor:        gethConfirmFunc,
			},
		},
		{
			name:         "valid initialization with seth config",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor: ConfirmFuncSeth(
					rpc.HTTPURL, 10*time.Millisecond, []string{}, configPath,
				),
			},
		},
		{
			name:              "returns an already initialized chain",
			giveSelector:      chainSelector,
			giveExistingChain: existingChain,
		},
		{
			name:         "fails config validation",
			giveSelector: chainSelector,
			giveConfig:   RPCChainProviderConfig{},
			wantErr:      "deployer transactor generator is required",
		},
		{
			name:         "fails getting chain ID from selector",
			giveSelector: 1, // Invalid selector
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to get chain ID from selector",
		},
		{
			name:         "fails to generate deployer transactor",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: &alwaysFailingTransactorGenerator{},
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to generate deployer key",
		},
		{
			name:         "fails to generate users transactors",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				UsersTransactorGen: []SignerGenerator{
					&alwaysFailingTransactorGenerator{}, // This will always fail
				},
				ConfirmFunctor: gethConfirmFunc,
			},
			wantErr: "failed to generate user transactor",
		},
		{
			name:         "fails to create multi client",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{{}},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to create multi-client",
		},
		{
			name:         "fails to parse seth configuration",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor: ConfirmFuncSeth(
					rpc.HTTPURL, 10*time.Millisecond, []string{}, "nonexistent.toml",
				), // Invalid path
			},
			wantErr: "no such file or directory",
		},
		{
			name:         "fails to generate confirm function",
			giveSelector: chainSelector,
			giveConfig: RPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []rpcclient.RPC{rpc},
				ConfirmFunctor:        &alwaysFailingConfirmFunctor{},
			},
			wantErr: "failed to generate confirm function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
			p := NewRPCChainProvider(tt.giveSelector, tt.giveConfig)

			if tt.giveExistingChain != nil {
				p.chain = tt.giveExistingChain
			}

			got, err := p.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p.chain)

				gotChain, ok := got.(evm.Chain)
				require.True(t, ok, "expected got to be of type evm.Chain")

				// For the already initialized chain case, we can skip the rest of the checks
				if tt.giveExistingChain != nil {
					return
				}

				// Otherwise, check the fields of the chain
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.NotNil(t, gotChain.Client)
				assert.NotNil(t, gotChain.DeployerKey)
				assert.NotNil(t, gotChain.Confirm)
				assert.Len(t, gotChain.Users, len(tt.giveConfig.UsersTransactorGen))
				assert.NotNil(t, gotChain.SignHash)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "EVM RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chainsel.TEST_1000.Selector}
	assert.Equal(t, chainsel.TEST_1000.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &evm.Chain{}

	p := &RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
