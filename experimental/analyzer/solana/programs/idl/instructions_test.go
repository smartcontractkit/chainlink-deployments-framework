package idl

import (
	"bytes"
	"testing"

	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"
)

func TestIsInstruction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "nil",
			data: nil,
			want: false,
		},
		{
			name: "empty",
			data: []byte{},
			want: false,
		},
		{
			name: "discriminator only - 8 bytes",
			data: append([]byte{}, Discriminator...),
			want: false,
		},
		{
			name: "valid - exactly 9 bytes",
			data: append(append([]byte{}, Discriminator...), 0),
			want: true,
		},
		{
			name: "wrong discriminator",
			data: append(make([]byte, 8), 0),
			want: false,
		},
		{
			name: "first byte correct but rest wrong",
			data: append([]byte{Discriminator[0]}, make([]byte, 8)...),
			want: false,
		},
		{
			name: "valid - create payload",
			data: makePayload(InstructionCreate, mustBorshEncode(t, uint64(0))),
			want: true,
		},
		{
			name: "valid - write payload with data",
			data: makePayload(InstructionWrite, mustBorshEncode(t, []byte{1, 2, 3})),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsInstruction(tt.data)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeInstruction(t *testing.T) {
	t.Parallel()

	authority := solana.PublicKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	errorContains := func(errMsg string) func(t *testing.T, inst *Instruction, err error) {
		return func(t *testing.T, inst *Instruction, err error) {
			t.Helper()
			require.ErrorContains(t, err, errMsg)
		}
	}

	tests := []struct {
		name   string
		data   []byte
		assert func(t *testing.T, inst *Instruction, err error)
	}{
		{
			name:   "nil data",
			data:   nil,
			assert: errorContains("not a valid IDL instruction"),
		},
		{
			name:   "empty data",
			data:   []byte{},
			assert: errorContains("not a valid IDL instruction"),
		},
		{
			name:   "wrong discriminator",
			data:   append(make([]byte, 8), 0),
			assert: errorContains("not a valid IDL instruction"),
		},
		{
			name:   "discriminator only - missing type byte",
			data:   append([]byte{}, Discriminator...),
			assert: errorContains("not a valid IDL instruction"),
		},
		{
			name:   "unknown type ID",
			data:   makePayload(255, nil),
			assert: errorContains("unable to decode IDL instruction"),
		},
		{
			name: "create",
			data: makePayload(InstructionCreate, mustBorshEncode(t, uint64(1000))),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionCreate, inst.TypeID.Uint8())
				impl, ok := inst.Impl.(*IDLCreate)
				require.True(t, ok)
				require.Equal(t, uint64(1000), impl.DataLen)
			},
		},
		{
			name: "create_buffer",
			data: makePayload(InstructionCreateBuffer, nil),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionCreateBuffer, inst.TypeID.Uint8())
				_, ok := inst.Impl.(*IDLCreateBuffer)
				require.True(t, ok)
			},
		},
		{
			name: "write",
			data: makePayload(InstructionWrite, mustBorshEncode(t, []byte{0xDE, 0xAD, 0xBE, 0xEF})),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionWrite, inst.TypeID.Uint8())
				impl, ok := inst.Impl.(*IDLWrite)
				require.True(t, ok)
				require.Equal(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}, impl.Data)
			},
		},
		{
			name: "set_buffer",
			data: makePayload(InstructionSetBuffer, nil),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionSetBuffer, inst.TypeID.Uint8())
				_, ok := inst.Impl.(*IDLSetBuffer)
				require.True(t, ok)
			},
		},
		{
			name: "set_authority",
			data: makePayload(InstructionSetAuthority, mustBorshEncode(t, authority)),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionSetAuthority, inst.TypeID.Uint8())
				impl, ok := inst.Impl.(*IDLSetAuthority)
				require.True(t, ok)
				require.Equal(t, authority, impl.NewAuthority)
			},
		},
		{
			name: "close",
			data: makePayload(InstructionClose, nil),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionClose, inst.TypeID.Uint8())
				_, ok := inst.Impl.(*IDLClose)
				require.True(t, ok)
			},
		},
		{
			name: "resize",
			data: makePayload(InstructionResize, mustBorshEncode(t, uint64(512))),
			assert: func(t *testing.T, inst *Instruction, _ error) {
				t.Helper()
				require.Equal(t, InstructionResize, inst.TypeID.Uint8())
				impl, ok := inst.Impl.(*IDLResize)
				require.True(t, ok)
				require.Equal(t, uint64(512), impl.DataLen)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, err := DecodeInstruction(nil, tt.data)
			tt.assert(t, inst, err)
		})
	}
}

func TestInstruction_ProgramID(t *testing.T) {
	t.Parallel()
	inst := &Instruction{}
	require.Equal(t, ProgramID, inst.ProgramID())
}

// TestInstruction_Accounts verifies that Accounts() returns nil for all IDL instruction types, since
// none of them embed an AccountMetaSlice or implement AccountsGettable.
func TestInstruction_Accounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "create",
			payload: makePayload(InstructionCreate, mustBorshEncode(t, uint64(0))),
		},
		{
			name:    "create_buffer",
			payload: makePayload(InstructionCreateBuffer, nil),
		},
		{
			name:    "write",
			payload: makePayload(InstructionWrite, mustBorshEncode(t, []byte{1})),
		},
		{
			name:    "set_buffer",
			payload: makePayload(InstructionSetBuffer, nil),
		},
		{
			name:    "set_authority",
			payload: makePayload(InstructionSetAuthority, mustBorshEncode(t, solana.PublicKey{})),
		},
		{
			name:    "close",
			payload: makePayload(InstructionClose, nil),
		},
		{
			name:    "resize",
			payload: makePayload(InstructionResize, mustBorshEncode(t, uint64(0))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, err := DecodeInstruction(nil, tt.payload)
			require.NoError(t, err)
			require.Nil(t, inst.Accounts())
		})
	}
}

// TestInstruction_Data verifies that Data() re-encodes an instruction to bytes that are identical
// to the portion of the original payload that follows the discriminator (i.e. the round-trip is lossless).
func TestInstruction_Data(t *testing.T) {
	t.Parallel()

	authority := solana.PublicKey{31: 0xFF}

	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "create",
			payload: makePayload(InstructionCreate, mustBorshEncode(t, uint64(42))),
		},
		{
			name:    "create_buffer",
			payload: makePayload(InstructionCreateBuffer, nil),
		},
		{
			name:    "write",
			payload: makePayload(InstructionWrite, mustBorshEncode(t, []byte{0xCA, 0xFE})),
		},
		{
			name:    "set_buffer",
			payload: makePayload(InstructionSetBuffer, nil),
		},
		{
			name:    "set_authority",
			payload: makePayload(InstructionSetAuthority, mustBorshEncode(t, authority)),
		},
		{
			name:    "close",
			payload: makePayload(InstructionClose, nil),
		},
		{
			name:    "resize",
			payload: makePayload(InstructionResize, mustBorshEncode(t, uint64(8192))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst, err := DecodeInstruction(nil, tt.payload)
			require.NoError(t, err)

			got, err := inst.Data()
			require.NoError(t, err)
			require.Equal(t, tt.payload[len(Discriminator):], got)
		})
	}
}

func TestIDLCreate_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    uint64
		wantErr string
	}{
		{
			name:  "zero",
			input: mustBorshEncode(t, uint64(0)),
			want:  0,
		},
		{
			name:  "small",
			input: mustBorshEncode(t, uint64(100)),
			want:  100,
		},
		{
			name:  "large",
			input: mustBorshEncode(t, uint64(1<<32)),
			want:  1 << 32,
		},
		{
			name:  "max_uint64",
			input: mustBorshEncode(t, ^uint64(0)),
			want:  ^uint64(0),
		},
		{
			name:    "truncated - 4 bytes instead of 8",
			input:   []byte{0, 0, 0, 0},
			wantErr: "decode: uint64 required [8] bytes, remaining [4]",
		},
		{
			name:    "empty",
			input:   []byte{},
			wantErr: "decode: uint64 required [8] bytes, remaining [0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inst := &IDLCreate{}
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, inst.DataLen)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestIDLResize_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    uint64
		wantErr string
	}{
		{
			name:  "zero",
			input: mustBorshEncode(t, uint64(0)),
			want:  0,
		},
		{
			name:  "typical_10kb",
			input: mustBorshEncode(t, uint64(10240)),
			want:  10240,
		},
		{
			name:  "large",
			input: mustBorshEncode(t, uint64(1<<20)),
			want:  1 << 20,
		},
		{
			name:    "truncated - 4 bytes instead of 8",
			input:   []byte{0, 0, 0, 0},
			wantErr: "decode: uint64 required [8] bytes, remaining [4]",
		},
		{
			name:    "empty",
			input:   []byte{},
			wantErr: "decode: uint64 required [8] bytes, remaining [0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inst := new(IDLResize)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, inst.DataLen)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestIDLWrite_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr string
	}{
		{
			name:  "empty_data",
			input: mustBorshEncode(t, []byte{}),
			want:  nil,
		},
		{
			name:  "single_byte",
			input: mustBorshEncode(t, []byte{0xFF}),
			want:  []byte{0xFF},
		},
		{
			name:  "multi_byte",
			input: mustBorshEncode(t, []byte{0x01, 0x02, 0x03, 0x04}),
			want:  []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:  "binary_data",
			input: mustBorshEncode(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}),
			want:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
		},
		{
			name:    "truncated - no length prefix",
			input:   []byte{},
			wantErr: "uint32 required [4] bytes, remaining [0]",
		},
		{
			name:    "truncated - length claims more bytes than available",
			input:   []byte{0x05, 0x00, 0x00, 0x00, 0x01},
			wantErr: "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inst := new(IDLWrite)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, inst.Data)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestIDLSetAuthority_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	key1 := solana.MustPublicKeyFromBase58("11111111111111111111111111111111") // all-zero key

	tests := []struct {
		name    string
		input   []byte
		want    solana.PublicKey
		wantErr string
	}{
		{
			name:  "zero_key",
			input: mustBorshEncode(t, solana.PublicKey{}),
			want:  solana.PublicKey{},
		},
		{
			name:  "non_zero_key",
			input: mustBorshEncode(t, key1),
			want:  key1,
		},
		{
			name:    "truncated - 16 bytes instead of 32",
			input:   make([]byte, 16),
			wantErr: "unexpected EOF",
		},
		{
			name:    "empty",
			input:   []byte{},
			wantErr: "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inst := new(IDLSetAuthority)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, inst.NewAuthority)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestIDLClose_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty_input",
			input: []byte{},
		},
		{
			name:  "extra_bytes_ignored",
			input: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst := new(IDLClose)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))
			require.NoError(t, err)
		})
	}
}

func TestIDLCreateBuffer_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty_input",
			input: []byte{},
		},
		{
			name:  "extra_bytes_ignored",
			input: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst := new(IDLCreateBuffer)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))
			require.NoError(t, err)
		})
	}
}

func TestIDLSetBuffer_UnmarshalWithDecoder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty_input",
			input: []byte{},
		},
		{
			name:  "extra_bytes_ignored",
			input: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inst := new(IDLSetBuffer)
			err := inst.UnmarshalWithDecoder(binary.NewBorshDecoder(tt.input))
			require.NoError(t, err)
		})
	}
}

// ---- helpers -----

func makePayload(typeID uint8, implBytes []byte) []byte {
	out := make([]byte, 0, len(Discriminator)+1+len(implBytes))
	out = append(out, Discriminator...)
	out = append(out, typeID)
	out = append(out, implBytes...)

	return out
}

func mustBorshEncode(t *testing.T, v any) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	err := binary.NewBorshEncoder(buf).Encode(v)
	require.NoError(t, err)

	return buf.Bytes()
}
