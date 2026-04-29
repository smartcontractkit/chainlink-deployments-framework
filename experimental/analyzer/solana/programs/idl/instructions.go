package idl

import (
	"bytes"
	"errors"
	"fmt"

	ag_binary "github.com/gagliardetto/binary"
	ag_solanago "github.com/gagliardetto/solana-go"
)

const ProgramName = "AnchorIDL"

// ProgramID is a zero sentinel since IDL instructions are sent to the target program
// rather than a dedicated IDL program address.
var ProgramID ag_solanago.PublicKey

// Discriminator is the fixed 8-byte Anchor discriminator that identifies IDL instructions.
var Discriminator = []byte{64, 244, 188, 120, 167, 233, 105, 10}

const (
	InstructionCreate       uint8 = iota // One-time initializer for creating the program's IDL account.
	InstructionCreateBuffer              // Creates a new IDL account buffer. Can be called several times.
	InstructionWrite                     // Appends the given data to the end of the IDL account buffer.
	InstructionSetBuffer                 // Sets a new data buffer for the IdlAccount.
	InstructionSetAuthority              // Sets a new authority on the IdlAccount.
	InstructionClose                     // Closes the IDL PDA account.
	InstructionResize                    // Increases account size for accounts that need over 10kb.
)

var InstructionImplDef = ag_binary.NewVariantDefinition(
	ag_binary.Uint8TypeIDEncoding,
	[]ag_binary.VariantType{
		{Name: "create", Type: (*IDLCreate)(nil)},
		{Name: "create_buffer", Type: (*IDLCreateBuffer)(nil)},
		{Name: "write", Type: (*IDLWrite)(nil)},
		{Name: "set_buffer", Type: (*IDLSetBuffer)(nil)},
		{Name: "set_authority", Type: (*IDLSetAuthority)(nil)},
		{Name: "close", Type: (*IDLClose)(nil)},
		{Name: "resize", Type: (*IDLResize)(nil)},
	},
)

type Instruction struct {
	ag_binary.BaseVariant
}

func (inst *Instruction) ProgramID() ag_solanago.PublicKey {
	return ProgramID
}

func (inst *Instruction) Accounts() (out []*ag_solanago.AccountMeta) {
	if gettable, ok := inst.Impl.(ag_solanago.AccountsGettable); ok {
		return gettable.GetAccounts()
	}

	return nil
}

func (inst *Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := ag_binary.NewBorshEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode IDL instruction: %w", err)
	}

	return buf.Bytes(), nil
}

func (inst *Instruction) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return inst.UnmarshalBinaryVariant(decoder, InstructionImplDef)
}

func (inst *Instruction) MarshalWithEncoder(encoder *ag_binary.Encoder) error {
	if err := encoder.WriteUint8(inst.TypeID.Uint8()); err != nil {
		return fmt.Errorf("unable to write variant type: %w", err)
	}

	return encoder.Encode(inst.Impl)
}

// IsInstruction reports whether data starts with the IDL instruction discriminator.
func IsInstruction(data []byte) bool {
	return len(data) >= 9 && bytes.Equal(data[:8], Discriminator)
}

// DecodeInstruction decodes a Solana IDL instruction from the given accounts and data.
// The data must start with the 8-byte IDL discriminator followed by a 1-byte instruction type.
func DecodeInstruction(accounts []*ag_solanago.AccountMeta, data []byte) (*Instruction, error) {
	if !IsInstruction(data) {
		return nil, errors.New("not a valid IDL instruction: invalid discriminator or insufficient data length")
	}
	inst := new(Instruction)
	if err := ag_binary.NewBorshDecoder(data[8:]).Decode(inst); err != nil {
		return nil, fmt.Errorf("unable to decode IDL instruction: %w", err)
	}
	if v, ok := inst.Impl.(ag_solanago.AccountsSettable); ok {
		if err := v.SetAccounts(accounts); err != nil {
			return nil, fmt.Errorf("unable to set accounts for IDL instruction: %w", err)
		}
	}

	return inst, nil
}
