package provider

import (
	"testing"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

func Test_ZkSyncRPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	rpc := evm.RPC{
		Name:               "Test",
		HTTPURL:            "http://localhost:8545",
		WSURL:              "ws://localhost:8546",
		PreferredURLScheme: evm.URLSchemePreferenceHTTP,
	}

	confirmFuncGeth := ConfirmFuncGeth(10 * time.Millisecond)

	tests := []struct {
		name    string
		config  ZkSyncRPCChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        confirmFuncGeth,
			},
		},
		{
			name: "missing deployer transactor generator",
			config: ZkSyncRPCChainProviderConfig{
				ZkSyncSignerGen: ZKSyncSignerRandom(),
				RPCs:            []evm.RPC{rpc},
				ConfirmFunctor:  confirmFuncGeth,
			},
			wantErr: "deployer transactor generator is required",
		},
		{
			name: "missing signer generator",
			config: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        confirmFuncGeth,
			},
			wantErr: "signer generator is required",
		},
		{
			name: "missing confirm functor",
			config: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
			},
			wantErr: "confirm functor is required",
		},
		{
			name: "missing rpcs",
			config: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
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
func Test_ZkSyncRPCChainProvider_Initialize(t *testing.T) {
	var (
		chainSelector = chain_selectors.TEST_1000.Selector
		existingChain = &evm.Chain{}
		configPath    = writeSethConfigFile(t)
	)

	// Create a mock RPC server that always returns a valid response for eth_blockNumber
	mockSrv := newFakeRPCServer(t)

	// Define a general RPC configuration for use
	rpc := evm.RPC{
		Name:               "Test",
		HTTPURL:            mockSrv.URL,
		PreferredURLScheme: evm.URLSchemePreferenceHTTP,
	}

	gethConfirmFunc := ConfirmFuncGeth(1 * time.Second)

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfig        ZkSyncRPCChainProviderConfig
		giveExistingChain *evm.Chain // Use this to simulate an already initialized chain
		wantErr           string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
		},
		{
			name:         "valid initialization with logger",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				Logger:                logger.Test(t),
				ConfirmFunctor:        gethConfirmFunc,
			},
		},
		{
			name:         "valid initialization with seth config",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
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
			giveConfig:   ZkSyncRPCChainProviderConfig{},
			wantErr:      "deployer transactor generator is required",
		},
		{
			name:         "fails getting chain ID from selector",
			giveSelector: 1, // Invalid selector
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to get chain ID from selector",
		},
		{
			name:         "fails to generate deployer transactor",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: &alwaysFailingTransactorGenerator{},
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to generate deployer key",
		},
		{
			name:         "fails to create multi client",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{{}},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to create multi-client",
		},
		{
			name:         "fails to parse seth configuration",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor: ConfirmFuncSeth(
					rpc.HTTPURL, 10*time.Millisecond, []string{}, "nonexistent.toml",
				), // Invalid path
			},
			wantErr: "no such file or directory",
		},
		{
			name:         "fails to generate confirm function",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       ZKSyncSignerRandom(),
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        &alwaysFailingConfirmFunctor{},
			},
			wantErr: "failed to generate confirm function",
		},
		{
			name:         "fails to generate zkSync signer",
			giveSelector: chainSelector,
			giveConfig: ZkSyncRPCChainProviderConfig{
				DeployerTransactorGen: TransactorRandom(),
				ZkSyncSignerGen:       &alwaysFailingZKSyncSignerGenerator{},
				RPCs:                  []evm.RPC{rpc},
				ConfirmFunctor:        gethConfirmFunc,
			},
			wantErr: "failed to generate zkSync signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
			p := NewZkSyncRPCChainProvider(tt.giveSelector, tt.giveConfig)

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
				assert.True(t, gotChain.IsZkSyncVM)
				assert.NotNil(t, gotChain.ClientZkSyncVM)
				assert.NotNil(t, gotChain.DeployerKeyZkSyncVM)
				assert.NotNil(t, gotChain.SignHash)
			}
		})
	}
}

func Test_ZkSyncRPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &ZkSyncRPCChainProvider{}
	assert.Equal(t, "ZkSync EVM RPC Chain Provider", p.Name())
}

func Test_ZkSyncRPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &ZkSyncRPCChainProvider{selector: chain_selectors.TEST_1000.Selector}
	assert.Equal(t, chain_selectors.TEST_1000.Selector, p.ChainSelector())
}

func Test_ZkSyncRPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &evm.Chain{}

	p := &ZkSyncRPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, p.BlockChain())
}
