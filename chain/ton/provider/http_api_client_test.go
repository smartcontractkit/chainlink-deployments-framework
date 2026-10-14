package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	tonlib "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func testAddr(b byte) *address.Address {
	return address.NewAddress(0, 0, bytes.Repeat([]byte{b}, 32))
}

func rawEntry(t *testing.T, typ string, value any) []json.RawMessage {
	t.Helper()

	rt, err := json.Marshal(typ)
	require.NoError(t, err)
	rv, err := json.Marshal(value)
	require.NoError(t, err)

	return []json.RawMessage{rt, rv}
}

func Test_isHTTPAPIURL(t *testing.T) {
	t.Parallel()

	require.True(t, isHTTPAPIURL("https://toncenter.com/api/v2"))
	require.True(t, isHTTPAPIURL("http://localhost:8081"))
	require.False(t, isHTTPAPIURL("liteserver://key@127.0.0.1:1234"))
	require.False(t, isHTTPAPIURL(""))
}

func Test_validateNodeURL(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateNodeURL("https://toncenter.com/api/v2"))
	require.NoError(t, validateNodeURL("liteserver://key@127.0.0.1:1234"))
	require.ErrorContains(t, validateNodeURL(""), "liteserver url is required")
	require.ErrorContains(t, validateNodeURL("ws://host"), "expected liteserver:// prefix")
}

func Test_decodeStackEntry(t *testing.T) {
	t.Parallel()

	c := cell.BeginCell().MustStoreUInt(0xCAFE, 16).EndCell()
	addr := testAddr(0x11)
	addrCell := cell.BeginCell().MustStoreAddr(addr).EndCell()

	t.Run("num", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "num", "0x1f"))
		require.NoError(t, err)
		require.Equal(t, big.NewInt(31), v)

		v, err = decodeStackEntry(rawEntry(t, "num", "-0x1"))
		require.NoError(t, err)
		require.Equal(t, big.NewInt(-1), v)
	})

	t.Run("cell object", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "cell", map[string]any{
			"bytes": base64.StdEncoding.EncodeToString(c.ToBOC()),
		}))
		require.NoError(t, err)
		require.Equal(t, c.Hash(), v.(*cell.Cell).Hash())
	})

	t.Run("slice object", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "slice", map[string]any{
			"bytes": base64.StdEncoding.EncodeToString(addrCell.ToBOC()),
		}))
		require.NoError(t, err)
		got, err := v.(*cell.Slice).LoadAddr()
		require.NoError(t, err)
		require.True(t, addr.Equals(got))
	})

	t.Run("address serialized as cell becomes slice", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "cell", map[string]any{
			"bytes": base64.StdEncoding.EncodeToString(addrCell.ToBOC()),
		}))
		require.NoError(t, err)
		got, err := v.(*cell.Slice).LoadAddr()
		require.NoError(t, err)
		require.True(t, addr.Equals(got))

		// addr_none (2 bits) too
		none := cell.BeginCell().MustStoreUInt(0, 2).EndCell()
		v, err = decodeStackEntry(rawEntry(t, "cell", map[string]any{
			"bytes": base64.StdEncoding.EncodeToString(none.ToBOC()),
		}))
		require.NoError(t, err)
		require.IsType(t, &cell.Slice{}, v)
	})

	t.Run("dictionary-like cells stay cells", func(t *testing.T) {
		t.Parallel()
		withRef := cell.BeginCell().MustStoreUInt(0, 8).MustStoreRef(c).EndCell()
		v, err := decodeStackEntry(rawEntry(t, "cell", map[string]any{
			"bytes": base64.StdEncoding.EncodeToString(withRef.ToBOC()),
		}))
		require.NoError(t, err)
		require.IsType(t, &cell.Cell{}, v)
	})

	t.Run("null", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "null", nil))
		require.NoError(t, err)
		require.Nil(t, v)
	})

	t.Run("empty list is TVM null", func(t *testing.T) {
		t.Parallel()
		v, err := decodeStackEntry(rawEntry(t, "list", map[string]any{"@type": "tvm.list", "elements": []any{}}))
		require.NoError(t, err)
		require.Nil(t, v)

		_, err = decodeStackEntry(rawEntry(t, "list", map[string]any{"@type": "tvm.list", "elements": []any{map[string]any{}}}))
		require.ErrorContains(t, err, "non-empty list")
	})

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		_, err := decodeStackEntry(rawEntry(t, "tuple", map[string]any{}))
		require.ErrorContains(t, err, "unsupported stack entry type")
	})
}

func Test_encodeStackParam(t *testing.T) {
	t.Parallel()

	got, err := encodeStackParam(big.NewInt(255))
	require.NoError(t, err)
	require.Equal(t, []string{"num", "0xff"}, got)

	got, err = encodeStackParam(uint64(7))
	require.NoError(t, err)
	require.Equal(t, []string{"num", "0x7"}, got)

	got, err = encodeStackParam(big.NewInt(-2))
	require.NoError(t, err)
	require.Equal(t, []string{"num", "-0x2"}, got)

	_, err = encodeStackParam("nope")
	require.ErrorContains(t, err, "unsupported stack param type")
}

func Test_httpAPIClient_RunGetMethod(t *testing.T) {
	t.Parallel()

	var gotReq runGetMethodRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2root/runGetMethod", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotReq))
		_, _ = w.Write([]byte(`{"ok":true,"result":{"gas_used":1,"exit_code":0,"stack":[["num","0x2a"],["null",null]]}}`))
	}))
	defer srv.Close()

	// the URL is the v2 root as given; a trailing slash is tolerated
	for _, base := range []string{srv.URL + "/v2root", srv.URL + "/v2root/"} {
		c := newHTTPAPIClient(base, "")
		addr := testAddr(0x01)

		res, err := c.WaitForBlock(1).RunGetMethod(t.Context(), nil, addr, "getOpCount", big.NewInt(5))
		require.NoError(t, err)

		n, err := res.Int(0)
		require.NoError(t, err)
		require.Equal(t, int64(42), n.Int64())
		isNil, err := res.IsNil(1)
		require.NoError(t, err)
		require.True(t, isNil)

		require.Equal(t, "getOpCount", gotReq.Method)
		require.Equal(t, [][]string{{"num", "0x5"}}, gotReq.Stack)
	}
}

func Test_httpAPIClient_RunGetMethod_ExitCode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"gas_used":1,"exit_code":9,"stack":[]}}`))
	}))
	defer srv.Close()

	c := newHTTPAPIClient(srv.URL, "")
	addr := testAddr(0x01)

	_, err := c.RunGetMethod(t.Context(), nil, addr, "boom")
	require.Error(t, err)
	require.Contains(t, err.Error(), "9")
}

func Test_httpAPIClient_GetMasterchainInfo(t *testing.T) {
	t.Parallel()

	root := base64.StdEncoding.EncodeToString(make([]byte, 32))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v2/getMasterchainInfo", r.URL.Path)
		_, _ = w.Write([]byte(`{"ok":true,"result":{"last":{"workchain":-1,"shard":"-9223372036854775808","seqno":123,` +
			`"root_hash":"` + root + `","file_hash":"` + root + `"},"state_root_hash":"` + root + `","init":{"workchain":-1,"shard":"-9223372036854775808","seqno":0,"root_hash":"` + root + `","file_hash":"` + root + `"}}}`))
	}))
	defer srv.Close()

	c := newHTTPAPIClient(srv.URL+"/api/v2", "")
	blk, err := c.CurrentMasterchainInfo(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint32(123), blk.SeqNo)
	require.Equal(t, int32(-1), blk.Workchain)
	require.Equal(t, int64(-9223372036854775808), blk.Shard)
}

func Test_httpAPIClient_RunGetMethod_UninitializedAccount(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"gas_used":0,"exit_code":-13,"stack":[["num","0x1de82"]]}}`))
	}))
	defer srv.Close()

	c := newHTTPAPIClient(srv.URL, "")
	_, err := c.RunGetMethod(t.Context(), nil, testAddr(0x01), "seqno")

	var execErr tonlib.ContractExecError
	require.ErrorAs(t, err, &execErr)
	require.Equal(t, int32(tonlib.ErrCodeContractNotInitialized), execErr.Code)
}

func newFastRetryClient(url string) *httpAPIClient {
	c := newHTTPAPIClient(url, "")
	c.backoffBase = time.Millisecond

	return c
}

func Test_httpAPIClient_RetriesTransientErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failStatus   int
		failures     int
		wantErr      string
		wantAttempts int32
	}{
		{name: "429 then success", failStatus: http.StatusTooManyRequests, failures: 2, wantAttempts: 3},
		{name: "503 then success", failStatus: http.StatusServiceUnavailable, failures: 4, wantAttempts: 5},
		{name: "persistent 500 gives up", failStatus: http.StatusInternalServerError, failures: 100, wantErr: "status 500", wantAttempts: httpMaxAttempts},
		{name: "400 is not retried", failStatus: http.StatusBadRequest, failures: 100, wantErr: "status 400", wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := attempts.Add(1)
				// the POST body must be re-sent intact on every attempt
				var req runGetMethodRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Method != "getOpCount" {
					w.WriteHeader(http.StatusTeapot)
					return
				}

				if int(n) <= tt.failures {
					w.WriteHeader(tt.failStatus)
					_, _ = w.Write([]byte(`{"ok":false,"error":"Ratelimit exceed","code":` + strconv.Itoa(tt.failStatus) + `}`))

					return
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":{"gas_used":1,"exit_code":0,"stack":[["num","0x7"]]}}`))
			}))
			defer srv.Close()

			c := newFastRetryClient(srv.URL)
			res, err := c.RunGetMethod(t.Context(), nil, testAddr(0x01), "getOpCount")
			require.Equal(t, tt.wantAttempts, attempts.Load())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			n, err := res.Int(0)
			require.NoError(t, err)
			require.Equal(t, int64(7), n.Int64())
		})
	}
}

func Test_httpAPIClient_RetryStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newHTTPAPIClient(srv.URL, "")
	c.backoffBase = time.Hour // the first retry wait outlives the context

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	_, err := c.GetMasterchainInfo(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "status 429") // the underlying cause is kept
	require.Equal(t, int32(1), attempts.Load())
}

func Test_parseRetryAfter(t *testing.T) {
	t.Parallel()

	require.Equal(t, 2*time.Second, parseRetryAfter("2"))
	require.Equal(t, httpRetryAfterMax, parseRetryAfter("3600"))
	require.Zero(t, parseRetryAfter(""))
	require.Zero(t, parseRetryAfter("Wed, 21 Oct 2015 07:28:00 GMT"))
}

func Test_retryDelay(t *testing.T) {
	t.Parallel()

	c := newHTTPAPIClient("http://x", "")
	for attempt := range 10 {
		d := c.retryDelay(attempt, 0)
		want := min(httpBackoffBase<<attempt, httpBackoffMax)
		require.GreaterOrEqual(t, d, want/2)
		require.Less(t, d, want)
	}
	// Retry-After wins when longer
	require.Equal(t, 20*time.Second, c.retryDelay(0, 20*time.Second))
}

func Test_httpAPIClient_SendBocRetriesOnlyRateLimit(t *testing.T) {
	t.Parallel()

	ext := &tlb.ExternalMessage{
		DstAddr: testAddr(0x01),
		Body:    cell.BeginCell().MustStoreUInt(1, 32).EndCell(),
	}

	tests := []struct {
		name         string
		statuses     []int // response status per attempt; 200 afterwards
		wantErr      string
		wantAttempts int32
	}{
		{name: "429 is retried", statuses: []int{429, 429}, wantAttempts: 3},
		{name: "500 is not retried", statuses: []int{500, 500}, wantErr: "status 500", wantAttempts: 1},
		{name: "503 is not retried", statuses: []int{503}, wantErr: "status 503", wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(attempts.Add(1))
				var req struct {
					Boc []byte `json:"boc"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Boc) == 0 || r.URL.Path != "/sendBoc" {
					w.WriteHeader(http.StatusTeapot)
					return
				}
				if n <= len(tt.statuses) {
					w.WriteHeader(tt.statuses[n-1])
					_, _ = w.Write([]byte(`{"ok":false,"error":"boom"}`))

					return
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":{"@type":"ok"}}`))
			}))
			defer srv.Close()

			err := newFastRetryClient(srv.URL).SendExternalMessage(t.Context(), ext)
			require.Equal(t, tt.wantAttempts, attempts.Load())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
