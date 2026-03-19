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

func TestOkLinkVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(okLinkResponse[oklinkVerifyContractInfo]{
			Code: oklinkStatusOK,
			Msg:  oklinkMessageOK,
			Data: []oklinkVerifyContractInfo{{SourceCode: "contract", ContractName: "Test", CompilerVersion: "0.8.19"}},
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET_XLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyOkLink, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test-key"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*oklinkVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestOkLinkVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(okLinkResponse[oklinkVerifyContractInfo]{
			Code: oklinkStatusOK,
			Msg:  oklinkMessageOK,
			Data: []oklinkVerifyContractInfo{},
		})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET_XLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyOkLink, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test-key"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*oklinkVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}
