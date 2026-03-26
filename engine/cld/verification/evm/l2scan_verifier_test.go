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

func TestL2ScanVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(l2scanAPIResponse{
			Status:  "1",
			Message: "OK",
			Result:  `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	client := server.Client()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MERLIN_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyL2Scan, VerifierConfig{
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

	verified, err := v.(*l2scanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestL2ScanVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(l2scanAPIResponse{
			Status:  "0",
			Message: "Contract source code not verified",
			Result:  "",
		})
	}))
	defer server.Close()

	client := server.Client()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MERLIN_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyL2Scan, VerifierConfig{
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

	verified, err := v.(*l2scanVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}
