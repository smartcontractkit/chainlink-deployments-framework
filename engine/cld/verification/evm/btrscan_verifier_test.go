package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestBtrScanVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(btrScanSourceCodeResponse{
			Status:  "1",
			Message: "OK",
			Result:  []btrScanSourceCodeResult{{SourceCode: "contract source here"}},
		})
	}))
	defer server.Close()

	client := server.Client()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MAINNET_BITLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL, APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*btrscanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestBtrScanVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(btrScanSourceCodeResponse{
			Status:  "1",
			Message: "OK",
			Result:  []btrScanSourceCodeResult{{SourceCode: ""}},
		})
	}))
	defer server.Close()

	client := server.Client()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MAINNET_BITLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL, APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*btrscanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}
