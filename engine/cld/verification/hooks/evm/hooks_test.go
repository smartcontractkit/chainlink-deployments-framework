package evm

import (
	"context"
	"errors"
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
	require.NoError(t, mkdirAllAndWrite(t, envDir.AddressRefsFilePath(), ""))
	require.NoError(t, mkdirAllAndWrite(t, envDir.ChainMetadataFilePath(), ""))
	require.NoError(t, mkdirAllAndWrite(t, envDir.ContractMetadataFilePath(), ""))
	require.NoError(t, mkdirAllAndWrite(t, envDir.EnvMetadataFilePath(), ""))

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

func mkdirAllAndWrite(t *testing.T, path, content string) error {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	return os.WriteFile(path, []byte(content), 0o644)
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
