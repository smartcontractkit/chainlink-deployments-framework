package provider

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/assert"
)

// newFakeRPCServer returns a fake RPC server which always answers with a valid `eth_blockNumber“
// response.
//
// When the test is done, the server is closed automatically.
func newFakeRPCServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a valid eth_blockNumber response
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	})

	srv := httptest.NewServer(handler)

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

// alwaysFailingTransactorGenerator returns a TransactorGenerator that always fails
// with an error.
type alwaysFailingTransactorGenerator struct{}

// Generate implements the TransactorGenerator interface and always returns an error.
func (a *alwaysFailingTransactorGenerator) Generate(chainID *big.Int) (*bind.TransactOpts, error) {
	return nil, assert.AnError
}
