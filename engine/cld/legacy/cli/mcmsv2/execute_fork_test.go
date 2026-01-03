package mcmsv2

import (
	"maps"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfchainprovider "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	cldfconfig "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// const (
// 	evmPrivateKey = "0xc4a07ce3bd71783bee3bd9d662d95e11d285f978f00dc9fa59ab8ff481bc325b"
// 	evmPublicKey  = "0xaFe8dD9F9A0eAE11e3ADA9BdD15aE3dBA4D17E73"
// )

const (
	domainName = "testdomain"
	envName    = "testnet"
)

var _, modulePath, _, _ = runtime.Caller(0)

func Test_executeFork(t *testing.T) {
	t.Parallel()

	lggr, _ := logger.TestObserved(t, zapcore.DebugLevel)

	domainsRoot := filepath.Clean(filepath.Join(modulePath, "..", "testdata", "domains"))
	domain := cldfdomain.NewDomain(domainsRoot, domainName)
	domainConfig, err := cldfconfig.Load(domain, envName, lggr)
	require.NoError(t, err)

	anvilConfig := cldfchainprovider.CTFAnvilChainProviderConfig{
		Once:                  &sync.Once{},
		ConfirmFunctor:        cldfchainprovider.ConfirmFuncGeth(3 * time.Minute),
		Image:                 "f4hrenh9it/foundry:latest",
		Port:                  strconv.Itoa(getFreePort(t)),
		DeployerTransactorGen: cldfchainprovider.TransactorFromRaw(domainConfig.Env.Onchain.EVM.DeployerKey),
		T:                     t,
	}
	provider := cldfchainprovider.NewCTFAnvilChainProvider(chainsel.GETH_TESTNET.Selector, anvilConfig)
	evmChain, err := provider.Initialize(t.Context())
	require.NoError(t, err)

	networks := domainConfig.Networks.Networks()
	require.Len(t, networks, 1)
	require.Len(t, networks[0].RPCs, 1)
	networks[0].RPCs[0].HTTPURL = provider.GetNodeHTTPURL()
	networkConfigYaml, err := yaml.Marshal(networks)
	require.NoError(t, err)
	networkConfigPath := domain.ConfigNetworksFilePath(envName + ".yaml")
	err = os.WriteFile(networkConfigPath, networkConfigYaml, 0o600)
	require.NoError(t, err)

	enV, err := cldfenv.Load(t.Context(), domain, envName)
	require.NoError(t, err)
	env := &enV
	envDir := domain.EnvDir(env.Name)
	env.BlockChains = cldfchain.NewBlockChains(map[uint64]cldfchain.BlockChain{
		chainsel.GETH_TESTNET.Selector: evmChain,
	})

	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	// signer := mcms.NewPrivateKeySigner(privateKey)
	signerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]
	mcmAddress, env := deployMcm(t, env, chain, signerAddress)
	timelockAddress, _, env := deployTimelockAndCallProxy(t, env, chain, []string{mcmAddress}, nil, nil)
	timelockProposal := testTimelockProposal(t, chain, timelockAddress, mcmAddress)
	mcmProposal, _, err := timelockProposal.Convert(t.Context(), map[mcmstypes.ChainSelector]mcmssdk.TimelockConverter{
		mcmstypes.ChainSelector(chain.Selector): &mcmsevmsdk.TimelockConverter{},
	})
	require.NoError(t, err)
	err = envDir.MergeMigrationAddressBook("deploy-mcms", "")
	require.NoError(t, err)
	err = envDir.MergeMigrationDataStore("deploy-mcms", "")
	require.NoError(t, err)

	forkedEnv, err := cldfenv.LoadFork(t.Context(), domain, env.Name, nil,
		cldfenv.WithLogger(lggr), cldfenv.OnlyLoadChainsFor([]uint64{chain.Selector}),
		cldfenv.WithAnvilKeyAsDeployer(), cldfenv.WithoutJD())
	require.NoError(t, err)

	proposalCtx, err := analyzer.NewDefaultProposalContext(*env)
	require.NoError(t, err)

	tests := []struct {
		name       string
		cfg        *cfgv2
		testSigner bool
		wantErr    string
	}{
		{
			name: "success",
			cfg: &cfgv2{
				kind:             mcmstypes.KindTimelockProposal,
				proposal:         mcmProposal,
				timelockProposal: &timelockProposal,
				chainSelector:    chain.Selector,
				blockchains:      env.BlockChains,
				envStr:           env.Name,
				env:              *env,
				fork:             true,
				forkedEnv:        forkedEnv,
				proposalCtx:      proposalCtx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := executeFork(t.Context(), lggr, tt.cfg, tt.testSigner)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// --- helpers and fixtures ---

func getFreePort(t *testing.T) int {
	t.Helper()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}
