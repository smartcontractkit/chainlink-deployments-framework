package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestBlockscoutVerifier_Verify_HTTP200WithStatusZero_ReturnsError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			// One creation tx; bytecode prefix mismatch → empty constructor args
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "0", "message": "NOTOK", "result": "Compilation error",
			})
		default:
			http.Error(w, "unexpected request "+r.Method+" action="+action, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "blockscout verifysourcecode rejected")
}

func TestBlockscoutVerifier_Verify_StatusOne_Succeeds(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "message": "OK", "result": "abc-guid"})
		case r.Method == http.MethodGet && action == "checkverifystatus":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "message": "OK", "result": "Pass - Verified"})
		default:
			http.Error(w, "unexpected request "+r.Method+" action="+action, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "Test",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"Test.sol": map[string]any{"content": "contract Test {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		PollInterval: time.Nanosecond,
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.NoError(t, err)
}

func TestTruncateForLog(t *testing.T) {
	t.Parallel()

	require.Equal(t, "short", truncateForLog("short", 100))
	require.Empty(t, truncateForLog("", 0))
	long := strings.Repeat("x", 100)
	out := truncateForLog(long, 10)
	require.True(t, strings.HasPrefix(out, strings.Repeat("x", 10)))
	require.Contains(t, out, "truncated")
}

func TestBlockscoutVerifier_Verify_AlreadyVerified_ShortCircuits(t *testing.T) {
	t.Parallel()

	var txlistCalls, postCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "result": `[{"type":"function"}]`})
		case r.Method == http.MethodGet && action == "txlist":
			txlistCalls++
			http.Error(w, "should not call txlist", http.StatusTeapot)
		case r.Method == http.MethodPost && action == "verifysourcecode":
			postCalls++
			http.Error(w, "should not submit", http.StatusTeapot)
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "X",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"X.sol": map[string]any{"content": "contract X {}"}},
		},
		ContractType: "X",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	require.NoError(t, v.(*blockscoutVerifier).Verify(context.Background()))
	require.Equal(t, 0, txlistCalls)
	require.Equal(t, 0, postCalls)
}

func TestBlockscoutVerifier_Verify_FailCheckVerifyStatus_MessageOK(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "message": "OK", "result": "job-guid"})
		case r.Method == http.MethodGet && action == "checkverifystatus":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "1", "message": "OK", "result": "Fail - Unable to verify",
			})
		default:
			http.Error(w, "unexpected "+r.Method+" "+action, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "contracts/Foo.sol:Foo",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"Foo.sol": map[string]any{"content": "contract Foo {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "Foo",
		Version:      "1.0.0",
		PollInterval: time.Nanosecond,
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "blockscout verification failed")
	require.Contains(t, err.Error(), "explorer uses status=1/message=OK")
	require.Contains(t, err.Error(), `contractname="contracts/Foo.sol:Foo"`)
}

func TestBlockscoutVerifier_Verify_PendingThenTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "message": "OK", "result": "g"})
		case r.Method == http.MethodGet && action == "checkverifystatus":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "1", "message": "OK", "result": "Pending in queue"})
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "T",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"T.sol": map[string]any{"content": "contract T {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "T",
		Version:      "1.0.0",
		PollInterval: time.Nanosecond,
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out after")
}

func TestBlockscoutVerifier_getConstructorArgs_MatchingBytecode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "txlist", r.URL.Query().Get("action"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "1", "message": "OK",
			"result": []map[string]string{{"input": "0xaabbccddeeff"}},
		})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Bytecode: "0xaabbcc",
		},
		Logger:     logger.Nop(),
		HTTPClient: server.Client(),
	})
	require.NoError(t, err)

	args, err := v.(*blockscoutVerifier).getConstructorArgs(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ddeeff", args)
}

func TestBlockscoutVerifier_getConstructorArgs_TxlistNotOK(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// result must decode as []transactionInfo; empty array keeps JSON shape valid.
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "0", "message": "NOTOK", "result": []any{}})
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:      chain,
		Network:    cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address:    "0x1234567890123456789012345678901234567890",
		Metadata:   SolidityContractMetadata{},
		Logger:     logger.Nop(),
		HTTPClient: server.Client(),
	})
	require.NoError(t, err)

	_, err = v.(*blockscoutVerifier).getConstructorArgs(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "txlist API error")
}

func TestBlockscoutVerifier_Verify_SubmitDecodeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("not-json{"))
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name: "N", Version: "0.8.19", Language: "Solidity",
			Settings: map[string]any{}, Sources: map[string]any{"N.sol": map[string]any{"content": "contract N {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "N",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode verifysourcecode response")
}

func TestBlockscoutVerifier_Verify_SubmitHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			http.Error(w, "server error", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "N",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"N.sol": map[string]any{"content": "contract N {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "N",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	err = v.(*blockscoutVerifier).Verify(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "http error")
}

func TestBlockscoutVerifier_Verify_UsesContractTypeWhenMetadataNameEmpty(t *testing.T) {
	t.Parallel()

	var sawContractName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "getabi":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "result": ""})
		case r.Method == http.MethodGet && action == "txlist":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "1", "message": "OK",
				"result": []map[string]string{{"input": "0xbbbb"}},
			})
		case r.Method == http.MethodPost && action == "verifysourcecode":
			assert.NoError(t, r.ParseForm())
			sawContractName = r.FormValue("contractname")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0", "message": "bad", "result": "x"})
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	chain, ok := chainsel.ChainBySelector(chainsel.ZORA_MAINNET.Selector)
	require.True(t, ok)

	v, err := newBlockscoutVerifier(VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: server.URL}},
		Address: "0x1234567890123456789012345678901234567890",
		Metadata: SolidityContractMetadata{
			Name:     "",
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{},
			Sources:  map[string]any{"P.sol": map[string]any{"content": "contract P {}"}},
			Bytecode: "0xaaaa",
		},
		ContractType: "PoolType",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
		HTTPClient:   server.Client(),
	})
	require.NoError(t, err)

	_ = v.(*blockscoutVerifier).Verify(context.Background())
	require.Equal(t, "PoolType", sawContractName)
}
