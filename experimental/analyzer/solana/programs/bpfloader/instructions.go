package bpfloader

import (
	"bytes"
	"encoding/binary"
	"fmt"

	ag_spew "github.com/davecgh/go-spew/spew"
	ag_binary "github.com/gagliardetto/binary"
	ag_solanago "github.com/gagliardetto/solana-go"
	ag_text "github.com/gagliardetto/solana-go/text"
	ag_treeout "github.com/gagliardetto/treeout"
)

var ProgramID ag_solanago.PublicKey = ag_solanago.BPFLoaderUpgradeableProgramID

func SetProgramID(pubkey ag_solanago.PublicKey) error {
	ProgramID = pubkey
	ag_solanago.RegisterInstructionDecoder(ProgramID, registryDecodeInstruction)

	return nil
}

const ProgramName = "BPFLoaderUpgradeable"

func init() {
	ag_solanago.RegisterInstructionDecoder(ProgramID, registryDecodeInstruction)
}

const (
	Instruction_InitializeBuffer uint32 = iota
	Instruction_Write
	Instruction_DeployWithMaxDataLen
	Instruction_Upgrade
	Instruction_SetAuthority
	Instruction_Close
	Instruction_ExtendProgram
)

// InstructionIDToName returns the name of the instruction given its ID.
func InstructionIDToName(id uint32) string {
	switch id {
	case Instruction_InitializeBuffer:
		return "InitializeBuffer"
	case Instruction_Write:
		return "Write"
	case Instruction_DeployWithMaxDataLen:
		return "DeployWithMaxDataLen"
	case Instruction_Upgrade:
		return "Upgrade"
	case Instruction_SetAuthority:
		return "SetAuthority"
	case Instruction_Close:
		return "Close"
	case Instruction_ExtendProgram:
		return "ExtendProgram"
	default:
		return ""
	}
}

type Instruction struct {
	ag_binary.BaseVariant
}

func (inst *Instruction) EncodeToTree(parent ag_treeout.Branches) {
	if enToTree, ok := inst.Impl.(ag_text.EncodableToTree); ok {
		enToTree.EncodeToTree(parent)
	} else {
		parent.Child(ag_spew.Sdump(inst))
	}
}

type (
	InitializeBuffer     struct{}
	Write                struct{}
	DeployWithMaxDataLen struct{}
	Upgrade              struct{}
	SetAuthority         struct{}
	Close                struct{}
	ExtendProgram        struct{}
)

var InstructionImplDef = ag_binary.NewVariantDefinition(
	ag_binary.Uint32TypeIDEncoding,
	[]ag_binary.VariantType{
		{Name: "InitializeBuffer", Type: (*InitializeBuffer)(nil)},
		{Name: "Write", Type: (*Write)(nil)},
		{Name: "DeployWithMaxDataLen", Type: (*DeployWithMaxDataLen)(nil)},
		{Name: "Upgrade", Type: (*Upgrade)(nil)},
		{Name: "SetAuthority", Type: (*SetAuthority)(nil)},
		{Name: "Close", Type: (*Close)(nil)},
		{Name: "ExtendProgram", Type: (*ExtendProgram)(nil)},
	},
)

func (inst *Instruction) ProgramID() ag_solanago.PublicKey {
	return ProgramID
}

func (inst *Instruction) Accounts() (out []*ag_solanago.AccountMeta) {
	return inst.Impl.(ag_solanago.AccountsGettable).GetAccounts()
}

func (inst *Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := ag_binary.NewBinEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}

	return buf.Bytes(), nil
}

func (inst *Instruction) TextEncode(encoder *ag_text.Encoder, option *ag_text.Option) error {
	return encoder.Encode(inst.Impl, option)
}

func (inst *Instruction) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return inst.UnmarshalBinaryVariant(decoder, InstructionImplDef)
}

func (inst Instruction) MarshalWithEncoder(encoder *ag_binary.Encoder) error {
	err := encoder.WriteUint32(inst.TypeID.Uint32(), binary.LittleEndian)
	if err != nil {
		return fmt.Errorf("unable to write variant type: %w", err)
	}

	return encoder.Encode(inst.Impl)
}

func registryDecodeInstruction(accounts []*ag_solanago.AccountMeta, data []byte) (any, error) {
	inst, err := DecodeInstruction(accounts, data)
	if err != nil {
		return nil, err
	}

	return inst, nil
}

func DecodeInstruction(accounts []*ag_solanago.AccountMeta, data []byte) (*Instruction, error) {
	inst := new(Instruction)
	if err := ag_binary.NewBinDecoder(data).Decode(inst); err != nil {
		return nil, fmt.Errorf("unable to decode instruction: %w", err)
	}
	if v, ok := inst.Impl.(ag_solanago.AccountsSettable); ok {
		err := v.SetAccounts(accounts)
		if err != nil {
			return nil, fmt.Errorf("unable to set accounts for instruction: %w", err)
		}
	}

	return inst, nil
}
