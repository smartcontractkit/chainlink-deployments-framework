package environment

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

// createMockServer creates a mock HTTP server that fails on specific JSON-RPC methods
// methodsToFail: slice of method names that should fail (e.g., ["anvil_setBalance", "eth_sendTransaction"])
// Returns the server
func createMockServer(t *testing.T, methodsToFail []string) *httptest.Server {
	t.Helper()

	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse the JSON-RPC request
		var rpcReq JSONRPCRequest
		if err = json.Unmarshal(body, &rpcReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check if this method should fail
		for _, failMethod := range methodsToFail {
			if rpcReq.Method == failMethod {
				// Force connection error for this method
				server.CloseClientConnections()
				return
			}
		}

		// Other calls succeed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x123"}`))
		assert.NoError(t, err)
	}))

	return server
}

// createMockAnvilClient creates an anvilClient for testing with the given server URL
func createMockAnvilClient(serverURL string) *anvilClient {
	return &anvilClient{
		url: serverURL,
		client: resty.New().
			SetTimeout(2 * time.Second).
			SetHeaders(map[string]string{"Content-Type": "application/json"}),
	}
}

func Test_AnvilClient_SendTransaction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		methodsToFail []string
		expectedError string
	}{
		{
			name:          "Success",
			methodsToFail: []string{},
			expectedError: "",
		},
		{
			name:          "FailOnSetBalance",
			methodsToFail: []string{"anvil_setBalance"},
			expectedError: "failed to update balance",
		},
		{
			name:          "FailOnEthSendTransaction",
			methodsToFail: []string{"eth_sendTransaction"},
			expectedError: "failed to send transaction",
		},
		{
			name:          "FailOnMine",
			methodsToFail: []string{"anvil_mine"},
			expectedError: "failed to mine transaction",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := createMockServer(t, tc.methodsToFail)
			defer server.Close()

			client := createMockAnvilClient(server.URL)

			from := "0x1234567890123456789012345678901234567890"
			to := "0x0987654321098765432109876543210987654321"
			data := []byte("test data")

			err := client.SendTransaction(t.Context(), from, to, data)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}

func Test_selectRPCs(t *testing.T) { //nolint:paralleltest
	httpmock.Activate(t)

	lggr := logger.Test(t)
	nosetup := func(t *testing.T) { t.Helper() }

	tests := []struct {
		name          string
		metadata      *cfgnet.EVMMetadata
		chainSelector uint64
		rpcs          []cfgnet.RPC
		setup         func(t *testing.T)
		want          []string
		wantErr       string
	}{
		{
			name: "success: metadata has url",
			metadata: &cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{
				ArchiveHTTPURL: "http://metadata.url",
			}},
			rpcs: []cfgnet.RPC{
				{HTTPURL: "http://other.url"},
			},
			setup: func(t *testing.T) {
				t.Helper()
				httpmock.RegisterResponder("POST", "http://metadata.url",
					httpmock.NewStringResponder(200, `{"jsonrpc":"2.0","id":1,"result":"0x123"}`))
			},
			want: []string{"http://metadata.url"},
		},
		{
			name: "success: archive_http_urls are included alongside archive_http_url",
			metadata: &cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{
				ArchiveHTTPURL:  "http://metadata.url",
				ArchiveHTTPURLs: []string{"http://archive1.url", "http://archive2.url"},
			}},
			setup: func(t *testing.T) {
				t.Helper()
				httpmock.RegisterResponder("POST", "http://metadata.url",
					httpmock.NewStringResponder(200, `{"jsonrpc":"2.0","id":1,"result":"0x123"}`))
				httpmock.RegisterResponder("POST", "http://archive1.url",
					httpmock.NewStringResponder(200, `{"jsonrpc":"2.0","id":1,"result":"0x123"}`))
				// archive2.url is intentionally left unregistered so its health check fails.
			},
			want: []string{"http://metadata.url", "http://archive1.url"},
		},
		{
			name: "success: selects only health public rpcs",
			metadata: &cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{
				ArchiveHTTPURL: "http://gap-rpc.prod.cldev.sh/ethereum/sepolia",
			}},
			rpcs: []cfgnet.RPC{
				{HTTPURL: "http://rpcs.cldev.sh/ethereum/sepolia"},
				{HTTPURL: "http://public.rpc1.url"},
				{HTTPURL: "http://public.rpc2.url"},
				{HTTPURL: "http://public.rpc3.url"},
			},
			setup: func(t *testing.T) {
				t.Helper()
				httpmock.RegisterResponder("POST", "http://public.rpc1.url",
					httpmock.NewStringResponder(200, `{"jsonrpc":"2.0","id":1,"result":"0x123"}`))
				httpmock.RegisterResponder("POST", "http://public.rpc3.url",
					httpmock.NewStringResponder(200, `{"jsonrpc":"2.0","id":1,"result":"0x456"}`))
			},
			want: []string{"http://public.rpc1.url", "http://public.rpc3.url"},
		},
		{
			name: "failure: no public or healthy rpcs found",
			metadata: &cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{
				ArchiveHTTPURL: "http://gap-rpc.prod.cldev.sh/ethereum/sepolia",
			}},
			rpcs: []cfgnet.RPC{
				{HTTPURL: "http://rpcs.cldev.sh/ethereum/sepolia"},
				{HTTPURL: "http://unhealthy.rpc.url"},
			},
			setup:   nosetup,
			wantErr: "no public RPCs found for chain 0",
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			urls, err := selectRPCs(t.Context(), lggr, hostRPCHealthChecker{}, tt.metadata, tt.chainSelector, tt.rpcs)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, urls)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
