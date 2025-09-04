package environment

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
