package mcms

import (
	"encoding/json"
	"io"
	"maps"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	chainsel "github.com/smartcontractkit/chain-selectors"
	ctf "github.com/smartcontractkit/chainlink-testing-framework/framework"
	ctfchain "github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmsevmbindings "github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	evmchain "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldfchainprovider "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfconfig "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cldfconfignet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	testruntime "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/runtime"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	integrationDomainName = "testdomain"
	integrationEnvName    = "testnet"
)

var _, integrationModulePath, _, _ = runtime.Caller(0)

func Test_executeFork_Integration(t *testing.T) { //nolint:paralleltest
	lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)

	domainsRoot := filepath.Clean(filepath.Join(integrationModulePath, "..", "testdata", "domains"))
	domain := cldfdomain.NewDomain(domainsRoot, integrationDomainName)
	t.Setenv("ONCHAIN_EVM_DEPLOYER_KEY", ctfchain.DefaultAnvilPrivateKey)
	domainConfig, err := cldfconfig.Load(domain, integrationEnvName, lggr)
	require.NoError(t, err)

	// initialize anvil container with main blockchain
	anvilConfig := cldfchainprovider.CTFAnvilChainProviderConfig{
		Name:                  "anvil-main-blockchain",
		Once:                  &sync.Once{},
		ConfirmFunctor:        cldfchainprovider.ConfirmFuncGeth(3 * time.Minute),
		Image:                 "f4hrenh9it/foundry:latest",
		Port:                  strconv.Itoa(getFreePortForIntegration(t)),
		DeployerTransactorGen: cldfchainprovider.TransactorFromRaw(domainConfig.Env.Onchain.EVM.DeployerKey),
		T:                     t,
	}
	provider := cldfchainprovider.NewCTFAnvilChainProvider(chainsel.GETH_TESTNET.Selector, anvilConfig)
	evmChain, err := provider.Initialize(t.Context())
	require.NoError(t, err)

	saveDomainNetworkConfigForIntegration(t, &domain, integrationEnvName, domainConfig, provider, anvilConfig.Port)

	env, err := cldfenv.Load(t.Context(), domain, integrationEnvName)
	require.NoError(t, err)
	env.BlockChains = cldfchain.NewBlockChains(map[uint64]cldfchain.BlockChain{
		chainsel.GETH_TESTNET.Selector: evmChain,
	})
	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]

	mcmAddress, timelockAddress, callProxyAddress, env := deployMCMSForIntegration(t, env)
	saveChangesetOutputsForIntegration(t, domain, env, "deploy-mcms")

	timelockProposal, mcmProposal := testTimelockProposalForIntegration(t, chain, timelockAddress, mcmAddress)

	forkedEnv, err := cldfenv.LoadFork(t.Context(), domain, env.Name, nil,
		cldfenv.WithLogger(lggr), cldfenv.OnlyLoadChainsFor([]uint64{chain.Selector}),
		cldfenv.WithAnvilKeyAsDeployer(), cldfenv.WithoutJD())
	require.NoError(t, err)

	proposalCtx, err := analyzer.NewDefaultProposalContext(env)
	require.NoError(t, err)

	tests := []struct {
		name   string
		cfg    *forkConfig
		assert func(err error)
	}{
		{
			name: "success",
			cfg: &forkConfig{
				kind:             mcmstypes.KindTimelockProposal,
				proposal:         mcmProposal,
				timelockProposal: &timelockProposal,
				chainSelector:    chain.Selector,
				blockchains:      forkedEnv.BlockChains,
				envStr:           env.Name,
				env:              env,
				fork:             true,
				forkedEnv:        forkedEnv,
				proposalCtx:      proposalCtx,
			},
			assert: func(err error) {
				require.NoError(t, err)
				require.Equal(t, 1, logs.FilterMessageSnippet("MCM.setRoot() - success").Len())
				require.Equal(t, 1, logs.FilterMessageSnippet("MCM.execute() - success").Len())
				require.Equal(t, 1, logs.FilterMessageSnippet("Timelock.execute() - success").Len())

				sendTxLogs := logs.FilterMessage("sending on-chain transaction").AllUntimed()
				require.Len(t, sendTxLogs, 3)
				require.Equal(t, sendTxLogs[0].ContextMap(), map[string]any{ //nolint:testifylint
					"from":  ctfchain.DefaultAnvilPublicKey,
					"to":    mcmAddress,
					"value": "0",
					"data":  "7cc38b289f3238149e22b29dfc2efbce2cb0fe486074260593a5845eb8b9e209dcc76ecd000000000000000000000000000000000000000000000000000000007c245eff00000000000000000000000000000000000000000000000000000000000005390000000000000000000000005fbdb2315678afecb367f032d93f642f64180aa3000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001600000000000000000000000000000000000000000000000000000000000000001da9a3f10e279f0c947e2347b15cdafa99d0946ad70edf1de3e3f1fb3870cec8f0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000001bdd38e4c5069fd0f522fbe74f823e7c59171c4472c62250d3f537868be75b212056bb3a98355115f41e32a250530cf6f06b17311d2cae58577ab6b8651186f4f5",
				})
				require.Equal(t, sendTxLogs[1].ContextMap(), map[string]any{ //nolint:testifylint
					"from":  ctfchain.DefaultAnvilPublicKey,
					"to":    mcmAddress,
					"value": "0",
					"data":  "b759d685000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000002a000000000000000000000000000000000000000000000000000000000000005390000000000000000000000005fbdb2315678afecb367f032d93f642f64180aa30000000000000000000000000000000000000000000000000000000000000000000000000000000000000000cf7ed3acca5a467e9e704c703e8d87f634fb0fc9000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000164a944142d000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000007c245eff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000002307800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000105727eee6ea33160de91251c73bd0fcbb3d72c4869961bd3164ed6cdfee2d1e8",
				})
				require.Equal(t, sendTxLogs[2].ContextMap(), map[string]any{ //nolint:testifylint
					"from":  ctfchain.DefaultAnvilPublicKey,
					"to":    callProxyAddress,
					"value": "0",
					"data":  "6ceef480000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000007c245eff0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb922660000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000023078000000000000000000000000000000000000000000000000000000000000",
				})
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			err := executeFork(t.Context(), lggr, tt.cfg, true)

			tt.assert(err)
			logs.TakeAll() // clear logs
		})
	}
}

// --- helpers and fixtures ---

func getFreePortForIntegration(t *testing.T) int {
	t.Helper()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func mutableDataStoreForIntegration(t *testing.T, ds datastore.DataStore) datastore.MutableDataStore {
	t.Helper()

	mutDS := datastore.NewMemoryDataStore()
	err := mutDS.Merge(ds)
	require.NoError(t, err)

	return mutDS
}

func deployMCMSForIntegration(t *testing.T, env cldf.Environment) (string, string, string, cldf.Environment) {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	signerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]
	mcmAddress, env := deployMcmForIntegration(t, env, chain, signerAddress)
	timelockAddress, callProxyAddress, env := deployTimelockAndCallProxyForIntegration(t, env, chain, []string{mcmAddress}, nil, nil)

	return mcmAddress, timelockAddress, callProxyAddress, env
}

func deployMcmForIntegration(
	t *testing.T, env cldf.Environment, chain evmchain.Chain, signerAddress common.Address,
) (string, cldf.Environment) {
	t.Helper()

	mcmAddress := common.Address{}
	changeset := cldf.CreateChangeSet(
		func(e cldf.Environment, config struct{}) (cldf.ChangesetOutput, error) {
			ds := datastore.NewMemoryDataStore()
			var tx *ethtypes.Transaction

			// deploy mcm
			var mcmContract *mcmsevmbindings.ManyChainMultiSig
			var err error
			mcmAddress, tx, mcmContract, err = mcmsevmbindings.DeployManyChainMultiSig(chain.DeployerKey, chain.Client)
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       mcmAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "ManyChainMultiSig",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)

			// set config
			tx, err = mcmContract.SetConfig(chain.DeployerKey,
				[]common.Address{signerAddress}, // signerAddresses
				[]uint8{0},                      // signerGroups
				[32]uint8{1},                    // groupQuorums
				[32]uint8{0},                    // groupParents
				true,
			)
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			return cldf.ChangesetOutput{DataStore: ds}, nil
		},
		func(e cldf.Environment, config struct{}) error { return nil }, // verify,
	)

	task := testruntime.ChangesetTask(changeset, struct{}{})
	runtime := testruntime.NewFromEnvironment(env)
	err := runtime.Exec(task)
	require.NoError(t, err)

	return mcmAddress.Hex(), env
}

func deployTimelockAndCallProxyForIntegration(
	t *testing.T, env cldf.Environment, chain evmchain.Chain, proposers []string, bypassers []string, cancellers []string,
) (string, string, cldf.Environment) {
	t.Helper()

	callProxyAddress := common.Address{}
	timelockAddress := common.Address{}
	changeset := cldf.CreateChangeSet(
		func(e cldf.Environment, config struct{}) (cldf.ChangesetOutput, error) {
			ds := datastore.NewMemoryDataStore()
			ab := cldf.NewMemoryAddressBook()
			var tx *ethtypes.Transaction
			var err error

			// deploy call proxy
			callProxyAddress, tx, _, err = mcmsevmbindings.DeployCallProxy(chain.DeployerKey, chain.Client, common.Address{})
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       callProxyAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "CallProxy",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)
			err = ab.Save(chain.Selector, callProxyAddress.Hex(), cldf.MustTypeAndVersionFromString("CallProxy 1.0.0"))
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			// deploy timelock
			timelockAddress, tx, _, err = mcmsevmbindings.DeployRBACTimelock(chain.DeployerKey, chain.Client, big.NewInt(0),
				chain.DeployerKey.From,
				lo.Map(proposers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
				[]common.Address{callProxyAddress},
				lo.Map(bypassers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
				lo.Map(cancellers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
			)
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       timelockAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "RBACTimelock",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)
			err = ab.Save(chain.Selector, timelockAddress.Hex(), cldf.MustTypeAndVersionFromString("RBACTimelock 1.0.0"))
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			return cldf.ChangesetOutput{AddressBook: ab, DataStore: ds}, nil
		},
		func(e cldf.Environment, config struct{}) error { return nil }, // verify,
	)

	task := testruntime.ChangesetTask(changeset, struct{}{})
	runtime := testruntime.NewFromEnvironment(env)
	err := runtime.Exec(task)
	require.NoError(t, err)

	return timelockAddress.Hex(), callProxyAddress.Hex(), env
}

func testTimelockProposalForIntegration(
	t *testing.T,
	chain evmchain.Chain,
	timelockAddress string,
	mcmAddress string,
) (mcms.TimelockProposal, mcms.Proposal) {
	t.Helper()

	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(2082758399).
		SetDescription("test timelock proposal").
		SetOverridePreviousRoot(true).
		SetAction(mcmstypes.TimelockActionSchedule).
		AddTimelockAddress(mcmstypes.ChainSelector(chain.Selector), timelockAddress).
		AddChainMetadata(mcmstypes.ChainSelector(chain.Selector), mcmstypes.ChainMetadata{MCMAddress: mcmAddress}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: mcmstypes.ChainSelector(chain.Selector),
			Transactions: []mcmstypes.Transaction{{
				To:               chain.DeployerKey.From.Hex(),
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			}},
		}).Build()
	require.NoError(t, err)

	mcmProposal, _, err := timelockProposal.Convert(t.Context(), map[mcmstypes.ChainSelector]mcmssdk.TimelockConverter{
		mcmstypes.ChainSelector(chain.Selector): &mcmsevmsdk.TimelockConverter{},
	})
	require.NoError(t, err)

	return *timelockProposal, mcmProposal
}

func saveDomainNetworkConfigForIntegration(
	t *testing.T, domain *cldfdomain.Domain, envName string, domainConfig *cldfconfig.Config,
	provider *cldfchainprovider.CTFAnvilChainProvider, containerPort string,
) {
	t.Helper()

	containerURL := provider.GetNodeHTTPURL()
	networkAliases, err := provider.Container.NetworkAliases(t.Context())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(networkAliases[ctf.DefaultNetworkName]), 1)

	networks := domainConfig.Networks.Networks()
	require.Len(t, networks, 1)
	require.Len(t, networks[0].RPCs, 1)

	networks[0].RPCs[0].HTTPURL = containerURL
	networks[0].Metadata = &cldfconfignet.EVMMetadata{AnvilConfig: &cldfconfignet.AnvilConfig{
		Image:          "f4hrenh9it/foundry:latest",
		Port:           uint64(getFreePortForIntegration(t)), //nolint:gosec
		ArchiveHTTPURL: "http://" + networkAliases[ctf.DefaultNetworkName][0] + ":" + containerPort,
	}}

	cldfNetConfig, err := cldfconfignet.NewConfig(networks).MarshalYAML()
	require.NoError(t, err)
	networkConfigYaml, err := yaml.Marshal(cldfNetConfig)
	require.NoError(t, err)

	filePath := domain.ConfigNetworksFilePath(envName + ".yaml")
	backupPath := filePath + ".bkp"
	copyFileForIntegration(t, filePath, backupPath)
	err = os.WriteFile(filePath, networkConfigYaml, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() {
		copyFileForIntegration(t, backupPath, filePath)
		err = os.Remove(backupPath)
		require.NoError(t, err)
	})
}

func saveChangesetOutputsForIntegration(t *testing.T, domain cldfdomain.Domain, env cldf.Environment, changesetName string) {
	t.Helper()

	envDir := domain.EnvDir(env.Name)
	addressBookBkpPath := envDir.AddressBookFilePath() + ".bkp"
	copyFileForIntegration(t, envDir.AddressBookFilePath(), addressBookBkpPath)
	addressRefsBkpPath := envDir.AddressRefsFilePath() + ".bkp"
	copyFileForIntegration(t, envDir.AddressRefsFilePath(), addressRefsBkpPath)

	err := envDir.ArtifactsDir().SaveChangesetOutput(changesetName, cldf.ChangesetOutput{
		AddressBook: env.ExistingAddresses, //nolint:staticcheck
		DataStore:   mutableDataStoreForIntegration(t, env.DataStore),
	})
	require.NoError(t, err)
	err = envDir.MergeChangesetAddressBook(changesetName, "")
	require.NoError(t, err)
	err = envDir.MergeChangesetDataStore(changesetName, "")
	require.NoError(t, err)

	t.Cleanup(func() {
		copyFileForIntegration(t, addressBookBkpPath, envDir.AddressBookFilePath())
		copyFileForIntegration(t, addressRefsBkpPath, envDir.AddressRefsFilePath())
		err = os.Remove(addressBookBkpPath)
		require.NoError(t, err)
		err = os.Remove(addressRefsBkpPath)
		require.NoError(t, err)
		err = os.RemoveAll(envDir.ArtifactsDir().ArtifactsDirPath())
		require.NoError(t, err)
	})
}

func copyFileForIntegration(t *testing.T, src, dest string) {
	t.Helper()

	srcFile, err := os.Open(src)
	require.NoError(t, err)
	defer srcFile.Close()
	destFile, err := os.Create(dest)
	require.NoError(t, err)
	defer destFile.Close()
	_, err = io.Copy(destFile, srcFile)
	require.NoError(t, err)
}
