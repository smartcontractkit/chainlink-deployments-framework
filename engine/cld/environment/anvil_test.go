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

func Test_isPublicRPC(t *testing.T) {
	t.Parallel()
	tests := []struct {
		url  string
		want bool
	}{
		{"http://rpcs.cldev.sh/", false},
		{"https://rpcs.cldev.sh/", false},
		{"https://rpcs.cldev.sh/anything", false},
		{"https://gap-rpcs.stage.cldev.sh/anything", false},
		{"https://gap-rpcs.prod.cldev.sh/anything", false},
		{"https://gap-other.prod.cldev.sh/anything", false},
		{"https://gap-other.stage.cldev.sh/anything", false},
		{"https://gap-other.stage.cldev.sh/anything", false},
		{"https://gap-grpc-job-distributor.public.main.prod.cldev.sh/", false},
		{"https://gap-ws-job-distributor.public.main.prod.cldev.sh/", false},
		{"https://gap-rpc-proxy.public.main.prod.cldev.sh/", false},
		{"https://gap-grpc-job-distributor.public.main.stage.cldev.sh/", false},
		{"https://gap-ws-job-distributor.public.main.stage.cldev.sh/", false},
		{"https://gap-grpc-chainlink-catalog.public.main.stage.cldev.sh/", false},
		{"https://gap-rpc-proxy.public.main.prod.cldev.sh:4443/ethereum/sepolia/archive", false},
		{"https://gap-rpc-proxy.public.main.prod.cldev.sh:9443/ethereum/sepolia/archive", false},
		{"", true},
		{"http://", true},
		{"https://", true},
		{"https://rpcs.cldev.sh", true},
		{"https://rpcs.prod.cldev.sh/anything", true},
		{"https://rpcs.stage.cldev.sh/anything", true},
		{"https://gap.stage.cldev.sh/anything", true},
		{"https://rpcs-test.tailec123.ts.net/", false},
		{"https://rpcs-test.tail123abc.ts.net/anything", false},
		{"http://rpc-proxy-some-env.tailec123.ts.net/anything", false},
		{"http://tail.publicrpc.ts.net/anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isPublicRPC(tt.url))
		})
	}
}

func Test_selectPublicRPC(t *testing.T) {
	t.Parallel()
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
			setup: nosetup,
			want:  []string{"http://metadata.url"},
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.setup(t)
			urls, err := selectPublicRPC(t.Context(), lggr, tt.metadata, tt.chainSelector, tt.rpcs)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, urls)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
