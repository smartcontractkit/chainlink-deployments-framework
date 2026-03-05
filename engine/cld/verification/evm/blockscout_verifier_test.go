package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestBlockscoutVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Blockscout API lives at /api; requests must target it when base URL has no path
		assert.Equal(t, "/api", r.URL.Path, "IsVerified and Verify must consistently use /api path")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "1",
			"result": `[{"type":"constructor"}]`,
		})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Name:    "Test",
			Version: "0.8.19",
			Sources: map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*blockscoutVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestBlockscoutVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "0",
			"result": "",
		})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Name:    "Test",
			Version: "0.8.19",
			Sources: map[string]any{},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*blockscoutVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestBlockscoutVerifier_ApiBase_PreservesConfiguredPath(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When apiURL already includes /api/v2, we must not clobber it
		assert.Equal(t, "/api/v2", r.URL.Path, "configured path must be preserved")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "result": "[]"})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL + "/api/v2"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Name:    "Test",
			Version: "0.8.19",
			Sources: map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*blockscoutVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestBlockscoutVerifier_String(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://explorer.zora.energy"}},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{},
		ContractType: "MyContract",
		Version:      "2.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)

	require.Equal(t, "MyContract 2.0.0 (0xabc on zora-mainnet)", v.String())
}
