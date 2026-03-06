package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestEtherscanVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewEtherscanV2ContractVerifier(
		chain, "api-key", "0x123",
		SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		"Test", "1.0.0", 0, logger.Nop(),
		client,
	)
	require.NoError(t, err)

	verified, err := v.(*etherscanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestEtherscanVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(etherscanAPIResponse[string]{
			Status:  statusOK,
			Message: messageOK,
			Result:  "Contract source code not verified",
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewEtherscanV2ContractVerifier(
		chain, "api-key", "0x123",
		SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		"Test", "1.0.0", 0, logger.Nop(),
		client,
	)
	require.NoError(t, err)

	verified, err := v.(*etherscanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestEtherscanVerifier_String(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewEtherscanV2ContractVerifier(
		chain, "key", "0xabc",
		SolidityContractMetadata{},
		"MyContract", "2.0.0", 0, logger.Nop(),
		nil,
	)
	require.NoError(t, err)

	require.Equal(t, "MyContract 2.0.0 (0xabc on ethereum-mainnet)", v.String())
}

// redirectTransport redirects all requests to the target URL for testing.
type redirectTransport struct {
	target *url.URL
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host

	return http.DefaultTransport.RoundTrip(req)
}
