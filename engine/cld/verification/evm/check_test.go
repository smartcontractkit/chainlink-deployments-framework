package evm

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestCheckVerified_ConfigValidation(t *testing.T) {
	t.Parallel()

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0x123", ChainSelector: ethSelector, Type: "Test", Version: semver.MustParse("1.0.0")},
	}
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: ethSelector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
			RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
		},
	})

	t.Run("nil ContractInputsProvider", func(t *testing.T) {
		t.Parallel()
		unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
			ContractInputsProvider: nil,
			NetworkConfig:          networkCfg,
			Logger:                 logger.Nop(),
		})
		require.Error(t, err)
		require.Nil(t, unverified)
		require.Contains(t, err.Error(), "ContractInputsProvider is required")
	})

	t.Run("nil NetworkConfig", func(t *testing.T) {
		t.Parallel()
		unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
			ContractInputsProvider: &mockContractInputsProvider{},
			NetworkConfig:          nil,
			Logger:                 logger.Nop(),
		})
		require.Error(t, err)
		require.Nil(t, unverified)
		require.Contains(t, err.Error(), "NetworkConfig is required")
	})
}

func TestCheckVerified_NilVersion(t *testing.T) {
	t.Parallel()

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0x123", ChainSelector: ethSelector, Type: "Test", Version: nil},
	}
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: ethSelector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
			RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
		},
	})

	unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
		ContractInputsProvider: &mockContractInputsProvider{},
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
	})
	require.Error(t, err)
	require.Nil(t, unverified)
	require.Contains(t, err.Error(), "version is required")
}

func TestCheckVerified_NetworkNotFound(t *testing.T) {
	t.Parallel()

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0x123", ChainSelector: ethSelector, Type: "Test", Version: semver.MustParse("1.0.0")},
	}
	// Empty network config - eth mainnet not in config
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{})

	unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
		ContractInputsProvider: &mockContractInputsProvider{},
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
	})
	require.Error(t, err)
	require.Nil(t, unverified)
	require.Contains(t, err.Error(), "not found in configuration")
}

func TestCheckVerified_MetadataProviderError(t *testing.T) {
	t.Parallel()

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0x123", ChainSelector: ethSelector, Type: "Test", Version: semver.MustParse("1.0.0")},
	}
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: ethSelector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
			RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
		},
	})

	provider := &mockContractInputsProvider{}
	provider.getInputsErr = errors.New("metadata unavailable")

	unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
		ContractInputsProvider: provider,
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
	})
	require.Error(t, err)
	require.Nil(t, unverified)
	require.Contains(t, err.Error(), "metadata unavailable")
}

func TestCheckVerified_AllVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(etherscanAPIResponse[string]{
			Status:  statusOK,
			Message: messageOK,
			Result:  `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0x123", ChainSelector: ethSelector, Type: "LinkToken", Version: semver.MustParse("1.0.0")},
	}
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: ethSelector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
			RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
		},
	})

	unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
		ContractInputsProvider: &mockContractInputsProvider{},
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
		HTTPClient:             client,
	})
	require.NoError(t, err)
	require.Empty(t, unverified)
}

func TestCheckVerified_SomeUnverified(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// First contract verified, second not
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(etherscanAPIResponse[string]{
				Status:  statusOK,
				Message: messageOK,
				Result:  `[{"type":"constructor"}]`,
			})
		} else {
			_ = json.NewEncoder(w).Encode(etherscanAPIResponse[string]{
				Status:  statusOK,
				Message: messageOK,
				Result:  "Contract source code not verified",
			})
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0xVerified", ChainSelector: ethSelector, Type: "LinkToken", Version: semver.MustParse("1.0.0")},
		{Address: "0xUnverified", ChainSelector: ethSelector, Type: "LinkToken", Version: semver.MustParse("1.0.0")},
	}
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: ethSelector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"},
			RPCs:          []cfgnet.RPC{{HTTPURL: "https://eth.llamarpc.com"}},
		},
	})

	unverified, err := CheckVerified(t.Context(), refs, CheckConfig{
		ContractInputsProvider: &mockContractInputsProvider{},
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
		HTTPClient:             client,
	})
	require.NoError(t, err)
	require.Len(t, unverified, 1)
	require.Equal(t, "0xUnverified", unverified[0].Address)
}

func TestCheckVerified_EmptyRefs(t *testing.T) {
	t.Parallel()

	networkCfg := cfgnet.NewConfig([]cfgnet.Network{})
	unverified, err := CheckVerified(t.Context(), nil, CheckConfig{
		ContractInputsProvider: &mockContractInputsProvider{},
		NetworkConfig:          networkCfg,
		Logger:                 logger.Nop(),
	})
	require.NoError(t, err)
	require.Empty(t, unverified)
}

// mockContractInputsProvider is a test double for ContractInputsProvider.
type mockContractInputsProvider struct {
	getInputsErr error
}

func (m *mockContractInputsProvider) GetInputs(_ datastore.ContractType, _ *semver.Version) (SolidityContractMetadata, error) {
	if m.getInputsErr != nil {
		return SolidityContractMetadata{}, m.getInputsErr
	}

	return SolidityContractMetadata{
		Name:    "Test",
		Version: "0.8.19",
		Sources: map[string]any{"test.sol": map[string]any{"content": "contract Test {}"}},
	}, nil
}
