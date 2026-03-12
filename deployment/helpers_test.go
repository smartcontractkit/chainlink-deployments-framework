package deployment

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleABI = "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transmissionId\",\"type\":\"bytes32\"}],\"name\":\"AlreadyAttempted\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"DuplicateSigner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"numSigners\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSigners\",\"type\":\"uint256\"}],\"name\":\"ExcessSigners\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FaultToleranceMustBePositive\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transmissionId\",\"type\":\"bytes32\"}],\"name\":\"InsufficientGasForRouting\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"numSigners\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minSigners\",\"type\":\"uint256\"}],\"name\":\"InsufficientSigners\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"configId\",\"type\":\"uint64\"}],\"name\":\"InvalidConfig\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidReport\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"InvalidSignatureCount\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"InvalidSigner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnauthorizedForwarder\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"donId\",\"type\":\"uint32\"},{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"configVersion\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"f\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address[]\",\"name\":\"signers\",\"type\":\"address[]\"}],\"name\":\"ConfigSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"}],\"name\":\"ForwarderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"}],\"name\":\"ForwarderRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"OwnershipTransferRequested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"workflowExecutionId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes2\",\"name\":\"reportId\",\"type\":\"bytes2\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"}],\"name\":\"ReportProcessed\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"}],\"name\":\"addForwarder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"donId\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"configVersion\",\"type\":\"uint32\"}],\"name\":\"clearConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"workflowExecutionId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes2\",\"name\":\"reportId\",\"type\":\"bytes2\"}],\"name\":\"getTransmissionId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"workflowExecutionId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes2\",\"name\":\"reportId\",\"type\":\"bytes2\"}],\"name\":\"getTransmissionInfo\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"transmissionId\",\"type\":\"bytes32\"},{\"internalType\":\"enumIRouter.TransmissionState\",\"name\":\"state\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"transmitter\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"invalidReceiver\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"uint80\",\"name\":\"gasLimit\",\"type\":\"uint80\"}],\"internalType\":\"structIRouter.TransmissionInfo\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"workflowExecutionId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes2\",\"name\":\"reportId\",\"type\":\"bytes2\"}],\"name\":\"getTransmitter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"}],\"name\":\"isForwarder\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"}],\"name\":\"removeForwarder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"rawReport\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"reportContext\",\"type\":\"bytes\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"}],\"name\":\"report\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transmissionId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"transmitter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"validatedReport\",\"type\":\"bytes\"}],\"name\":\"route\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"donId\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"configVersion\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"f\",\"type\":\"uint8\"},{\"internalType\":\"address[]\",\"name\":\"signers\",\"type\":\"address[]\"}],\"name\":\"setConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"typeAndVersion\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// callRevertedABI is a minimal ABI containing only the CallReverted(bytes) error,
// matching the MCM contract's error signature.
const callRevertedABI = `[{"inputs":[{"internalType":"bytes","name":"","type":"bytes"}],"name":"CallReverted","type":"error"}]`

func TestIsPanicRevert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "Panic(uint256) selector",
			data: append([]byte{0x4e, 0x48, 0x7b, 0x71}, make([]byte, 32)...),
			want: true,
		},
		{
			name: "Error(string) selector",
			data: []byte{0x08, 0xc3, 0x79, 0xa0, 0x00},
			want: false,
		},
		{
			name: "too short",
			data: []byte{0x4e, 0x48, 0x7b},
			want: false,
		},
		{
			name: "nil",
			data: nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsPanicRevert(tc.data))
		})
	}
}

func TestFormatUnpackedRevert(t *testing.T) {
	t.Parallel()

	panicData := append([]byte{0x4e, 0x48, 0x7b, 0x71}, make([]byte, 32)...)
	errorData := append([]byte{0x08, 0xc3, 0x79, 0xa0}, make([]byte, 32)...)

	assert.Equal(t, `Panic("division or modulo by zero")`, FormatUnpackedRevert(panicData, "division or modulo by zero"))
	assert.Equal(t, `Error("Only callable by owner")`, FormatUnpackedRevert(errorData, "Only callable by owner"))
}

// buildPanicPayload builds a valid Panic(uint256) ABI-encoded payload.
func buildPanicPayload(t *testing.T, code uint64) []byte {
	t.Helper()
	typ, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	packed, err := abi.Arguments{{Type: typ}}.Pack(new(big.Int).SetUint64(code))
	require.NoError(t, err)

	return append(panicSelector, packed...)
}

func TestParseErrorFromABI(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name               string
		RevertReason       string
		ABI                string
		ParsedRevertReason string
		ExpectError        bool
	}{
		{
			Name:               "Generic error with string msg",
			RevertReason:       "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000164f6e6c792063616c6c61626c65206279206f776e657200000000000000000000",
			ABI:                "", // ABI is not required for this case
			ParsedRevertReason: "error - `Only callable by owner`",
		},
		{
			Name:               "Custom typed error",
			RevertReason:       "0xdf3b81ea0000000000000000000000000000000000000000000000000000000100000001",
			ABI:                sampleABI,
			ParsedRevertReason: "error -`InvalidConfig` args [4294967297]",
		},
		{
			Name:               "Panic division by zero",
			RevertReason:       "0x4e487b710000000000000000000000000000000000000000000000000000000000000012",
			ABI:                "",
			ParsedRevertReason: "panic - `division or modulo by zero`",
		},
		{
			Name:               "Panic assertion failure",
			RevertReason:       "0x4e487b710000000000000000000000000000000000000000000000000000000000000001",
			ABI:                "",
			ParsedRevertReason: "panic - `assert(false)`",
		},
		{
			Name:               "Empty error string",
			RevertReason:       "",
			ABI:                sampleABI,
			ParsedRevertReason: "",
			ExpectError:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			revertReason, err := parseErrorFromABI(tc.RevertReason, tc.ABI)
			if tc.ExpectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.ParsedRevertReason, revertReason)
			}
		})
	}
}

// wrapInCallReverted ABI-encodes innerPayload inside a CallReverted(bytes) error.
func wrapInCallReverted(t *testing.T, innerPayload []byte) []byte {
	t.Helper()
	callRevertedParsed, err := abi.JSON(strings.NewReader(callRevertedABI))
	require.NoError(t, err)
	callRevertedErr := callRevertedParsed.Errors["CallReverted"]
	outerData, err := callRevertedErr.Inputs.Pack(innerPayload)
	require.NoError(t, err)

	return append(callRevertedErr.ID[:4], outerData...)
}

func TestParseErrorFromABI_CallReverted(t *testing.T) {
	t.Parallel()

	combinedABI := `[{"inputs":[{"type":"bytes"}],"name":"CallReverted","type":"error"},` +
		`{"inputs":[{"type":"uint64","name":"configId"}],"name":"InvalidConfig","type":"error"}]`

	tests := []struct {
		name            string
		buildHex        func(t *testing.T) string
		contractABI     string
		wantExact       string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "wrapping Error(string)",
			buildHex: func(_ *testing.T) string {
				return "0x70de1b4b" +
					"0000000000000000000000000000000000000000000000000000000000000020" +
					"00000000000000000000000000000000000000000000000000000000000000e4" +
					"08c379a0" +
					"0000000000000000000000000000000000000000000000000000000000000020" +
					"0000000000000000000000000000000000000000000000000000000000000094" +
					"416363657373436f6e74726f6c3a206163636f756e742030786135643562306238" +
					"3434633866313162363166323861633938626261383464656139623830393533" +
					"206973206d697373696e6720726f6c6520307862303961613561656233373032" +
					"6366643530623662363262633435333236303439333866323132343861323761" +
					"3164356361373336303832623638313963633100000000000000000000000000" +
					"0000000000000000000000000000000000000000000000000000000000000000"
			},
			contractABI:     callRevertedABI,
			wantContains:    []string{"CallReverted", "AccessControl", "0xa5d5b0b844c8f11b61f28ac98bba84dea9b80953", "is missing role"},
			wantNotContains: []string{"[8 195"},
		},
		{
			name: "wrapping nested custom error",
			buildHex: func(t *testing.T) string {
				parsedABI, err := abi.JSON(strings.NewReader(sampleABI))
				require.NoError(t, err)
				invalidConfigErr := parsedABI.Errors["InvalidConfig"]
				innerData, err := invalidConfigErr.Inputs.Pack(uint64(42))
				require.NoError(t, err)
				innerPayload := append(invalidConfigErr.ID[:4], innerData...)

				return "0x" + hex.EncodeToString(wrapInCallReverted(t, innerPayload))
			},
			contractABI: combinedABI,
			wantExact:   "error -`CallReverted` args [error -`InvalidConfig` args [42]]",
		},
		{
			name: "wrapping Panic(uint256)",
			buildHex: func(t *testing.T) string {
				return "0x" + hex.EncodeToString(wrapInCallReverted(t, buildPanicPayload(t, 0x12)))
			},
			contractABI: callRevertedABI,
			wantExact:   "error -`CallReverted` args [Panic(\"division or modulo by zero\")]",
		},
		{
			name: "wrapping unknown inner error",
			buildHex: func(t *testing.T) string {
				unknownInner := make([]byte, 36)
				copy(unknownInner[:4], []byte{0xde, 0xad, 0xbe, 0xef})
				return "0x" + hex.EncodeToString(wrapInCallReverted(t, unknownInner))
			},
			contractABI:  callRevertedABI,
			wantContains: []string{"CallReverted", "0xdeadbeef"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseErrorFromABI(tc.buildHex(t), tc.contractABI)
			require.NoError(t, err)

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, result)
			}
			for _, s := range tc.wantContains {
				assert.Contains(t, result, s)
			}
			for _, s := range tc.wantNotContains {
				assert.NotContains(t, result, s)
			}
		})
	}
}
