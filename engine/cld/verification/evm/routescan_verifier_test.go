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

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestRoutescanVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(routeScanAPIResponse[string]{
			Status:  statusOK,
			Message: messageOK,
			Result:  `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.AVALANCHE_TESTNET_FUJI.Selector)
	require.True(t, ok)

	v, err := newRouteScanVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{Type: cfgnet.NetworkTypeTestnet, ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*routescanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestRoutescanVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(routeScanAPIResponse[string]{
			Status:  "0",
			Message: "NOTOK",
			Result:  "Contract source code not verified",
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.AVALANCHE_TESTNET_FUJI.Selector)
	require.True(t, ok)

	v, err := newRouteScanVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{Type: cfgnet.NetworkTypeTestnet, ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*routescanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestRoutescanVerifier_String(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.AVALANCHE_TESTNET_FUJI.Selector)
	require.True(t, ok)

	v, err := newRouteScanVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{Type: cfgnet.NetworkTypeTestnet, ChainSelector: chain.Selector},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{},
		ContractType: "MyContract",
		Version:      "2.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)

	require.Equal(t, "MyContract 2.0.0 (0xabc on avalanche-testnet-fuji)", v.String())
}
