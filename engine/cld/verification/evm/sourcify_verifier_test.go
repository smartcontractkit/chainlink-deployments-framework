package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestSourcifyVerifier_IsVerified_AlreadyVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sourcifyAPIResponse{Status: "full"})
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.True(t, verified)
}

func TestSourcifyVerifier_IsVerified_NotVerified(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Files have not been found"))
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	verified, err := v.(*sourcifyVerifier).IsVerified(context.Background())
	require.NoError(t, err)
	require.False(t, verified)
}

func TestSourcifyVerifier_Verify_Success(t *testing.T) {
	t.Parallel()

	var pollCount atomic.Int32
	matchStr := "exact_match"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/any/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Files have not been found"))

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v2/verify/"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.NotNil(t, body["stdJsonInput"])
			assert.Equal(t, "0.8.19+commit.abc", body["compilerVersion"])
			assert.Equal(t, "contracts/Test.sol:Test", body["contractIdentifier"])

			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(sourcifyV2SubmitResponse{
				VerificationID: "test-verification-id",
			})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2/verify/"):
			count := pollCount.Add(1)
			if count < 2 {
				_ = json.NewEncoder(w).Encode(sourcifyV2JobResponse{
					IsJobCompleted: false,
					VerificationID: "test-verification-id",
				})
			} else {
				_ = json.NewEncoder(w).Encode(sourcifyV2JobResponse{
					IsJobCompleted: true,
					VerificationID: "test-verification-id",
					Contract: &sourcifyV2Match{
						Match:   &matchStr,
						ChainID: "295",
						Address: "0x123",
					},
				})
			}
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Version:  "0.8.19+commit.abc",
			Language: "Solidity",
			Settings: map[string]any{"optimizer": map[string]any{"enabled": true}},
			Sources:  map[string]any{"contracts/Test.sol": map[string]any{"content": "contract Test {}"}},
			Name:     "contracts/Test.sol:Test",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: 10 * time.Millisecond,
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, pollCount.Load(), int32(2))
}

func TestSourcifyVerifier_Verify_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/any/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Files have not been found"))

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v2/verify/"):
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(sourcifyV2SubmitResponse{
				VerificationID: "test-verification-id",
			})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2/verify/"):
			_ = json.NewEncoder(w).Encode(sourcifyV2JobResponse{
				IsJobCompleted: true,
				VerificationID: "test-verification-id",
				Error: &sourcifyV2Error{
					CustomCode: "no_match",
					Message:    "The onchain and recompiled bytecodes don't match.",
				},
			})
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Version:  "0.8.19+commit.abc",
			Language: "Solidity",
			Sources:  map[string]any{"contracts/Test.sol": map[string]any{"content": "contract Test {}"}},
			Name:     "contracts/Test.sol:Test",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: 10 * time.Millisecond,
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no_match")
}

func TestSourcifyVerifier_Verify_ContractIdentifierWithoutPath(t *testing.T) {
	t.Parallel()

	var capturedIdentifier string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/any/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Files have not been found"))

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v2/verify/"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedIdentifier, _ = body["contractIdentifier"].(string)

			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(sourcifyV2SubmitResponse{
				VerificationID: "test-id",
			})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2/verify/"):
			matchStr := "exact_match"
			_ = json.NewEncoder(w).Encode(sourcifyV2JobResponse{
				IsJobCompleted: true,
				VerificationID: "test-id",
				Contract:       &sourcifyV2Match{Match: &matchStr},
			})
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Version:  "0.8.19+commit.abc",
			Language: "Solidity",
			Sources:  map[string]any{"contracts/MyContract.sol": map[string]any{"content": "contract MyContract {}"}},
			Name:     "MyContract",
		},
		ContractType: "MyContract",
		Version:      "1.0.0",
		PollInterval: 10 * time.Millisecond,
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)
	require.Contains(t, capturedIdentifier, "MyContract")
	require.Contains(t, capturedIdentifier, ":")
}

func TestSourcifyVerifier_Verify_V1Fallback(t *testing.T) {
	t.Parallel()

	var capturedV1Body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/any/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Files have not been found"))

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v2/verify/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<pre>Cannot POST /v2/verify/295/0x123</pre>`))

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/verify/solc-json"):
			_ = json.NewDecoder(r.Body).Decode(&capturedV1Body)
			_ = json.NewEncoder(w).Encode(sourcifyV1VerifyResponse{
				Result: []sourcifyV1Result{{Status: "perfect", Address: "0x123", ChainID: "295"}},
			})
		}
	}))
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	client := &http.Client{Transport: &redirectTransport{target: targetURL}}

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Version:  "0.8.19+commit.abc",
			Language: "Solidity",
			Settings: map[string]any{"optimizer": map[string]any{"enabled": true}},
			Sources:  map[string]any{"contracts/Test.sol": map[string]any{"content": "contract Test {}"}},
			Name:     "contracts/Test.sol:Test",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: 10 * time.Millisecond,
		Logger:       logger.Nop(),
		HTTPClient:   client,
	})
	require.NoError(t, err)

	err = v.Verify(context.Background())
	require.NoError(t, err)

	require.NotNil(t, capturedV1Body)
	assert.Equal(t, "0x123", capturedV1Body["address"])
	assert.Equal(t, "Test", capturedV1Body["contractName"])
	files, ok := capturedV1Body["files"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, files, "SolcJsonInput.json")
}
