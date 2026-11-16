package provider

import (
	"errors"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	ctfchain "github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestNewSubprocessAnvilChainProvider(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)

	tests := []struct {
		name     string
		lggr     logger.Logger
		selector uint64
		config   SubprocessAnvilChainProviderConfig
		want     *subprocessAnvilChainProvider
	}{
		{
			name:     "success",
			lggr:     lggr,
			selector: 1,
			config: SubprocessAnvilChainProviderConfig{
				Command:               "custom-anvil",
				Args:                  []string{"--chain-id", "1234"},
				RpcURLs:               []string{"http://example.com"},
				NumAdditionalAccounts: 2,
			},
			want: &subprocessAnvilChainProvider{
				selector: 1,
				url:      "",
				chain:    nil,
				config: SubprocessAnvilChainProviderConfig{
					Command:               "custom-anvil",
					Args:                  []string{"--chain-id", "1234"},
					RpcURLs:               []string{"http://example.com"},
					NumAdditionalAccounts: 2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewSubprocessAnvilChainProvider(tt.lggr, tt.selector, tt.config)

			require.Empty(t, cmp.Diff(tt.want, got,
				cmpopts.EquateComparable(),
				cmp.AllowUnexported(subprocessAnvilChainProvider{}),
				cmpopts.IgnoreFields(subprocessAnvilChainProvider{}, "logger", "freeports")))
		})
	}
}

func Test_subprocessAnvilChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("anvil"); err != nil {
		t.Skip("anvil binary not found in PATH, skipping test")
	}

	ports, err := freeport.Take(2)
	require.NoError(t, err)
	mainChainPort := ports[0]
	forkChainPort := ports[1]
	mainChainRpcURL := "http://127.0.0.1:" + strconv.Itoa(mainChainPort)

	// start main chain
	process := exec.CommandContext(t.Context(), "anvil", "--chain-id", "1337", "--port", strconv.Itoa(mainChainPort)) //nolint:gosec
	err = process.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = process.Cancel() })
	go func() { _ = process.Wait() }()
	err = waitRpcReady(t.Context(), mainChainRpcURL)
	require.NoError(t, err)

	tests := []struct {
		name     string
		selector uint64
		config   SubprocessAnvilChainProviderConfig
		portsFn  func(int) ([]int, error)
		want     fchain.BlockChain
		wantURL  string
		wantErr  string
	}{
		{
			name:     "success",
			selector: 3379446385462418246, // chain-id 1337
			portsFn:  func(int) ([]int, error) { return []int{forkChainPort}, nil },
			config: SubprocessAnvilChainProviderConfig{
				RpcURLs:               []string{mainChainRpcURL},
				DeployerTransactorGen: TransactorFromRaw(ctfchain.DefaultAnvilPrivateKey),
				ConfirmFunctor:        ConfirmFuncGeth(30 * time.Second),
			},
			want: &evm.Chain{
				Selector:    3379446385462418246,
				DeployerKey: &bind.TransactOpts{From: common.HexToAddress(ctfchain.DefaultAnvilPublicKey)},
			},
			wantURL: "http://localhost:" + strconv.Itoa(forkChainPort),
		},
		{
			name: "failure: no rpc urls provided",
			config: SubprocessAnvilChainProviderConfig{
				DeployerTransactorGen: TransactorFromRaw(ctfchain.DefaultAnvilPrivateKey),
			},
			wantErr: "invalid subprocess anvil chain provider config: no rpc URLs provided",
		},
		{
			name: "failure: no deployer transactor gen provided",
			config: SubprocessAnvilChainProviderConfig{
				RpcURLs: []string{mainChainRpcURL},
			},
			wantErr: "invalid subprocess anvil chain provider config: no deployer transactor gen provided",
		},
		{
			name:     "failure: unknown chain selector",
			selector: 9999999999999999999,
			config: SubprocessAnvilChainProviderConfig{
				RpcURLs:               []string{mainChainRpcURL},
				DeployerTransactorGen: TransactorFromRaw(ctfchain.DefaultAnvilPrivateKey),
			},
			wantErr: "failed to get chain details for selector: unknown chain selector 9999999999999999999",
		},
		{
			name:     "failure: error when getting free port",
			selector: 3379446385462418246,
			portsFn:  func(int) ([]int, error) { return nil, errors.New("free-port-error") },
			config: SubprocessAnvilChainProviderConfig{
				RpcURLs:               []string{mainChainRpcURL},
				DeployerTransactorGen: TransactorFromRaw(ctfchain.DefaultAnvilPrivateKey),
			},
			wantErr: "failed to get free port: free-port-error",
		},
		{
			name:     "failure: subprocess returns an error",
			selector: 3379446385462418246,
			portsFn:  func(int) ([]int, error) { return []int{forkChainPort}, nil },
			config: SubprocessAnvilChainProviderConfig{
				RpcURLs:               []string{mainChainRpcURL},
				DeployerTransactorGen: TransactorFromRaw(ctfchain.DefaultAnvilPrivateKey),
				Command:               "__invalid__command__",
			},
			wantErr: "#5: failed to start anvil subprocess: exec: \"__invalid__command__\": executable file not found in $PATH", // "#5": 5th retry
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider := &subprocessAnvilChainProvider{
				logger:    logger.Nop(),
				selector:  tt.selector,
				config:    tt.config,
				freeports: tt.portsFn,
			}

			got, err := provider.Initialize(t.Context())

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got,
					cmpopts.IgnoreFields(evm.Chain{}, "Client", "Confirm", "SignHash", "Users"),
					cmpopts.IgnoreFields(bind.TransactOpts{}, "Context", "Signer")))
				require.Equal(t, tt.wantURL, provider.GetNodeHTTPURL())
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
