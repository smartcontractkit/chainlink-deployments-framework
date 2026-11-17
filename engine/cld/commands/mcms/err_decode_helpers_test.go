package mcms

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestReadExecutionErrorFromFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name: "success parses execution_error payload",
			input: `{
  "execution_error": {
    "RevertReasonDecoded": "Error(test)",
    "UnderlyingReasonRaw": "0x70de1b4b"
  }
}`,
		},
		{
			name:        "missing execution_error key",
			input:       `{"other_key": {}}`,
			expectError: "must contain an 'execution_error' key",
		},
		{
			name:        "invalid json",
			input:       `{"execution_error":`,
			expectError: "error unmarshaling JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadExecutionErrorFromFile([]byte(tt.input))
			if tt.expectError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectError)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, "Error(test)", got.RevertReasonDecoded)
			require.Equal(t, "0x70de1b4b", got.UnderlyingReasonRaw)
		})
	}
}

// fakeABIRegistry is an EVMABIRegistry that only serves GetAllABIs.
type fakeABIRegistry struct {
	abis map[string]string
}

func (r fakeABIRegistry) GetAllABIs() map[string]string { return r.abis }

func (fakeABIRegistry) GetABIByAddress(uint64, string) (*abi.ABI, string, error) {
	return nil, "", errors.New("not implemented")
}

func (fakeABIRegistry) GetABIByType(cldf.TypeAndVersion) (*abi.ABI, string, error) {
	return nil, "", errors.New("not implemented")
}

func (fakeABIRegistry) AddABI(cldf.TypeAndVersion, string) error {
	return errors.New("not implemented")
}

func TestTryDecodeFromData(t *testing.T) {
	t.Parallel()

	const typeVer = "Router 1.2.0"

	parsed, err := abi.JSON(strings.NewReader(
		`[{"type":"error","name":"InsufficientFunds","inputs":[{"name":"needed","type":"uint256"}]}]`))
	require.NoError(t, err)

	dec, err := NewErrDecoder(fakeABIRegistry{abis: map[string]string{
		typeVer: `[{"type":"error","name":"InsufficientFunds","inputs":[{"name":"needed","type":"uint256"}]}]`,
	}})
	require.NoError(t, err)

	var knownSelector [4]byte
	knownID := parsed.Errors["InsufficientFunds"].ID
	copy(knownSelector[:], knownID[:4])
	knownArgs := common.LeftPadBytes(big.NewInt(7).Bytes(), 32)

	// Error(string) revert data, the way a require with a message reverts.
	stringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	boomArgs, err := abi.Arguments{{Type: stringType}}.Pack("boom")
	require.NoError(t, err)
	var errorStringSelector [4]byte
	copy(errorStringSelector[:], []byte{0x08, 0xc3, 0x79, 0xa0})

	unknownSelector := [4]byte{0xde, 0xad, 0xbe, 0xef}

	tests := []struct {
		name        string
		raw         *evm.CustomErrorData
		decoder     *ErrDecoder
		want        string
		wantDecoded bool
	}{
		{
			name:        "selector and arguments decode to the error name and arguments",
			raw:         &evm.CustomErrorData{Selector: knownSelector, Data: knownArgs},
			decoder:     dec,
			want:        "InsufficientFunds(7) @" + typeVer,
			wantDecoded: true,
		},
		{
			name:        "the first 4 bytes of the arguments are not read as the selector",
			raw:         &evm.CustomErrorData{Selector: unknownSelector, Data: knownArgs},
			decoder:     dec,
			want:        "custom error 0x" + hex.EncodeToString(unknownSelector[:]),
			wantDecoded: true,
		},
		{
			name:        "without a decoder the selector is shown, not the arguments",
			raw:         &evm.CustomErrorData{Selector: knownSelector, Data: knownArgs},
			decoder:     nil,
			want:        "custom error 0x" + hex.EncodeToString(knownSelector[:]),
			wantDecoded: true,
		},
		{
			name:        "Error(string) with the selector set decodes to the message",
			raw:         &evm.CustomErrorData{Selector: errorStringSelector, Data: boomArgs},
			decoder:     dec,
			want:        "boom",
			wantDecoded: true,
		},
		{
			name:        "without a selector the data is the whole revert data",
			raw:         &evm.CustomErrorData{Data: append(errorStringSelector[:], boomArgs...)},
			decoder:     dec,
			want:        "boom",
			wantDecoded: true,
		},
		{
			name:    "without a selector, data shorter than 4 bytes cannot be decoded",
			raw:     &evm.CustomErrorData{Data: []byte{0x01, 0x02}},
			decoder: dec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, decoded := tryDecodeFromData(tt.raw, tt.decoder)

			require.Equal(t, tt.wantDecoded, decoded)
			require.Equal(t, tt.want, got)
		})
	}
}
