package verification

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestRefsForChangeset(t *testing.T) {
	t.Parallel()

	ref1 := datastore.AddressRef{Address: "0x1", ChainSelector: 1, Type: "A", Version: semver.MustParse("1.0.0")}
	ref2 := datastore.AddressRef{Address: "0x2", ChainSelector: 2, Type: "B", Version: semver.MustParse("2.0.0")}

	provider := RefsForChangeset(map[string][]datastore.AddressRef{
		"cs-a":    {ref1},
		"cs-b":    {ref2},
		"cs-both": {ref1, ref2},
	})

	t.Run("returns refs for known key", func(t *testing.T) {
		t.Parallel()
		params := changeset.PreHookParams{ChangesetKey: "cs-a"}
		refs, err := provider(params)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		require.Equal(t, "0x1", refs[0].Address)
	})

	t.Run("returns empty for unknown key", func(t *testing.T) {
		t.Parallel()
		params := changeset.PreHookParams{ChangesetKey: "unknown"}
		refs, err := provider(params)
		require.NoError(t, err)
		require.Empty(t, refs)
	})

	t.Run("returns multiple refs", func(t *testing.T) {
		t.Parallel()
		params := changeset.PreHookParams{ChangesetKey: "cs-both"}
		refs, err := provider(params)
		require.NoError(t, err)
		require.Len(t, refs, 2)
	})
}

func TestRequireVerified_EmptyRefsSkips(t *testing.T) {
	t.Parallel()

	dom := fdomain.NewDomain(t.TempDir(), "test")
	provider := RefsForChangeset(map[string][]datastore.AddressRef{
		"my-cs": {}, // empty - should skip
	})
	hook := RequireVerified(dom, provider, &mockContractInputsProvider{})

	params := changeset.PreHookParams{
		Env:          changeset.HookEnv{Name: "staging", Logger: logger.Nop()},
		ChangesetKey: "my-cs",
	}
	err := hook.Func(t.Context(), params)
	require.NoError(t, err)
}

func TestRequireVerified_RefsProviderError(t *testing.T) {
	t.Parallel()

	dom := fdomain.NewDomain(t.TempDir(), "test")
	provider := func(changeset.PreHookParams) ([]datastore.AddressRef, error) {
		return nil, errors.New("refs lookup failed")
	}
	hook := RequireVerified(dom, provider, &mockContractInputsProvider{})

	params := changeset.PreHookParams{
		Env:          changeset.HookEnv{Name: "staging", Logger: logger.Nop()},
		ChangesetKey: "my-cs",
	}
	err := hook.Func(t.Context(), params)
	require.Error(t, err)
	require.Contains(t, err.Error(), "require-verified: get refs")
	require.Contains(t, err.Error(), "refs lookup failed")
}

func TestRequireVerified_HookDefinition(t *testing.T) {
	t.Parallel()

	dom := fdomain.NewDomain(t.TempDir(), "test")
	hook := RequireVerified(dom, RefsForChangeset(map[string][]datastore.AddressRef{}), &mockContractInputsProvider{})

	require.Equal(t, "require-verified", hook.Name)
	require.Equal(t, changeset.Abort, hook.FailurePolicy)
	require.NotZero(t, hook.Timeout)
}

func TestRequireVerified_FullFlow_AllVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "1",
			"message": "OK",
			"result":  `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{Transport: &redirectTransport{target: targetURL}}

	dom := setupDomainWithNetworks(t)
	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0xVerified", ChainSelector: ethSelector, Type: "LinkToken", Version: semver.MustParse("1.0.0")},
	}
	provider := RefsForChangeset(map[string][]datastore.AddressRef{"my-cs": refs})
	hook := RequireVerified(dom, provider, &mockContractInputsProvider{}, WithHTTPClient(httpClient))

	params := changeset.PreHookParams{
		Env:          changeset.HookEnv{Name: "staging", Logger: logger.Nop()},
		ChangesetKey: "my-cs",
	}
	err := hook.Func(t.Context(), params)
	require.NoError(t, err)
}

func TestRequireVerified_FullFlow_UnverifiedFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "1",
			"message": "OK",
			"result":  "Contract source code not verified",
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{Transport: &redirectTransport{target: targetURL}}

	dom := setupDomainWithNetworks(t)
	ethSelector := chainsel.ETHEREUM_MAINNET.Selector
	refs := []datastore.AddressRef{
		{Address: "0xUnverified", ChainSelector: ethSelector, Type: "LinkToken", Version: semver.MustParse("1.0.0")},
	}
	provider := RefsForChangeset(map[string][]datastore.AddressRef{"my-cs": refs})
	hook := RequireVerified(dom, provider, &mockContractInputsProvider{}, WithHTTPClient(httpClient))

	params := changeset.PreHookParams{
		Env:          changeset.HookEnv{Name: "staging", Logger: logger.Nop()},
		ChangesetKey: "my-cs",
	}
	err := hook.Func(t.Context(), params)
	require.Error(t, err)
	require.Contains(t, err.Error(), "require-verified:")
	require.Contains(t, err.Error(), "not verified")
}

// setupDomainWithNetworks creates a domain with .config/domain.yaml and .config/networks/*.yaml
// so config.LoadNetworks works.
func setupDomainWithNetworks(t *testing.T) fdomain.Domain {
	t.Helper()

	rootDir := t.TempDir()
	domainKey := "test-domain"
	domainDir := filepath.Join(rootDir, domainKey)
	configDir := filepath.Join(domainDir, ".config")
	networksDir := filepath.Join(configDir, "networks")

	require.NoError(t, os.MkdirAll(networksDir, 0700))

	domainYAML := `environments:
  staging:
    network_types:
      - mainnet
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "domain.yaml"), []byte(domainYAML), 0600))

	// Ethereum mainnet with block explorer - etherscan needs API key for NewVerifier
	networksYAML := `networks:
  - type: mainnet
    chain_selector: 5009297550715157269
    block_explorer:
      type: Etherscan
      api_key: test-key
      url: https://etherscan.io
    rpcs:
      - rpc_name: mainnet-rpc
        preferred_url_scheme: http
        http_url: https://eth.llamarpc.com
        ws_url: wss://eth.llamarpc.com
`
	require.NoError(t, os.WriteFile(filepath.Join(networksDir, "networks-mainnet.yaml"), []byte(networksYAML), 0600))

	return fdomain.NewDomain(rootDir, domainKey)
}

// redirectTransport redirects HTTP requests to a target URL for testing.
type redirectTransport struct {
	target *url.URL
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host

	return http.DefaultTransport.RoundTrip(req)
}

// mockContractInputsProvider is a test double for evm.ContractInputsProvider.
type mockContractInputsProvider struct{}

func (m *mockContractInputsProvider) GetInputs(_ datastore.ContractType, _ *semver.Version) (evm.SolidityContractMetadata, error) {
	return evm.SolidityContractMetadata{
		Name:    "Test",
		Version: "0.8.19",
		Sources: map[string]any{"test.sol": map[string]any{"content": "contract Test {}"}},
	}, nil
}
