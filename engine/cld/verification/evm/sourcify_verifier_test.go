package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func newTestSourcifyChain() chainsel.Chain {
	return chainsel.Chain{
		EvmChainID: 295,
		Selector:   chainsel.HEDERA_MAINNET.Selector,
		Name:       "hedera-mainnet",
	}
}

func TestSourcifyVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	match := "exact_match"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/contract/295/0xabc")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcifyContractResponse{Match: &match})
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0xabc",
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

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestSourcifyVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestSourcifyVerifier_IsVerified_NullMatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcifyContractResponse{Match: nil})
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestSourcifyVerifier_Verify_AlreadyVerified(t *testing.T) {
	t.Parallel()

	match := "match"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcifyContractResponse{Match: &match})
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0xabc",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Sources:  map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)
}

func TestSourcifyVerifier_Verify_SubmitAndPoll(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	match := "exact_match"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && callNum == 1:
			// IsVerified check -> not found
			w.WriteHeader(http.StatusNotFound)

		case r.Method == http.MethodPost:
			// Submit verification
			var body sourcifyVerifyRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "0.8.19", body.CompilerVersion)
			assert.Equal(t, "contracts/Test.sol:Test", body.ContractIdentifier)
			assert.NotNil(t, body.StdJsonInput)

			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(sourcifyVerifyResponse{VerificationID: "test-uuid-123"})

		case r.Method == http.MethodGet && callNum == 3:
			// First poll -> still processing
			_ = json.NewEncoder(w).Encode(sourcifyJobStatus{
				IsJobCompleted: false,
				VerificationID: "test-uuid-123",
			})

		case r.Method == http.MethodGet && callNum == 4:
			// Second poll -> done
			_ = json.NewEncoder(w).Encode(sourcifyJobStatus{
				IsJobCompleted: true,
				VerificationID: "test-uuid-123",
				Contract:       &sourcifyContractResponse{Match: &match},
			})
		}
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0xabc",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Sources:  map[string]any{"contracts/Test.sol": map[string]any{"content": "contract Test {}"}},
			Settings: map[string]any{"optimizer": map[string]any{"enabled": false}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: 1, // 1ns for fast tests
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)
}

func TestSourcifyVerifier_Verify_Conflict(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")

		if callNum == 1 {
			// IsVerified -> not found
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Submit -> 409 already verified
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"customCode": "already_verified",
			"message":    "Already verified",
		})
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0xabc",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Sources:  map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)
}

func TestSourcifyVerifier_Verify_Failed(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case callNum == 1:
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(sourcifyVerifyResponse{VerificationID: "fail-uuid"})
		default:
			_ = json.NewEncoder(w).Encode(sourcifyJobStatus{
				IsJobCompleted: true,
				VerificationID: "fail-uuid",
				Error: &sourcifyJobError{
					CustomCode: "no_match",
					Message:    "The onchain and recompiled bytecodes don't match.",
				},
			})
		}
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0xabc",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Sources:  map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: 1,
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no_match")
}

func TestSourcifyVerifier_String(t *testing.T) {
	t.Parallel()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{},
		ContractType: "MyContract",
		Version:      "2.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.Equal(t, "MyContract 2.0.0 (0xabc on hedera-mainnet)", v.String())
}

func TestSourcifyVerifier_BuildContractIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata SolidityContractMetadata
		expected string
	}{
		{
			name: "exact path match",
			metadata: SolidityContractMetadata{
				Name:    "Storage",
				Sources: map[string]any{"contracts/Storage.sol": map[string]any{"content": "..."}},
			},
			expected: "contracts/Storage.sol:Storage",
		},
		{
			name: "fallback to contract type when name empty",
			metadata: SolidityContractMetadata{
				Sources: map[string]any{"src/Foo.sol": map[string]any{"content": "..."}},
			},
			expected: "src/Foo.sol:Test",
		},
		{
			name: "name only when no sources",
			metadata: SolidityContractMetadata{
				Name:    "Standalone",
				Sources: map[string]any{},
			},
			expected: "Standalone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain := newTestSourcifyChain()
			sv := &sourcifyVerifier{
				chain:        chain,
				metadata:     tt.metadata,
				contractType: "Test",
			}
			got := sv.buildContractIdentifier()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestSourcifyVerifier_CustomBaseURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/contract/295/0xabc")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{Name: "Test", Version: "0.8.19"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestSourcifyVerifier_DefaultBaseURL(t *testing.T) {
	t.Parallel()

	chain := newTestSourcifyChain()
	v, err := newSourcifyVerifier(VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0xabc",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.Equal(t, sourcifyServerURL, v.(*sourcifyVerifier).baseURL)
}
