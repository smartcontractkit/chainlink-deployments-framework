package mcmsv2

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	mcmsbindings "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	timelockbindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/timelock"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cldfds "github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// --- helpers ---

func mustType(t *testing.T, typ string) abi.Type {
	t.Helper()
	ty, err := abi.NewType(typ, "", nil)
	require.NoError(t, err)

	return ty
}

func errorSelector(name string, args abi.Arguments) []byte {
	ts := make([]string, len(args))
	for i, a := range args {
		ts[i] = a.Type.String()
	}
	sig := fmt.Sprintf("%s(%s)", name, strings.Join(ts, ","))
	h := crypto.Keccak256([]byte(sig))

	return h[:4]
}

func buildCustomErrorRevert(t *testing.T, name string, args abi.Arguments, vals ...interface{}) []byte {
	t.Helper()
	enc, err := args.Pack(vals...)
	require.NoError(t, err)

	return append(errorSelector(name, args), enc...)
}

func buildStdErrorRevert(t *testing.T, msg string) []byte {
	t.Helper()
	stringTy := mustType(t, "string")
	args := abi.Arguments{{Type: stringTy}}
	enc, err := args.Pack(msg)
	require.NoError(t, err)
	sel, _ := hex.DecodeString("08c379a0")

	return append(sel, enc...)
}

type strDataError struct{ s string }

func (e strDataError) Error() string          { return "x" }
func (e strDataError) ErrorData() interface{} { return e.s }

type bytesDataError struct{ b []byte }

func (e bytesDataError) Error() string          { return "x" }
func (e bytesDataError) ErrorData() interface{} { return e.b }

type hexBytesError struct{ b hexutil.Bytes }

func (e hexBytesError) Error() string          { return "x" }
func (e hexBytesError) ErrorData() interface{} { return e.b }

type mapDataError struct{ data map[string]interface{} }

func (e mapDataError) Error() string          { return "map data err" }
func (e mapDataError) ErrorData() interface{} { return e.data }

// fake client used by tryDecodeTxRevertEVM (matches the CallContract signature)
type fakeCallClient struct{ err error }

func (f fakeCallClient) CallContract(ctx context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return nil, f.err
}

// --- tests ---

func Test_decodeWithABI(t *testing.T) {
	t.Parallel()

	addrTy := mustType(t, "address")
	u256Ty := mustType(t, "uint256")
	errArgs := abi.Arguments{
		{Name: "caller", Type: addrTy},
		{Name: "needLevel", Type: u256Ty},
	}
	abiJSON := `[
		{"type":"error","name":"Unauthorized","inputs":[
			{"name":"caller","type":"address"},
			{"name":"needLevel","type":"uint256"}]}
	]`

	caller := ethcommon.HexToAddress("0x000000000000000000000000000000000000bEEF")
	wantLvl := big.NewInt(3)
	customRevert := buildCustomErrorRevert(t, "Unauthorized", errArgs, caller, wantLvl)

	t.Run("decodeWithABI finds the custom error", func(t *testing.T) {
		t.Parallel()

		got, ok := decodeWithABI(abiJSON, customRevert)
		require.True(t, ok)
		assert.Contains(t, got, caller.Hex())
		assert.Contains(t, got, wantLvl.String())
	})

	t.Run("ErrDecoder falls back to registry", func(t *testing.T) {
		t.Parallel()
		ds := cldfds.NewMemoryDataStore()
		reg, err := analyzer.NewEnvironmentEVMRegistry(cldf.Environment{
			ExistingAddresses: cldf.NewMemoryAddressBook(),
			DataStore:         ds.Seal(),
		}, map[string]string{
			"Foo@1.0.0": abiJSON,
		})
		require.NoError(t, err)
		dec, err := NewErrDecoder(reg)
		require.NoError(t, err)
		got, ok := dec.decodeRecursive(customRevert, "")
		require.True(t, ok)
		assert.Contains(t, got, "@Foo@1.0.0")
	})

	t.Run("ErrDecoder prefers provided ABI", func(t *testing.T) {
		t.Parallel()
		ds := cldfds.NewMemoryDataStore()
		reg, err := analyzer.NewEnvironmentEVMRegistry(cldf.Environment{
			ExistingAddresses: cldf.NewMemoryAddressBook(),
			DataStore:         ds.Seal(),
		}, map[string]string{
			"Foo@1.0.0": abiJSON,
		})
		require.NoError(t, err)
		dec, err := NewErrDecoder(reg)
		require.NoError(t, err)
		got, ok := dec.decodeRecursive(customRevert, abiJSON)
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(got, "Unauthorized("))
		assert.NotContains(t, got, "@")
	})
}

func Test_prettyRevertFromError_StdError(t *testing.T) {
	t.Parallel()

	// Build standard Error(string) revert payload
	std := buildStdErrorRevert(t, "boom")
	err := hexBytesError{hexutil.Bytes(std)}
	ds := cldfds.NewMemoryDataStore()

	reg, errReg := analyzer.NewEnvironmentEVMRegistry(cldf.Environment{
		ExistingAddresses: cldf.NewMemoryAddressBook(),
		DataStore:         ds.Seal(),
	}, nil)
	require.NoError(t, errReg)
	dec, derr := NewErrDecoder(reg) // empty registry ok
	require.NoError(t, derr)

	out, ok := prettyRevertFromError(err, "", dec)
	require.True(t, ok)
	assert.Equal(t, "boom", out)
}

func Test_prettyRevertFromError_Recursive(t *testing.T) {
	t.Parallel()

	// Inner error: Unauthorized(address)
	addrTy := mustType(t, "address")
	innerArgs := abi.Arguments{{Name: "who", Type: addrTy}}
	who := ethcommon.HexToAddress("0x0000000000000000000000000000000000000123")
	inner := buildCustomErrorRevert(t, "Unauthorized", innerArgs, who)

	// Outer wrapper: CallReverted(bytes)
	bytesTy := mustType(t, "bytes")
	outerArgs := abi.Arguments{{Name: "data", Type: bytesTy}}
	outer := buildCustomErrorRevert(t, "CallReverted", outerArgs, inner)

	// Registry with both ABIs (JSON form) so the decoder can match selectors
	const innerABIJSON = `[
	  {"type":"error","name":"Unauthorized","inputs":[{"name":"who","type":"address"}]}
	]`
	const wrapperABIJSON = `[
	  {"type":"error","name":"CallReverted","inputs":[{"name":"data","type":"bytes"}]}
	]`

	ds := cldfds.NewMemoryDataStore()
	reg, err := analyzer.NewEnvironmentEVMRegistry(cldf.Environment{
		ExistingAddresses: cldf.NewMemoryAddressBook(),
		DataStore:         ds.Seal(),
	}, map[string]string{
		"Inner@1":   innerABIJSON,
		"Wrapper@1": wrapperABIJSON,
	})
	require.NoError(t, err)
	dec, derr := NewErrDecoder(reg)
	require.NoError(t, derr)

	err = hexBytesError{hexutil.Bytes(outer)}
	out, ok := prettyRevertFromError(err, "", dec)
	require.True(t, ok)
	// Expect both outer and inner to appear due to recursive unwrap
	assert.Contains(t, out, "CallReverted(")
	assert.Contains(t, out, "Unauthorized(")
}

func Test_tryDecodeTxRevertEVM(t *testing.T) {
	t.Parallel()

	// The simulated call will "revert" with Error("boom")
	std := buildStdErrorRevert(t, "boom")
	cli := fakeCallClient{err: hexBytesError{hexutil.Bytes(std)}}

	// Minimal transaction (unsigned is okay; sender recovery best-effort)
	to := ethcommon.HexToAddress("0x0000000000000000000000000000000000000aAa")
	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{
		ChainID:   big.NewInt(56),
		To:        &to,
		Gas:       21000,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Value:     big.NewInt(0),
		Data:      []byte{0x01, 0x02},
	})
	ds := cldfds.NewMemoryDataStore()
	env := cldf.Environment{DataStore: ds.Seal(), ExistingAddresses: cldf.NewMemoryAddressBook()}
	proposalCtx, err := analyzer.NewDefaultProposalContext(
		env,
		analyzer.WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0": mcmsbindings.RBACTimelockABI,
		}),
		analyzer.WithSolanaDecoders(map[string]analyzer.DecodeInstructionFn{
			"RBACTimelockProgram 1.0.0": analyzer.DIFn(timelockbindings.DecodeInstruction),
		}),
	)
	require.NoError(t, err)
	out, ok := tryDecodeTxRevertEVM(context.Background(), cli, tx, "", nil, proposalCtx)
	require.True(t, ok)
	assert.Equal(t, "boom", out)
}

func Test_parseEVMValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantStr string
		ok      bool
	}{
		{"empty additionalFields", "", "0", true},
		{"explicit 0", `{"value":0}`, "0", true},
		{"large value", `{"value":12345678901234567890}`, "12345678901234567890", true},
		{"invalid json", `{"value":`, "", false},
		{"negative rejected", `{"value":-1}`, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var raw json.RawMessage
			if tt.raw != "" {
				raw = json.RawMessage(tt.raw)
			}
			got, err := parseEVMValue(raw)
			if !tt.ok {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStr, got.String())
		})
	}
}

func Test_extractRevertData(t *testing.T) {
	t.Parallel()

	std := buildStdErrorRevert(t, "boom")
	hexStd := "0x" + hex.EncodeToString(std)

	tests := []struct {
		name     string
		err      error
		expected []byte // expected revert data
		ok       bool   // expected ok status
	}{
		// --- Success Cases ---
		{"nil error", nil, nil, false},
		{"string hex with 0x prefix", strDataError{hexStd}, std, true},
		{"[]byte", bytesDataError{std}, std, true},
		{"hexutil.Bytes", hexBytesError{hexutil.Bytes(std)}, std, true},
		{"map with data key", mapDataError{map[string]interface{}{"data": hexStd}}, std, true},
		{"wrapped error", fmt.Errorf("layer 2: %w", fmt.Errorf("layer 1: %w", strDataError{hexStd})), std, true},

		// --- Failure Cases ---
		{"plain error without data", errors.New("no data"), nil, false},
		{"string without 0x prefix", strDataError{"deadbeef"}, nil, false},
		{"invalid hex string", strDataError{"0xnot-hex"}, nil, false},
		{"map with reason key", mapDataError{map[string]interface{}{"reason": hexStd}}, std, true},
		{"map with wrong value type", mapDataError{map[string]interface{}{"data": 12345}}, nil, false},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := extractRevertData(tt.err)
			if !tt.ok {
				assert.False(t, ok, "expected ok to be false but it was true")
				return
			}
			require.True(t, ok, "expected ok to be true but it was false")
			assert.Equal(t, tt.expected, got, "extracted revert data does not match expected data")
		})
	}
}

func Test_DiagnoseTimelockRevert(t *testing.T) {
	t.Parallel()

	std1 := buildStdErrorRevert(t, "first revert")
	std2 := buildStdErrorRevert(t, "second revert")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type rpcReq struct {
			JSONRPC string            `json:"jsonrpc"`
			Method  string            `json:"method"`
			Params  []json.RawMessage `json:"params"`
			ID      json.RawMessage   `json:"id"`
		}
		var req rpcReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      nil,
				"error":   map[string]any{"code": -32700, "message": "parse error"},
			})

			return
		}

		writeResult := func(result any) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  result,
			})
		}
		writeError := func(code int, msg string, data any) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error": map[string]any{
					"code":    code,
					"message": msg,
					"data":    data,
				},
			})
		}

		switch req.Method {
		case "anvil_impersonateAccount", "anvil_stopImpersonatingAccount":
			writeResult(true)
			return

		case "eth_call":
			var callObj map[string]any
			if len(req.Params) == 0 {
				writeResult("0x")
				return
			}
			_ = json.Unmarshal(req.Params[0], &callObj)

			dataHex, _ := callObj["data"].(string)
			if dataHex == "" {
				dataHex, _ = callObj["input"].(string)
			}
			dataHex = strings.TrimPrefix(dataHex, "0x")
			b, _ := hex.DecodeString(dataHex)

			if len(b) > 0 {
				switch b[0] {
				case 0x01:
					writeError(-32000, "execution reverted", "0x"+hex.EncodeToString(std1))
					return
				case 0x02:
					writeError(-32000, "execution reverted", "0x"+hex.EncodeToString(std2))
					return
				}
			}
			writeResult("0x")

			return

		default:
			writeError(-32601, "method not found", nil)
			return
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	selector := uint64(101)
	timelock := ethcommon.HexToAddress("0xAAAA00000000000000000000000000000000AAAA")
	bops := []types.BatchOperation{
		{
			ChainSelector: types.ChainSelector(selector),
			Transactions: []types.Transaction{
				{To: "0x9999999999999999999999999999999999999999", Data: []byte{0x01}, AdditionalFields: json.RawMessage(`{}`)},
				{To: "0x9999999999999999999999999999999999999999", Data: []byte{0x02}, AdditionalFields: json.RawMessage(`{"value":0}`)},
			},
		},
	}

	lggr := logger.Test(t)
	var ab cldf.AddressBook = nil // unused; we trigger the standard Error(string) path

	ds := cldfds.NewMemoryDataStore()
	env := cldf.Environment{DataStore: ds.Seal(), ExistingAddresses: cldf.NewMemoryAddressBook()}
	proposalCtx, err := analyzer.NewDefaultProposalContext(
		env,
		analyzer.WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0": mcmsbindings.RBACTimelockABI,
		}),
		analyzer.WithSolanaDecoders(map[string]analyzer.DecodeInstructionFn{
			"RBACTimelockProgram 1.0.0": analyzer.DIFn(timelockbindings.DecodeInstruction),
		}),
	)
	require.NoError(t, err)
	err = diagnoseTimelockRevert(t.Context(), lggr, srv.URL, selector, bops, timelock, ab, proposalCtx)
	require.Error(t, err)

	es := err.Error()
	assert.Contains(t, es, "timelock diagnosis found issues:")
	assert.Contains(t, es, "first revert")
	assert.Contains(t, es, "second revert")
}
