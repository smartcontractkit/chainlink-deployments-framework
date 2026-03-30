package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldverification "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func init() {
	interVerifyDelay = 0
}

const verificationHookEnv = "hooktest"

// newDomainWithExplorerNetwork writes domain.yaml + networks.yaml so config.LoadNetworks resolves
// a single mainnet EVM network with BlockExplorer.URL set to explorerURL (e.g. httptest server).
func newDomainWithExplorerNetwork(t *testing.T, chainSelector uint64, explorerURL string) domain.Domain {
	t.Helper()

	dom := domain.NewDomain(t.TempDir(), "test")
	require.NoError(t, os.MkdirAll(dom.ConfigNetworksDirPath(), 0755))

	networkYAML := fmt.Sprintf(`networks:
  - type: mainnet
    chain_selector: %d
    block_explorer:
      url: %q
    rpcs:
      - http_url: http://127.0.0.1:8545
`, chainSelector, explorerURL)
	require.NoError(t, os.WriteFile(dom.ConfigNetworksFilePath("networks.yaml"), []byte(networkYAML), 0600))

	domainYAML := fmt.Sprintf(`environments:
  %s:
    network_types:
      - mainnet
`, verificationHookEnv)
	require.NoError(t, os.WriteFile(dom.ConfigDomainFilePath(), []byte(domainYAML), 0600))

	return dom
}

func writeEnvDatastoreWithRefs(t *testing.T, dom domain.Domain, refs []datastore.AddressRef) {
	t.Helper()

	envDir := dom.EnvDir(verificationHookEnv)
	require.NoError(t, os.MkdirAll(envDir.DataStoreDirPath(), 0755))

	refsJSON, err := json.Marshal(refs)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(envDir.AddressRefsFilePath(), refsJSON, 0600))
	require.NoError(t, os.WriteFile(envDir.ChainMetadataFilePath(), []byte("[]"), 0600))
	require.NoError(t, os.WriteFile(envDir.ContractMetadataFilePath(), []byte("[]"), 0600))
	require.NoError(t, os.WriteFile(envDir.EnvMetadataFilePath(), []byte("null"), 0600))
}

func TestNewVerifyDeployedEVMContractsPostHook_Definition(t *testing.T) {
	t.Parallel()

	h := NewVerifyDeployedEVMContractsPostHook(domain.Domain{}, &mockProvider{})
	require.Equal(t, verifyDeployedContractsHookName, h.Name)
	require.Equal(t, changeset.Abort, h.FailurePolicy)
	require.NotNil(t, h.Func)
}

func TestNewRequireVerifiedEVMContractsPreHook_Definition(t *testing.T) {
	t.Parallel()

	h := NewRequireVerifiedEVMContractsPreHook(domain.Domain{}, &mockProvider{})
	require.Equal(t, requireVerifiedEnvContractsHookName, h.Name)
	require.Equal(t, changeset.Abort, h.FailurePolicy)
	require.NotNil(t, h.Func)
}

func TestVerifyDeployed_PostHook_SkipsWhenApplyFailed(t *testing.T) {
	t.Parallel()

	h := NewVerifyDeployedEVMContractsPostHook(domain.Domain{}, &mockProvider{})
	err := h.Func(t.Context(), changeset.PostHookParams{
		Err: errors.New("apply failed"),
		Env: changeset.HookEnv{Name: "staging", Logger: logger.Test(t)},
		Output: deployment.ChangesetOutput{
			DataStore: datastore.NewMemoryDataStore(),
		},
	})
	require.NoError(t, err)
}

func TestVerifyDeployed_PostHook_SkipsWhenDataStoreNil(t *testing.T) {
	t.Parallel()

	h := NewVerifyDeployedEVMContractsPostHook(domain.Domain{}, &mockProvider{})
	err := h.Func(t.Context(), changeset.PostHookParams{
		Env: changeset.HookEnv{Name: "staging", Logger: logger.Test(t)},
		Output: deployment.ChangesetOutput{
			DataStore: nil,
		},
	})
	require.NoError(t, err)
}

func TestVerifyDeployed_PostHook_LoadNetworksError(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	h := NewVerifyDeployedEVMContractsPostHook(dom, &mockProvider{})
	err := h.Func(t.Context(), changeset.PostHookParams{
		Env: changeset.HookEnv{Name: "staging", Logger: logger.Test(t)},
		Output: deployment.ChangesetOutput{
			DataStore: datastore.NewMemoryDataStore(),
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "verify hook: load networks")
}

func TestRequireVerified_PreHook_DataStoreError(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	h := NewRequireVerifiedEVMContractsPreHook(dom, &mockProvider{})
	err := h.Func(t.Context(), changeset.PreHookParams{
		Env: changeset.HookEnv{Name: "staging", Logger: logger.Test(t)},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "require verified pre-hook: load datastore")
}

func TestRequireVerified_PreHook_LoadNetworksError(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	h := NewRequireVerifiedEVMContractsPreHook(dom, &mockProvider{})
	// Minimal env layout so DataStore() succeeds: empty files skip JSON decode in loadDataStore.
	envDir := dom.EnvDir("staging")
	require.NoError(t, mkdirAllAndWrite(t, envDir.AddressRefsFilePath()))
	require.NoError(t, mkdirAllAndWrite(t, envDir.ChainMetadataFilePath()))
	require.NoError(t, mkdirAllAndWrite(t, envDir.ContractMetadataFilePath()))
	require.NoError(t, mkdirAllAndWrite(t, envDir.EnvMetadataFilePath()))

	err := h.Func(t.Context(), changeset.PreHookParams{
		Env: changeset.HookEnv{Name: "staging", Logger: logger.Test(t)},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "require verified pre-hook: load networks")
}

func TestIterateEVMVerifiers_EmptyNetworks(t *testing.T) {
	t.Parallel()

	ds := datastore.NewMemoryDataStore().Seal()
	cfg := cfgnet.NewConfig(nil)
	err := iterateEVMVerifiers(t.Context(), ds, cfg, &mockProvider{}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			t.Fatal("step should not run")
			return nil
		},
	)
	require.NoError(t, err)
}

func TestIterateEVMVerifiers_SkipsUnknownChainSelector(t *testing.T) {
	t.Parallel()

	ds := datastore.NewMemoryDataStore().Seal()
	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chainsel.APTOS_MAINNET.Selector,
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})
	err := iterateEVMVerifiers(t.Context(), ds, cfg, &mockProvider{}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			t.Fatal("step should not run")
			return nil
		},
	)
	require.NoError(t, err)
}

func TestIterateEVMVerifiers_NoAPIKeyRecordsVerifierError(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chain.Selector,
		BlockExplorer: cfgnet.BlockExplorer{},
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})

	err := iterateEVMVerifiers(t.Context(), mds.Seal(), cfg, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
		},
	}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			t.Fatal("step should not run when verifier construction fails")
			return nil
		},
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "etherscan API key not configured")
}

func TestIterateEVMVerifiers_SkipsGetInputsError(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chain.Selector,
		BlockExplorer: cfgnet.BlockExplorer{APIKey: "k"},
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})

	err := iterateEVMVerifiers(t.Context(), mds.Seal(), cfg, &mockProvider{
		getInputsErr: errors.New("unknown type"),
	}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			t.Fatal("step should not run when GetInputs fails")
			return nil
		},
	)
	require.NoError(t, err)
}

func TestIterateEVMVerifiers_SkipsNilVersion(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       nil,
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chain.Selector,
		BlockExplorer: cfgnet.BlockExplorer{APIKey: "k"},
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})

	err := iterateEVMVerifiers(t.Context(), mds.Seal(), cfg, &mockProvider{}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			t.Fatal("step should not run when version is nil")
			return nil
		},
	)
	require.NoError(t, err)
}

func TestIterateEVMVerifiers_StepError(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chain.Selector,
		BlockExplorer: cfgnet.BlockExplorer{APIKey: "k"},
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})

	stepErr := errors.New("step failed")
	err := iterateEVMVerifiers(t.Context(), mds.Seal(), cfg, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
		},
	}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			return stepErr
		},
	)
	require.ErrorIs(t, err, stepErr)
}

func TestIterateEVMVerifiers_StepSuccess(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	cfg := cfgnet.NewConfig([]cfgnet.Network{{
		Type:          cfgnet.NetworkTypeMainnet,
		ChainSelector: chain.Selector,
		BlockExplorer: cfgnet.BlockExplorer{APIKey: "k"},
		RPCs:          []cfgnet.RPC{{HTTPURL: "http://localhost"}},
	}})

	var stepCalls int
	err := iterateEVMVerifiers(t.Context(), mds.Seal(), cfg, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
		},
	}, logger.Test(t), "test",
		func(_ context.Context, _ cldverification.Verifiable, _ datastore.AddressRef, _ chainsel.Chain) error {
			stepCalls++
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, stepCalls)
}

func TestRequireVerified_PreHook_Sourcify_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Files have not been found"))
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	dom := newDomainWithExplorerNetwork(t, chain.Selector, server.URL)
	writeEnvDatastoreWithRefs(t, dom, []datastore.AddressRef{{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}})

	h := NewRequireVerifiedEVMContractsPreHook(dom, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
		},
	})
	err := h.Func(t.Context(), changeset.PreHookParams{
		Env: changeset.HookEnv{Name: verificationHookEnv, Logger: logger.Test(t)},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "not verified on explorer")
}

func TestRequireVerified_PreHook_Sourcify_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "full"})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	dom := newDomainWithExplorerNetwork(t, chain.Selector, server.URL)
	writeEnvDatastoreWithRefs(t, dom, []datastore.AddressRef{{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}})

	h := NewRequireVerifiedEVMContractsPreHook(dom, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
		},
	})
	err := h.Func(t.Context(), changeset.PreHookParams{
		Env: changeset.HookEnv{Name: verificationHookEnv, Logger: logger.Test(t)},
	})
	require.NoError(t, err)
}

func TestVerifyDeployed_PostHook_Blockscout_CallsVerifyWhenNotVerified(t *testing.T) {
	t.Parallel()

	var verifyPOSTs int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodPost && action == "verify":
			verifyPOSTs++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	dom := newDomainWithExplorerNetwork(t, chain.Selector, server.URL)

	mds := datastore.NewMemoryDataStore()
	require.NoError(t, mds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chain.Selector,
		Type:          "MyContract",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0x0000000000000000000000000000000000000001",
	}))

	h := NewVerifyDeployedEVMContractsPostHook(dom, &mockProvider{
		metadata: evm.SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "MyContract",
			Sources: map[string]any{
				"MyContract.sol": map[string]any{"content": "pragma solidity 0.8.19; contract MyContract {}"},
			},
		},
	})
	err := h.Func(t.Context(), changeset.PostHookParams{
		Env: changeset.HookEnv{Name: verificationHookEnv, Logger: logger.Test(t)},
		Output: deployment.ChangesetOutput{
			DataStore: mds,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, verifyPOSTs, "Verify should POST to explorer when IsVerified is false")
}

func mkdirAllAndWrite(t *testing.T, path string) error {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))

	return os.WriteFile(path, nil, 0600)
}

type mockProvider struct {
	getInputsErr error
	metadata     evm.SolidityContractMetadata
}

func (m *mockProvider) GetInputs(_ datastore.ContractType, _ *semver.Version) (evm.SolidityContractMetadata, error) {
	if m.getInputsErr != nil {
		return evm.SolidityContractMetadata{}, m.getInputsErr
	}

	return m.metadata, nil
}
