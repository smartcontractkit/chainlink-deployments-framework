package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestVerifyEnv_RunVerifyEnv_EmptyConfig(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return datastore.NewMemoryDataStore().Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_NetworkLoaderError(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return nil, errors.New("network load failed")
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return datastore.NewMemoryDataStore().Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.Error(t, err)
	require.Equal(t, "failed to load network configuration: network load failed", err.Error())
}

func TestVerifyEnv_RunVerifyEnv_DataStoreLoaderError(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return nil, errors.New("datastore load failed")
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.Error(t, err)
	require.Equal(t, "failed to get datastore: datastore load failed", err.Error())
}

func TestVerifyEnv_RunVerifyEnv_WithNetworkNoAddresses(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return datastore.NewMemoryDataStore().Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_NetworkFilter(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return datastore.NewMemoryDataStore().Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{
		environment:    "staging",
		filterNetworks: "999999999999999",
	}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_PreRunE_TxHashMultipleNetworksFails(t *testing.T) {
	t.Parallel()

	cmd := NewVerifyEnvCmdWithUse(Config{
		Logger:                 logger.Nop(),
		Domain:                 domain.NewDomain(t.TempDir(), "testdomain"),
		ContractInputsProvider: &mockContractInputsProvider{},
	}, "verify-env")
	cmd.SetArgs([]string{"-e", "staging", "-t", "0xabc", "-a", "0x123", "-n", "1,2"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "--tx-hash requires --networks to have only one chain selector", err.Error())
}

func TestParseNetworkFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect map[uint64]struct{}
	}{
		{"empty", "", nil},
		{"single", "123", map[uint64]struct{}{123: {}}},
		{"multiple", "1,2,3", map[uint64]struct{}{1: {}, 2: {}, 3: {}}},
		{"with_spaces", " 100 , 200 ", map[uint64]struct{}{100: {}, 200: {}}},
		{"invalid_skipped", "1,abc,3", map[uint64]struct{}{1: {}, 3: {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseNetworkFilter(tt.input)
			require.Equal(t, tt.expect, got)
		})
	}
}

func TestVerifyEnv_RunVerifyEnv_WithFromLocalFlag(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, opts DataStoreLoadOptions) (datastore.DataStore, error) {
				require.True(t, opts.FromLocal)
				return datastore.NewMemoryDataStore().Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging", fromLocal: true}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_FilterContract(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	ds := datastore.NewMemoryDataStore()
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xTargetAddr",
		Type:          "LinkToken",
		Version:       mustParseVersion(t),
	})
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xOtherAddr",
		Type:          "LinkToken",
		Version:       mustParseVersion(t),
	})

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{
		environment:    "staging",
		filterContract: "0xNonexistent",
	}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_AddressWithNilVersionSkipped(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	ds := datastore.NewMemoryDataStore()
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xNoVersion",
		Type:          "LinkToken",
		Version:       nil,
	})

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_GetInputsErrorSkipped(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	ds := datastore.NewMemoryDataStore()
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xUnsupported",
		Type:          "UnknownType",
		Version:       mustParseVersion(t),
	})

	provider := &mockContractInputsProvider{}
	provider.getInputsErr = errors.New("unsupported contract type")

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: provider,
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_VerificationSuccess(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	ds := datastore.NewMemoryDataStore()
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xSuccessAddr",
		Type:          "LinkToken",
		Version:       mustParseVersion(t),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "1",
			"message": "OK",
			"result":  `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{Transport: &evmRedirectTransport{target: targetURL}}

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		VerifierHTTPClient:     httpClient,
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging", pollInterval: 1}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_VerificationFailure(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	ds := datastore.NewMemoryDataStore()
	_ = ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethSelector,
		Address:       "0xFailAddr",
		Type:          "LinkToken",
		Version:       mustParseVersion(t),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{Transport: &evmRedirectTransport{target: targetURL}}

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		VerifierHTTPClient:     httpClient,
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeMainnet,
						ChainSelector: ethSelector,
						BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging", pollInterval: 1}, dom)
	require.NoError(t, err)
}

func TestVerifyEnv_RunVerifyEnv_SkippedNetworksInSummary(t *testing.T) {
	t.Parallel()

	dom := domain.NewDomain(t.TempDir(), "test")
	cmd := &cobra.Command{}

	// ZORA_TESTNET has chain ID 999999999 which returns StrategyUnknown
	zoraTestnetSelector := chainsel.ZORA_TESTNET.Selector
	ds := datastore.NewMemoryDataStore()

	cfg := Config{
		Logger:                 logger.Nop(),
		Domain:                 dom,
		ContractInputsProvider: &mockContractInputsProvider{},
		Deps: Deps{
			NetworkLoader: func(_ string, _ domain.Domain) (*cfgnet.Config, error) {
				return cfgnet.NewConfig([]cfgnet.Network{
					{
						Type:          cfgnet.NetworkTypeTestnet,
						ChainSelector: zoraTestnetSelector,
						BlockExplorer: cfgnet.BlockExplorer{},
						RPCs:          []cfgnet.RPC{{HTTPURL: "https://example.com"}},
					},
				}), nil
			},
			DataStoreLoader: func(_ context.Context, _ domain.EnvDir, _ logger.Logger, _ DataStoreLoadOptions) (datastore.DataStore, error) {
				return ds.Seal(), nil
			},
		},
	}
	cfg.deps()

	err := runVerifyEnv(cmd, cfg, verifyEnvFlags{environment: "staging"}, dom)
	require.NoError(t, err)
}

func mustParseVersion(t *testing.T) *semver.Version {
	t.Helper()
	v, err := semver.NewVersion("1.0.0")
	require.NoError(t, err)

	return v
}

// evmRedirectTransport redirects HTTP requests for verifier API tests.
type evmRedirectTransport struct {
	target *url.URL
}

func (r *evmRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host

	return http.DefaultTransport.RoundTrip(req)
}
