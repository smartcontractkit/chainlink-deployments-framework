package provider

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_getErrorReasonFromTx(t *testing.T) {
	t.Parallel()

	var (
		tx = types.NewTransaction(
			1,                               // nonce
			common.HexToAddress("0xabc123"), // to address
			big.NewInt(1000000000000000000), // value: 1 ETH
			21000,                           // gas limit
			big.NewInt(20000000000),         // gas price: 20 Gwei
			[]byte{0xde, 0xad, 0xbe, 0xef},  // data
		)
	)

	tests := []struct {
		name        string
		beforeFunc  func(caller *MockContractCaller)
		giveTx      *types.Transaction
		giveReceipt *types.Receipt
		wantReason  string
		wantErr     string
	}{
		{
			name: "no transaction error",
			beforeFunc: func(caller *MockContractCaller) {
				caller.EXPECT().CallContract(
					mock.Anything,
					mock.AnythingOfType("ethereum.CallMsg"),
					mock.AnythingOfType("*big.Int"),
				).Return([]byte{}, nil)
			},
			giveTx:      tx,
			giveReceipt: &types.Receipt{},
			wantErr:     "reverted with no reason",
		},
		{
			name: "transaction error with reason",
			beforeFunc: func(caller *MockContractCaller) {
				caller.EXPECT().CallContract(
					mock.Anything,
					mock.AnythingOfType("ethereum.CallMsg"),
					mock.AnythingOfType("*big.Int"),
				).Return(nil, &jsonError{
					Code:    100,
					Message: "test error message",
					Data:    []byte("test error data"),
				})
			},
			giveTx:      tx,
			giveReceipt: &types.Receipt{},
			wantReason:  "test error data",
		},
		{
			name: "transaction error with no reason (non json error)",
			beforeFunc: func(caller *MockContractCaller) {
				caller.EXPECT().CallContract(
					mock.Anything,
					mock.AnythingOfType("ethereum.CallMsg"),
					mock.AnythingOfType("*big.Int"),
				).Return(nil, errors.New("error message"))
			},
			giveTx:      tx,
			giveReceipt: &types.Receipt{},
			wantReason:  "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			caller := NewMockContractCaller(t)
			if tt.beforeFunc != nil {
				tt.beforeFunc(caller)
			}

			got, err := getErrorReasonFromTx(
				t.Context(), caller, common.HexToAddress("0x123"), tt.giveTx, tt.giveReceipt,
			)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantReason, got)
			}
		})
	}
}

func Test_parseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    error
		want    string
		wantErr string
	}{
		{
			name: "valid error",
			give: &jsonError{
				Code:    100,
				Message: "execution reverted",
				Data:    "0x12345678",
			},
			want: "0x12345678",
		},
		{
			name:    "nil error",
			give:    nil,
			wantErr: "cannot parse nil error",
		},
		{
			name:    "invalid error type",
			give:    errors.New("invalid"),
			wantErr: "error must be of type jsonError",
		},
		{
			name: "trie error",
			give: &jsonError{
				Code:    -32000,
				Message: "missing trie node",
				Data:    []byte{},
			},
			wantErr: "missing trie node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getJSONErrorData(tt.give)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Dummy implementation of jsonError to satisfy the interface
type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}

	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}

func (err *jsonError) ErrorData() interface{} {
	return err.Data
}
