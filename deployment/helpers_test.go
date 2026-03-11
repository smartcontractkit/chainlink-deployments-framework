package deployment

import (
	"encoding/hex"
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

func TestParseErrorFromABI_CallRevertedWithErrorString(t *testing.T) {
	t.Parallel()

	// This is the exact hex from the bug report: CallReverted(bytes) wrapping
	// Error("AccessControl: account 0xa5d5b0b844c8f11b61f28ac98bba84dea9b80953
	// is missing role 0xb09aa5aeb3702cfd50b6b62bc4532604938f21248a27a1d5ca736082b6819cc1")
	revertHex := "0x70de1b4b" +
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

	result, err := parseErrorFromABI(revertHex, callRevertedABI)
	require.NoError(t, err)
	assert.Contains(t, result, "CallReverted")
	assert.Contains(t, result, "AccessControl")
	assert.Contains(t, result, "0xa5d5b0b844c8f11b61f28ac98bba84dea9b80953")
	assert.Contains(t, result, "is missing role")

	// The result should NOT contain raw byte arrays like "[8 195 121 160 ...]"
	assert.NotContains(t, result, "[8 195")
}

func TestParseErrorFromABI_CallRevertedWithNestedCustomError(t *testing.T) {
	t.Parallel()

	// Build CallReverted(bytes) wrapping InvalidConfig(uint64) from sampleABI.
	// InvalidConfig selector = keccak256("InvalidConfig(uint64)")[:4]
	parsedABI, err := abi.JSON(strings.NewReader(sampleABI))
	require.NoError(t, err)

	invalidConfigErr := parsedABI.Errors["InvalidConfig"]
	innerData, err := invalidConfigErr.Inputs.Pack(uint64(42))
	require.NoError(t, err)
	innerPayload := append(invalidConfigErr.ID[:4], innerData...)

	// Wrap in CallReverted(bytes)
	callRevertedParsed, err := abi.JSON(strings.NewReader(callRevertedABI))
	require.NoError(t, err)
	callRevertedErr := callRevertedParsed.Errors["CallReverted"]
	outerData, err := callRevertedErr.Inputs.Pack(innerPayload)
	require.NoError(t, err)
	fullData := append(callRevertedErr.ID[:4], outerData...)

	// sampleABI doesn't have CallReverted, so use a combined ABI
	combinedABI := `[{"inputs":[{"type":"bytes"}],"name":"CallReverted","type":"error"},` +
		`{"inputs":[{"type":"uint64","name":"configId"}],"name":"InvalidConfig","type":"error"}]`

	result, err := parseErrorFromABI("0x"+hex.EncodeToString(fullData), combinedABI)
	require.NoError(t, err)
	assert.Contains(t, result, "CallReverted")
	assert.Contains(t, result, "InvalidConfig")
}

func TestParseErrorFromABI_CallRevertedWithUnknownInnerError(t *testing.T) {
	t.Parallel()

	// Build CallReverted(bytes) wrapping unknown data (not Error(string), not in ABI)
	unknownInner := make([]byte, 36)
	copy(unknownInner[:4], []byte{0xde, 0xad, 0xbe, 0xef})

	callRevertedParsed, err := abi.JSON(strings.NewReader(callRevertedABI))
	require.NoError(t, err)
	callRevertedErr := callRevertedParsed.Errors["CallReverted"]
	outerData, err := callRevertedErr.Inputs.Pack(unknownInner)
	require.NoError(t, err)
	fullData := append(callRevertedErr.ID[:4], outerData...)

	result, err := parseErrorFromABI("0x"+hex.EncodeToString(fullData), callRevertedABI)
	require.NoError(t, err)
	assert.Contains(t, result, "CallReverted")
	// Should fall back to hex since the inner error is unrecognized
	assert.Contains(t, result, "0xdeadbeef")
}
