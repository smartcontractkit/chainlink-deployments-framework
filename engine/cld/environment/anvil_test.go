package environment

import (
	"context"
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

// createAnvilClientForTest creates an anvilClient for testing with the given server URL
func createAnvilClientForTest(serverURL string) *anvilClient {
	return &anvilClient{
		url: serverURL,
		client: resty.New().
			SetTimeout(2 * time.Second).
			SetHeaders(map[string]string{"Content-Type": "application/json"}),
	}
}

func Test_AnvilClient_SendTransaction_Success(t *testing.T) {
	t.Parallel()

	// Create mock server that never fails (empty methodsToFail slice)
	server := createMockServer(t, []string{})
	defer server.Close()

	client := createAnvilClientForTest(server.URL)

	from := "0x1234567890123456789012345678901234567890"
	to := "0x0987654321098765432109876543210987654321"
	data := []byte("test data")

	err := client.SendTransaction(context.Background(), from, to, data)

	require.NoError(t, err)
}

func Test_AnvilClient_SendTransaction_FailOnSetBalance(t *testing.T) {
	t.Parallel()

	// Create mock server that fails on anvil_setBalance method
	server := createMockServer(t, []string{"anvil_setBalance"})
	defer server.Close()

	client := createAnvilClientForTest(server.URL)

	from := "0x1234567890123456789012345678901234567890"
	to := "0x0987654321098765432109876543210987654321"
	data := []byte("test data")

	err := client.SendTransaction(context.Background(), from, to, data)

	// Should fail on the anvil_setBalance method
	require.ErrorContains(t, err, "failed to update balance")
}

func Test_AnvilClient_SendTransaction_FailOnEthSendTransaction(t *testing.T) {
	t.Parallel()

	// Create mock server that fails on eth_sendTransaction method
	server := createMockServer(t, []string{"eth_sendTransaction"})
	defer server.Close()

	client := createAnvilClientForTest(server.URL)

	from := "0x1234567890123456789012345678901234567890"
	to := "0x0987654321098765432109876543210987654321"
	data := []byte("test data")

	err := client.SendTransaction(context.Background(), from, to, data)

	require.ErrorContains(t, err, "failed to send transaction")
}

func Test_AnvilClient_SendTransaction_FailOnMine(t *testing.T) {
	t.Parallel()

	// Create mock server that fails on anvil_mine method
	server := createMockServer(t, []string{"anvil_mine"})
	defer server.Close()

	client := createAnvilClientForTest(server.URL)

	from := "0x1234567890123456789012345678901234567890"
	to := "0x0987654321098765432109876543210987654321"
	data := []byte("test data")

	err := client.SendTransaction(context.Background(), from, to, data)

	require.ErrorContains(t, err, "failed to mine transaction")
}
